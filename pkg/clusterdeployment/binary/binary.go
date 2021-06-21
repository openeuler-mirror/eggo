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
	"sync"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/addons"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/bootstrap"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/cleanupcluster"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/commontools"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/controlplane"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/etcdcluster"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/infrastructure"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/loadbalance"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/network"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/manager"
	"gitee.com/openeuler/eggo/pkg/utils/kubectl"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"

	"github.com/sirupsen/logrus"
)

const (
	name = "binary"
)

func init() {
	if err := manager.RegisterClusterDeploymentDriver(name, New); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("register binary")
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

	connLock    sync.Mutex
	connections map[string]runner.Runner
}

func (bcp *BinaryClusterDeployment) registerNodes() error {
	bcp.connLock.Lock()
	defer bcp.connLock.Unlock()

	var err error
	defer func() {
		if err != nil {
			bcp.Finish()
			nodemanager.UnRegisterAllNodes()
		}
	}()

	for _, cfg := range bcp.config.Nodes {
		if _, ok := bcp.connections[cfg.Address]; ok {
			continue
		}
		r, err := runner.NewSSHRunner(cfg)
		if err != nil {
			logrus.Errorf("connect node: %s failed: %v", cfg.Address, err)
			return err
		}
		bcp.connections[cfg.Address] = r

		err = nodemanager.RegisterNode(cfg, r)
		if err != nil {
			logrus.Errorf("register node: %s failed: %v", cfg.Address, err)
			return err
		}
	}
	return nil
}

func (bcp *BinaryClusterDeployment) Finish() {
	bcp.connLock.Lock()
	defer bcp.connLock.Unlock()
	for _, c := range bcp.connections {
		c.Close()
	}
	bcp.connections = make(map[string]runner.Runner)
}

func (bcp *BinaryClusterDeployment) PrepareInfrastructure() error {
	logrus.Info("do prepare infrastructure...")
	if err := infrastructure.Init(bcp.config); err != nil {
		logrus.Errorf("prepare infrastructure falied: %v", err)
		return err
	}

	logrus.Info("prepare infrastructe success")
	return nil
}

func (bcp *BinaryClusterDeployment) DeployEtcdCluster() error {
	logrus.Info("do deploy etcd cluster...")
	err := etcdcluster.Init(bcp.config)
	if err != nil {
		logrus.Errorf("deploy etcd cluster failed: %v", err)
	} else {
		logrus.Info("deploy etcd cluster success")
	}
	return err
}

func (bcp *BinaryClusterDeployment) DeployLoadBalancer() error {
	logrus.Info("do join loadbalancer...")
	if err := loadbalance.Init(bcp.config); err != nil {
		logrus.Errorf("bootstrap falied: %v", err)
		return err
	}

	logrus.Info("do join loadbalancer success")
	return nil
}

func (bcp *BinaryClusterDeployment) InitControlPlane() error {
	logrus.Info("do init control plane...")
	controlplane.Init(bcp.config)
	return nil
}

func (bcp *BinaryClusterDeployment) JoinBootstrap() error {
	logrus.Info("do join new worker or master...")
	if err := bootstrap.Init(bcp.config); err != nil {
		logrus.Errorf("bootstrap falied: %v", err)
		return err
	}

	logrus.Info("do join new worker or master success")
	return nil
}

func (bcp *BinaryClusterDeployment) UpgradeCluster() error {
	logrus.Info("do update cluster...")
	return nil
}

func (bcp *BinaryClusterDeployment) CleanupCluster() error {
	logrus.Info("do clean cluster...")
	err := cleanupcluster.Init(bcp.config)
	if err != nil {
		logrus.Infof("cleanup cluster failed: %v", err)
	} else {
		logrus.Info("cleanup cluster success")
	}
	return err
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
			r, ok := bcp.connections[node.Address]
			if !ok {
				logrus.Warnf("cannot find node: %s connection", node.Name)
				continue
			}
			err := kubectl.WaitNodeJoined(r, node.Name, bcp.config)
			if err != nil {
				logrus.Warnf("wait node: %s joined failed: %v", node.Name, err)
				continue
			}
			err = kubectl.AddNodeTaints(bcp.config, r, node.Name, taints)
			if err != nil {
				return err
			}
			err = kubectl.AddNodeLabels(bcp.config, r, node.Name, labels)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (bcp *BinaryClusterDeployment) PrepareNetwork() error {
	// Setup coredns at here, like need addons
	if err := commontools.SetUpCoredns(bcp.config); err != nil {
		logrus.Errorf("setup coredns failed: %v", err)
		return err
	}

	// setup network
	if err := network.SetupNetwork(bcp.config); err != nil {
		return err
	}

	return nil
}

func (bcp *BinaryClusterDeployment) ApplyAddons() error {
	// taint and label master node before apply addons
	err := bcp.taintAndLabelMasterNodes()
	if err != nil {
		logrus.Errorf("taint master node failed: %v", err)
		return err
	}

	err = addons.SetupAddons(bcp.config)
	if err != nil {
		logrus.Errorf("setup addons failed: %v", err)
		return err
	}

	return nil
}

func (bcp *BinaryClusterDeployment) ClusterStatus() (*api.ClusterStatus, error) {
	// TODO: support ClusterStatus
	return nil, nil
}
