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

	cp "gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/controlplane"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/infrastructure"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"

	"github.com/sirupsen/logrus"
)

const (
	name = "binary driver"
)

func init() {
	if err := cp.RegisterClusterDeploymentDriver(name, New); err != nil {
		logrus.Fatal(err)
	}
}

func New(conf *cp.ClusterConfig) (cp.ClusterDeploymentAPI, error) {
	// TODO: finish binary implements
	bcd := BinaryClusterDeployment{
		config:      conf,
		connections: make(map[string]runner.Runner),
	}
	// register and connect all nodes
	bcd.registerNodes()

	return &bcd, nil
}

type BinaryClusterDeployment struct {
	config *cp.ClusterConfig

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
	infrastructure.PrepareInfrastructure(bcp.config)
	return nil
}

func (bcp *BinaryClusterDeployment) DeployEtcdCluster() error {
	logrus.Info("do deploy etcd cluster...")
	return nil
}

func (bcp *BinaryClusterDeployment) InitControlPlane() error {
	logrus.Info("do init control plane...")
	controlplane.Init(bcp.config)
	return nil
}

func (bcp *BinaryClusterDeployment) JoinBootstrap() error {
	logrus.Info("do join new work or master...")
	return nil
}

func (bcp *BinaryClusterDeployment) UpgradeCluster() error {
	logrus.Info("do update cluster...")
	return nil
}

func (bcp *BinaryClusterDeployment) CleanupCluster() error {
	logrus.Info("do clean cluster...")
	return nil
}
