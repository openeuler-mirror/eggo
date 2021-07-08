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
 * Description: eggo binary implement
 ******************************************************************************/
package binary

import (
	"fmt"
	"sync"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/addons"
	"isula.org/eggo/pkg/clusterdeployment/binary/bootstrap"
	"isula.org/eggo/pkg/clusterdeployment/binary/cleanupcluster"
	"isula.org/eggo/pkg/clusterdeployment/binary/controlplane"
	"isula.org/eggo/pkg/clusterdeployment/binary/coredns"
	"isula.org/eggo/pkg/clusterdeployment/binary/etcdcluster"
	"isula.org/eggo/pkg/clusterdeployment/binary/infrastructure"
	"isula.org/eggo/pkg/clusterdeployment/binary/loadbalance"
	"isula.org/eggo/pkg/clusterdeployment/binary/network"
	"isula.org/eggo/pkg/clusterdeployment/manager"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/kubectl"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"

	"github.com/sirupsen/logrus"
)

const (
	name = "binary"
)

func init() {
	if err := manager.RegisterClusterDeploymentDriver(name, New); err != nil {
		logrus.Fatal(err)
	}
	logrus.Debug("register binary")
}

func New(conf *api.ClusterConfig) (api.ClusterDeploymentAPI, error) {
	bcd := &BinaryClusterDeployment{
		config:      conf,
		connections: make(map[string]runner.Runner),
	}
	// register and connect all nodes
	bcd.registerNodes()

	return bcd, nil
}

type BinaryClusterDeployment struct {
	config *api.ClusterConfig

	connLock    sync.RWMutex
	connections map[string]runner.Runner
}

func (b *BinaryClusterDeployment) exists(nodeID string) bool {
	b.connLock.RLock()
	defer b.connLock.RUnlock()
	_, ok := b.connections[nodeID]
	return ok
}

func (bcp *BinaryClusterDeployment) registerNode(hcf *api.HostConfig) error {
	bcp.connLock.Lock()
	defer bcp.connLock.Unlock()
	if _, ok := bcp.connections[hcf.Address]; ok {
		logrus.Debugf("node: %s is already registered", hcf.Address)
		return nil
	}
	r, err := runner.NewSSHRunner(hcf)
	if err != nil {
		logrus.Errorf("connect node: %s failed: %v", hcf.Address, err)
		return err
	}
	bcp.connections[hcf.Address] = r

	err = nodemanager.RegisterNode(hcf, r)
	if err != nil {
		logrus.Errorf("register node: %s failed: %v", hcf.Address, err)
		return err
	}
	return nil
}

