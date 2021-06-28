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

func doCreateCluster(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig) error {
	var loadbalancer *api.HostConfig
	var controlPlane *api.HostConfig
	var joinNodes []*api.HostConfig
	var joinNodeIDs []string
	var etcdNodes []string
	// Step1: setup infrastructure for all nodes in the cluster
	for _, n := range cc.Nodes {
		if err := handler.MachineInfraSetup(n); err != nil {
			return err
		}
		if utils.IsType(n.Type, api.LoadBalance) {
			loadbalancer = n
		}
		if utils.IsType(n.Type, api.ETCD) {
			etcdNodes = append(etcdNodes, n.Address)
		}
		if utils.IsType(n.Type, api.Worker) {
			joinNodes = append(joinNodes, n)
			joinNodeIDs = append(joinNodeIDs, n.Address)
		}

		if utils.IsType(n.Type, api.Master) {
			if controlPlane == nil {
				controlPlane = n
			} else {
				joinNodes = append(joinNodes, n)
				joinNodeIDs = append(joinNodeIDs, n.Address)
			}
		}
	}

	// Step2: setup etcd cluster
	// wait infrastructure task success on nodes of etcd cluster
	if err := nodemanager.WaitNodesFinish(etcdNodes, time.Minute*5); err != nil {
		return err
	}
	if err := handler.EtcdClusterSetup(); err != nil {
		return err
	}
	// Step3: setup loadbalance for cluster
	if err := handler.LoadBalancerSetup(loadbalancer); err != nil {
		return err
	}
	// Step4: setup control plane for cluster
	if err := handler.ClusterControlPlaneInit(controlPlane); err != nil {
		return err
	}
	// wait controlplane setup task success
	if err := nodemanager.WaitNodesFinish([]string{controlPlane.Address}, time.Minute*5); err != nil {
		return err
	}

	//Step5: setup left nodes for cluster
	for _, node := range joinNodes {
		if err := handler.ClusterNodeJoin(node); err != nil {
			return err
		}
	}
	//Step5: setup addons for cluster
	// wait all nodes ready
	if err := nodemanager.WaitNodesFinish(joinNodeIDs, time.Minute*5); err != nil {
		return err
	}

	return handler.AddonsSetup()
}

func CreateCluster(cc *api.ClusterConfig) error {
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

	// prepare eggo config directory
	if err := os.MkdirAll(api.GetClusterHomePath(cc.Name), 0750); err != nil {
		return err
	}

	if err := doCreateCluster(handler, cc); err != nil {
		return err
	}

	logrus.Infof("[cluster] create cluster '%s' successed", cc.Name)
	return nil
}

func doRemoveCluster(handler api.ClusterDeploymentAPI, cc *api.ClusterConfig) {
	// Step1: cleanup addons
	err := handler.AddonsDestroy()
	if err != nil {
		logrus.Warnf("[cluster] cleanup addons failed: %v", err)
	}

	//Step2: cleanup loadbalance
	for _, n := range cc.Nodes {
		if utils.IsType(n.Type, api.LoadBalance) {
			err = handler.LoadBalancerDestroy(n)
			if err != nil {
				logrus.Warnf("[cluster] cleanup loadbalance failed: %v", err)
			}
			break
		}
	}

	// Step3: cleanup works
	for _, n := range cc.Nodes {
		if utils.IsType(n.Type, api.Worker) {
			err = handler.ClusterNodeCleanup(n)
			if err != nil {
				logrus.Warnf("[cluster] cleanup node: %s failed: %v", n.Name, err)
			}
		}
	}

	// Step4: cleanup masters
	for _, n := range cc.Nodes {
		if utils.IsType(n.Type, api.Master) {
			err = handler.ClusterNodeCleanup(n)
			if err != nil {
				logrus.Warnf("[cluster] cleanup master: %s failed: %v", n.Name, err)
			}
		}
	}
	// Step5: cleanup etcd cluster
	err = handler.EtcdClusterDestroy()
	if err != nil {
		logrus.Warnf("[cluster] cleanup etcd cluster failed: %v", err)
	}
	// Step6: cleanup infrastructure
	for _, n := range cc.Nodes {
		err = handler.MachineInfraDestroy(n)
		if err != nil {
			logrus.Warnf("[cluster] cleanup infrastructure for node: %s failed: %v", n.Name, err)
		}
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
