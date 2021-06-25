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

	"isula.org/eggo/pkg/api"
	_ "isula.org/eggo/pkg/clusterdeployment/binary"
	"isula.org/eggo/pkg/clusterdeployment/manager"
	"github.com/sirupsen/logrus"
)

func CreateCluster(cc *api.ClusterConfig) error {
	if cc == nil {
		return fmt.Errorf("cluster config is required")
	}
	creator, err := manager.GetClusterDeploymentDriver(cc.DeployDriver)
	if err != nil {
		logrus.Errorf("get cluster deployment driver: %s failed: %v", cc.DeployDriver, err)
		return err
	}
	handler, err := creator(cc)
	if err != nil {
		logrus.Errorf("create cluster deployment instance with driver: %s, failed: %v", cc.DeployDriver, err)
		return err
	}
	defer handler.Finish()

	// prepare eggo config directory
	if err := os.MkdirAll(api.GetClusterHomePath(cc.Name), 0750); err != nil {
		return err
	}

	if err := handler.PrepareInfrastructure(); err != nil {
		return err
	}
	if err := handler.DeployEtcdCluster(); err != nil {
		return err
	}
	if err := handler.DeployLoadBalancer(); err != nil {
		return err
	}
	if err := handler.InitControlPlane(); err != nil {
		return err
	}
	if err := handler.JoinBootstrap(); err != nil {
		return err
	}
	if err := handler.PrepareNetwork(); err != nil {
		return err
	}
	if err := handler.ApplyAddons(); err != nil {
		return err
	}
	logrus.Infof("[cluster] create cluster '%s' successed", cc.Name)
	return nil
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
	if err := handler.CleanupCluster(); err != nil {
		return err
	}
	// cleanup eggo config directory
	if err := os.RemoveAll(api.GetClusterHomePath(cc.Name)); err != nil {
		logrus.Warnf("[cluster] cleanup eggo config directory failed: %v", err)
	}
	logrus.Infof("[cluster] remove cluster '%s' successed", cc.Name)
	return nil
}
