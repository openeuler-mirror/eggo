/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2021. All rights reserved.
 * eggo licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: haozi007
 * Create: 2021-05-11
 * Description: cluster deploy
 ******************************************************************************/

package clusterdeployment

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	_ "isula.org/eggo/pkg/clusterdeployment/binary"
	"isula.org/eggo/pkg/clusterdeployment/manager"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
)

func splitNodes(nodes []*api.HostConfig) (*api.HostConfig, []*api.HostConfig, []*api.HostConfig, []string) {
	var lb *api.HostConfig
	var masters []*api.HostConfig
	var workers []*api.HostConfig
	var etcdNodes []string

	for _, n := range nodes {
		if utils.IsType(n.Type, api.LoadBalance) {
			lb = n
		}
		if utils.IsType(n.Type, api.ETCD) {
			etcdNodes = append(etcdNodes, n.Address)
		}

		if utils.IsType(n.Type, api.Master) {
			masters = append(masters, n)
			// node with master and worker, just put into masters
			continue
		}

		if utils.IsType(n.Type, api.Worker) {
			workers = append(workers, n)
		}
	}

	return lb, masters, workers, etcdNodes
}

func doJoinNodeOfCluster(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig, masters, workers []*api.HostConfig) ([]string, []*api.HostConfig, error) {
	var joinedNodeIDs []string
	var failedNodes []*api.HostConfig
	for _, node := range workers {
		if err := handler.ClusterNodeJoin(node); err != nil {
			failedNodes = append(failedNodes, node)
			continue
		}
		joinedNodeIDs = append(joinedNodeIDs, node.Address)
	}
	for _, node := range masters {
		if err := handler.ClusterNodeJoin(node); err != nil {
			failedNodes = append(failedNodes, node)
			continue
		}
		joinedNodeIDs = append(joinedNodeIDs, node.Address)
	}
	// wait all nodes ready
	if err := nodemanager.WaitNodesFinishWithProgress(joinedNodeIDs, time.Minute*5); err != nil {
		tFailedNodes, successNodes := nodemanager.CheckNodesStatus(joinedNodeIDs)
		// update joined and failed nodes
		failedNodes = append(failedNodes, tFailedNodes...)
		joinedNodeIDs = successNodes
		if len(successNodes) == 0 {
			return joinedNodeIDs, nil, err
		}
		logrus.Warnf("wait some node to complete join failed: %v", err)
	}

	return joinedNodeIDs, failedNodes, nil
}

func doCreateCluster(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig, cstatus *api.ClusterStatus) ([]*api.HostConfig, error) {
	loadbalancer, masters, workers, etcdNodes := splitNodes(cc.Nodes)

	if len(masters) == 0 {
		return nil, fmt.Errorf("no master found")
	}
	controlPlaneNode, err := masters[0].DeepCopy()
	if err != nil {
		return nil, err
	}
	cstatus.ControlPlane = controlPlaneNode.Address
	masters = masters[1:]

	// Step1: setup infrastructure for all nodes in the cluster
	for _, n := range cc.Nodes {
		if err = handler.MachineInfraSetup(n); err != nil {
			return nil, err
		}
	}

	// Step2: run precreate cluster hooks
	if err = handler.PreCreateClusterHooks(); err != nil {
		return nil, err
	}

	// Step3: setup etcd cluster
	// wait infrastructure task success on nodes of etcd cluster
	if err = nodemanager.WaitNodesFinishWithProgress(etcdNodes, time.Minute*5); err != nil {
		return nil, err
	}
	if err = handler.EtcdClusterSetup(); err != nil {
		return nil, err
	}

	// Step4: setup loadbalance for cluster
	if err = handler.LoadBalancerSetup(loadbalancer); err != nil {
		return nil, err
	}

	// Step5: setup control plane for cluster
	if err = handler.ClusterControlPlaneInit(controlPlaneNode); err != nil {
		return nil, err
	}
	// wait controlplane setup task success
	if err = nodemanager.WaitNodesFinish([]string{controlPlaneNode.Address}, time.Minute*5); err != nil {
		return nil, err
	}
	if utils.IsType(controlPlaneNode.Type, api.Worker) {
		controlPlaneNode.Type = utils.ClearType(controlPlaneNode.Type, api.Master)
		if err = handler.ClusterNodeJoin(controlPlaneNode); err != nil {
			return nil, err
		}
	}

	//Step6: setup left nodes for cluster
	joinedNodeIDs, failedNodes, err := doJoinNodeOfCluster(handler, cc, masters, workers)
	if err != nil {
		return nil, err
	}

	//Step7: setup addons for cluster
	if err = handler.AddonsSetup(); err != nil {
		return nil, err
	}

	// Step8: run postcreate cluster hooks
	if err = handler.PostCreateClusterHooks(); err != nil {
		return nil, err
	}

	if err = nodemanager.WaitNodesFinishWithProgress(append(joinedNodeIDs, controlPlaneNode.Address), time.Minute*5); err != nil {
		return nil, err
	}

	for _, sid := range joinedNodeIDs {
		cstatus.StatusOfNodes[sid] = true
		cstatus.SuccessCnt += 1
	}
	cstatus.Working = true

	return failedNodes, nil
}