func (bcp *BinaryClusterDeployment) registerNodes() error {
	var err error
	defer func() {
		if err != nil {
			bcp.Finish()
			nodemanager.UnRegisterAllNodes()
		}
	}()

	for _, cfg := range bcp.config.Nodes {
		err := bcp.registerNode(cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bcp *BinaryClusterDeployment) taintAndLabelMasterNodes() error {
	taints := []kubectl.Taint{
		{
			Key:    "node-role.kubernetes.io/master",
			Value:  "",
			Effect: "NoSchedule",
		},
	}
	labels := make(map[string]string)
	labels["node-role.kubernetes.io/master"] = ""
	labels["node-role.kubernetes.io/control-plane"] = ""
	for _, node := range bcp.config.Nodes {
		if (node.Type&api.Master != 0) && (node.Type&api.Worker != 0) {
			// taint master node
			err := kubectl.WaitNodeRegister(node.Name, bcp.config.Name)
			if err != nil {
				logrus.Warnf("wait node: %s joined failed: %v", node.Name, err)
				continue
			}
			err = kubectl.NodeTaintAndLabel(bcp.config.Name, node.Name, labels, taints)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (bcp *BinaryClusterDeployment) prepareNetwork() error {
	// Setup coredns at here, like need addons
	if err := coredns.CorednsSetup(bcp.config); err != nil {
		logrus.Errorf("setup coredns failed: %v", err)
		return err
	}

	// setup network
	if err := network.SetupNetwork(bcp.config); err != nil {
		return err
	}

	return nil
}

func (bcp *BinaryClusterDeployment) cleanupNetwork() error {
	// cleanup network
	if err := network.CleanupNetwork(bcp.config); err != nil {
		logrus.Errorf("cleanup network failed: %v", err)
		return err
	}

	// cleanup coredns at here
	if err := coredns.CorednsCleanup(bcp.config); err != nil {
		logrus.Errorf("cleanup coredns failed: %v", err)
		return err
	}

	return nil
}

// support new apis
func (bcp *BinaryClusterDeployment) MachineInfraSetup(hcf *api.HostConfig) error {
	if hcf == nil {
		logrus.Warnf("empty host config")
		return nil
	}

	logrus.Infof("do setup %s infrastrucure...", hcf.Address)

	role := []uint16{api.Master, api.Worker, api.ETCD, api.LoadBalance}
	for _, r := range role {
		if !utils.IsType(hcf.Type, r) {
			continue
		}

		err := infrastructure.NodeInfrastructureSetup(bcp.config, hcf.Address, r)
		if err != nil {
			return err
		}
	}

	logrus.Infof("setup %s infrastrucure success", hcf.Address)
	return nil
}

func (bcp *BinaryClusterDeployment) MachineInfraDestroy(hcf *api.HostConfig) error {
	if hcf == nil {
		logrus.Warnf("empty host config")
		return nil
	}

	logrus.Infof("do destroy %s infrastrucure...", hcf.Address)

	role := []uint16{api.Master, api.Worker, api.ETCD, api.LoadBalance}
	for _, r := range role {
		if !utils.IsType(hcf.Type, r) {
			continue
		}

		err := infrastructure.NodeInfrastructureDestroy(bcp.config, hcf.Address, r)
		if err != nil {
			logrus.Errorf("role %d infrastructure destory failed: %v", r, err)
		}
	}

	logrus.Infof("destroy %s infrastrucure success", hcf.Address)
	return nil
}

func (bcp *BinaryClusterDeployment) EtcdClusterSetup() error {
	logrus.Info("do deploy etcd cluster...")
	err := etcdcluster.Init(bcp.config)
	if err != nil {
		logrus.Errorf("deploy etcd cluster failed: %v", err)
	} else {
		logrus.Info("deploy etcd cluster success")
	}
	return err
}

func (bcp *BinaryClusterDeployment) EtcdClusterDestroy() error {
	logrus.Info("do etcd cluster destroy...")
	if err := cleanupcluster.CleanupAllEtcds(bcp.config); err != nil {
		return fmt.Errorf("etcd cluster destroy failed: %v", err)
	}

	logrus.Info("do etcd cluster destroy done")
	return nil
}

func (bcp *BinaryClusterDeployment) EtcdNodeSetup(machine *api.HostConfig) error {
	// TODO: add implement
	return nil
}

func (bcp *BinaryClusterDeployment) EtcdNodeDestroy(machine *api.HostConfig) error {
	logrus.Info("do etcd node destroy...")
	if err := cleanupcluster.CleanupEtcdMember(bcp.config, machine); err != nil {
		return fmt.Errorf("cleanup etcd member %v failed", machine.Name)
	}

	logrus.Info("do etcd node destroy done")
	return nil
}

func (bcp *BinaryClusterDeployment) ClusterControlPlaneInit(master *api.HostConfig) error {
	logrus.Info("do init control plane...")
	if !bcp.exists(master.Address) {
		logrus.Errorf("cannot found master %s", master.Address)
		return fmt.Errorf("cannot found master %s", master.Address)
	}
	return controlplane.Init(bcp.config, master.Address)
}

func (bcp *BinaryClusterDeployment) ClusterNodeJoin(node *api.HostConfig) error {
	if node == nil {
		logrus.Warnf("empty join node config")
		return nil
	}

	logrus.Infof("do join node %s...", node.Address)

	if utils.IsType(node.Type, api.Master) {
		err := bootstrap.JoinMaster(bcp.config, node)
		if err != nil {
			return err
		}
	}

	if utils.IsType(node.Type, api.Worker) {
		var controlPlane *api.HostConfig
		for _, n := range bcp.config.Nodes {
			if utils.IsType(n.Type, api.Master) {
				controlPlane = n
				break
			}
		}

		if controlPlane == nil {
			return fmt.Errorf("no useful controlPlane")
		}

		err := bootstrap.JoinWorker(bcp.config, controlPlane, node)
		if err != nil {
			return err
		}
	}

	logrus.Infof("join node %s success", node.Address)
	return nil
}

func (bcp *BinaryClusterDeployment) ClusterNodeCleanup(node *api.HostConfig, delType uint16) error {
	logrus.Info("do node cleanup...")
	if err := cleanupcluster.CleanupNode(bcp.config, node, delType); err != nil {
		return fmt.Errorf("cleanup node %v failed: %v", node.Name, err)
	}
	logrus.Info("node cleanup success.")
	return nil
}

func (bcp *BinaryClusterDeployment) ClusterUpgrade() error {
	// TODO: add implement
	return nil
}

func (bcp *BinaryClusterDeployment) ClusterStatus() (*api.ClusterStatus, error) {
	// TODO: add implement
	return nil, nil
}

func (bcp *BinaryClusterDeployment) AddonsSetup() error {
	logrus.Info("do apply addons...")
	// taint and label master node before apply addons
	err := bcp.taintAndLabelMasterNodes()
	if err != nil {
		logrus.Errorf("[addons] taint master node failed: %v", err)
		return err
	}

	err = bcp.prepareNetwork()
	if err != nil {
		logrus.Errorf("[addons] prepare network failed: %v", err)
		return err
	}

	err = addons.SetupAddons(bcp.config)
	if err != nil {
		logrus.Errorf("[addons] setup addons failed: %v", err)
		return err
	}

	logrus.Info("[addons] apply addons success.")
	return nil
}

func (bcp *BinaryClusterDeployment) AddonsDestroy() error {
	logrus.Info("do destroy addons...")
	err := addons.CleanupAddons(bcp.config)
	if err != nil {
		logrus.Errorf("[addons] destroy addons failed: %v", err)
		return err
	}
	err = bcp.cleanupNetwork()
	if err != nil {
		logrus.Errorf("[addons] cleanup network failed: %v", err)
		return err
	}

	logrus.Info("[addons] destroy addons success.")
	return nil
}

func (bcp *BinaryClusterDeployment) LoadBalancerSetup(lb *api.HostConfig) error {
	if lb == nil {
		logrus.Warnf("empty loadbalancer config")
		return nil
	}

	logrus.Info("do deploy loadbalancer...")

	if err := loadbalance.SetupLoadBalancer(bcp.config, lb); err != nil {
		logrus.Errorf("bootstrap falied: %v", err)
		return err
	}

	logrus.Info("deploy loadbalancer success")
	return nil
}

func (bcp *BinaryClusterDeployment) LoadBalancerUpdate(lb *api.HostConfig) error {
	if lb == nil {
		logrus.Warnf("empty loadbalancer config")
		return nil
	}

	logrus.Info("do deploy loadbalancer...")

	if err := loadbalance.UpdateLoadBalancer(bcp.config, lb); err != nil {
		logrus.Errorf("bootstrap falied: %v", err)
		return err
	}

	logrus.Info("deploy loadbalancer success")
	return nil
}

func (bcp *BinaryClusterDeployment) LoadBalancerDestroy(lb *api.HostConfig) error {
	if lb == nil {
		logrus.Warnf("empty loadbalancer config")
		return nil
	}

	cleanupcluster.CleanupLoadBalance(bcp.config, lb)
	return nil
}

func (bcp *BinaryClusterDeployment) Finish() {
	logrus.Info("do finish binary deployment...")
	bcp.connLock.Lock()
	defer bcp.connLock.Unlock()
	for _, c := range bcp.connections {
		c.Close()
	}
	bcp.connections = make(map[string]runner.Runner)
	logrus.Info("do finish binary deployment success")
}
