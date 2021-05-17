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
 * Description: eggo controlplane binary implement
 ******************************************************************************/

package controlplane

import (
	"fmt"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

const (
	KubeSoftwares = []string{"kubectl", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}
)

type ControlPlaneTask struct {
	ccfg *clusterdeployment.ClusterConfig
	task.Labels
}

var ctask *ControlPlaneTask

func (ct *ControlPlaneTask) Name() string {
	return "ControlplaneTask"
}

func (ct *ControlPlaneTask) Run(r runner.Runner, hcf *clusterdeployment.HostConfig) error {
	// do precheck phase
	err := check(r)
	if err != nil {
		return err
	}

	return nil
}

func check(r runner.Runner) error {
	// check dependences softwares
	for s := range KubeSoftwares {
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"which %s\"", s))
		if err != nil {
			logrus.Errorf("chech kubernetes software: %s, failed: %v\n", s, err)
			return err
		}
		logrus.Debugf("check kubernetes software: %s success\n", s)
	}
	return nil
}

func installDependences() error {
	// TODO: maybe just do in infrastructure
	return nil
}

func generateCerts() error {
	return nil
}

func generateKubeconfigs() error {
	return nil
}

func runKubernetesServices() error {
	return nil
}

func Init(conf *clusterdeployment.ClusterConfig) error {
	ctask = &ControlPlaneTask{
		ccfg: conf,
	}

	// TODO: run task on every controlplane node
	return nil
}