func rollbackFailedNoeds(handler api.ClusterDeploymentAPI, nodes []*api.HostConfig) {
	if nodes == nil {
		return
	}
	var rollIDs []string
	for _, n := range nodes {
		// do best to cleanup, if error, just ignore
		handler.ClusterNodeCleanup(n, n.Type)
		handler.MachineInfraDestroy(n)
		rollIDs = append(rollIDs, n.Address)
	}

	if err := nodemanager.WaitNodesFinishWithProgress(rollIDs, time.Minute*5); err != nil {
		logrus.Warnf("rollback failed: %v", err)
	}
}

func CreateCluster(cc *api.ClusterConfig) (api.ClusterStatus, error) {
	cstatus := api.ClusterStatus{
		StatusOfNodes: make(map[string]bool),
	}
	if cc == nil {
		return cstatus, fmt.Errorf("[cluster] cluster config is required")
	}

	creator, err := manager.GetClusterDeploymentDriver(cc.DeployDriver)
	if err != nil {
		logrus.Errorf("[cluster] get cluster deployment driver: %s failed: %v", cc.DeployDriver, err)
		return cstatus, err
	}
	handler, err := creator(cc)
	if err != nil {
		logrus.Errorf("[cluster] create cluster deployment instance with driver: %s, failed: %v", cc.DeployDriver, err)
		return cstatus, err
	}
	defer handler.Finish()

	// prepare eggo config directory
	if err := os.MkdirAll(api.GetClusterHomePath(cc.Name), 0750); err != nil {
		return cstatus, err
	}

	failedNodes, err := doCreateCluster(handler, cc, &cstatus)
	if err != nil {
		logrus.Warnf("rollback cluster: %s", cc.Name)
		doRemoveCluster(handler, cc)
		cstatus.Message = err.Error()
		return cstatus, err
	}
	// rollback failed nodes
	rollbackFailedNoeds(handler, failedNodes)
	// update status of cluster
	if failedNodes != nil {
		var failureIDs []string
		for _, fid := range failedNodes {
			failureIDs = append(failureIDs, fid.Address)
			cstatus.StatusOfNodes[fid.Address] = false
			cstatus.FailureCnt += 1
		}
		logrus.Warnf("[cluster] failed nodes: %v", failureIDs)
		cstatus.Message = "partial success of create cluster"
		return cstatus, nil
	}

	cstatus.Message = "create cluster success"
	return cstatus, nil
}

func doJoinNode(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if err := handler.MachineInfraSetup(hostconfig); err != nil {
		return err
	}

	// Pre node join Hooks
	if err := handler.PreNodeJoinHooks(hostconfig); err != nil {
		return err
	}

	// wait infrastructure task success on node
	if err := nodemanager.WaitNodesFinish([]string{hostconfig.Address}, time.Minute*5); err != nil {
		return err
	}

	// join node to cluster
	if err := handler.ClusterNodeJoin(hostconfig); err != nil {
		return err
	}

	// Post node join Hooks
	if err := handler.PostNodeJoinHooks(hostconfig); err != nil {
		return err
	}

	// wait node ready
	if err := nodemanager.WaitNodesFinishWithProgress([]string{hostconfig.Address}, time.Minute*5); err != nil {
		return err
	}

	return nil
}

func JoinNode(cc *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if cc == nil {
		return fmt.Errorf("[cluster] cluster config is required")
	}

	creator, err := manager.GetClusterDeploymentDriver(cc.DeployDriver)
	if err != nil {
		logrus.Errorf("[cluster] get cluster deployment driver: %s failed: %v", cc.DeployDriver, err)
		return err
	}
	handler, err := creator(cc)
	if err != nil {
		logrus.Errorf("[cluster] create cluster deployment instance with driver: %s, failed: %v", cc.DeployDriver, err)
		return err
	}
	defer handler.Finish()

	if err := doJoinNode(handler, cc, hostconfig); err != nil {
		logrus.Errorf("join node failed: %v", err)
		if err1 := doDeleteNode(handler, cc, hostconfig); err1 != nil {
			logrus.Errorf("try cleanup node failed when join node failed: %v", err1)
		}
	}

	logrus.Infof("[cluster] join '%s' to cluster successed", cc.Name)
	return nil
}

func doDeleteNode(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig, h *api.HostConfig) error {
	// Pre node delete Hooks
	handler.PreNodeCleanupHooks(h)

	if utils.IsType(h.Type, api.Worker) {
		if err := handler.ClusterNodeCleanup(h, api.Worker); err != nil {
			return fmt.Errorf("delete worker %s failed: %v", h.Name, err)
		}
	}

	if utils.IsType(h.Type, api.Master) {
		if err := handler.ClusterNodeCleanup(h, api.Master); err != nil {
			return fmt.Errorf("delete master %s failed: %v", h.Name, err)
		}
	}

	if utils.IsType(h.Type, api.ETCD) {
		if err := handler.EtcdNodeDestroy(h); err != nil {
			return fmt.Errorf("delete etcd of node %s failed: %v", h.Name, err)
		}
	}

	// Post node delete Hooks
	handler.PostNodeCleanupHooks(h)

	if err := handler.MachineInfraDestroy(h); err != nil {
		logrus.Warnf("cleanup infrastructure for node: %s failed: %v", h.Name, err)
		return err
	}

	if err := nodemanager.WaitNodesFinishWithProgress([]string{h.Address}, time.Minute*5); err != nil {
		logrus.Warnf("wait cleanup finish failed: %v", err)
	}

	return nil
}

func DeleteNode(cc *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if cc == nil {
		return fmt.Errorf("[cluster] cluster config is required")
	}

	creator, err := manager.GetClusterDeploymentDriver(cc.DeployDriver)
	if err != nil {
		logrus.Errorf("[cluster] get cluster deployment driver: %s failed: %v", cc.DeployDriver, err)
		return err
	}
	handler, err := creator(cc)
	if err != nil {
		logrus.Errorf("[cluster] create cluster deployment instance with driver: %s, failed: %v", cc.DeployDriver, err)
		return err
	}
	defer handler.Finish()

	if err := doDeleteNode(handler, cc, hostconfig); err != nil {
		return err
	}

	logrus.Infof("[cluster] delete node '%s' from cluster successed", cc.Name)
	return nil
}

func doRemoveCluster(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig) {
	// Step1: Pre delete cluster Hooks
	handler.PreDeleteClusterHooks()

	// Step2: cleanup addons
	err := handler.AddonsDestroy()
	if err != nil {
		logrus.Warnf("[cluster] cleanup addons failed: %v", err)
	}

	allNodes := utils.GetAllIPs(cc.Nodes)
	if err = nodemanager.WaitNodesFinish(allNodes, time.Minute*5); err != nil {
		logrus.Warnf("[cluster] wait cleanup addons failed: %v", err)
	}

	// Step3: cleanup workers
	for _, n := range cc.Nodes {
		if utils.IsType(n.Type, api.Worker) {
			err = handler.ClusterNodeCleanup(n, api.Worker)
			if err != nil {
				logrus.Warnf("[cluster] cleanup node: %s failed: %v", n.Name, err)
			}
		}
	}

	// Step4: cleanup masters
	for _, n := range cc.Nodes {
		if utils.IsType(n.Type, api.Master) {
			err = handler.ClusterNodeCleanup(n, api.Master)
			if err != nil {
				logrus.Warnf("[cluster] cleanup master: %s failed: %v", n.Name, err)
			}
		}
	}

	//Step5: cleanup loadbalance
	for _, n := range cc.Nodes {
		if utils.IsType(n.Type, api.LoadBalance) {
			err = handler.LoadBalancerDestroy(n)
			if err != nil {
				logrus.Warnf("[cluster] cleanup loadbalance failed: %v", err)
			}
			break
		}
	}

	// Step6: cleanup etcd cluster
	err = handler.EtcdClusterDestroy()
	if err != nil {
		logrus.Warnf("[cluster] cleanup etcd cluster failed: %v", err)
	}

	// Step7: Post delete cluster Hooks
	handler.PostDeleteClusterHooks()

	// Step8: cleanup infrastructure
	for _, n := range cc.Nodes {
		err = handler.MachineInfraDestroy(n)
		if err != nil {
			logrus.Warnf("[cluster] cleanup infrastructure for node: %s failed: %v", n.Name, err)
		}
	}

	if err = nodemanager.WaitNodesFinishWithProgress(allNodes, time.Minute*5); err != nil {
		logrus.Warnf("[cluster] wait all cleanup finish failed: %v", err)
	}
}

func RemoveCluster(cc *api.ClusterConfig) error {
	if cc == nil {
		return fmt.Errorf("cluster config is required")
	}
	creator, err := manager.GetClusterDeploymentDriver(cc.DeployDriver)
	if err != nil {
		logrus.Errorf("[cluster] get cluster deployment driver: %s failed: %v", cc.DeployDriver, err)
		return err
	}
	handler, err := creator(cc)
	if err != nil {
		logrus.Errorf("[cluster] remove cluster deployment instance with driver: %s, failed: %v", cc.DeployDriver, err)
		return err
	}
	defer handler.Finish()

	// cleanup cluster
	doRemoveCluster(handler, cc)

	// cleanup eggo config directory
	if err := os.RemoveAll(api.GetClusterHomePath(cc.Name)); err != nil {
		logrus.Warnf("[cluster] cleanup eggo config directory failed: %v", err)
		return nil
	}
	logrus.Infof("[cluster] remove cluster '%s' successed", cc.Name)
	return nil
}
