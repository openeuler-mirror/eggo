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
 * Author: zhangxiaoyu
 * Create: 2021-05-12
 * Description: eggo infrastructure binary implement
 ******************************************************************************/

package infrastructure

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"

	"github.com/sirupsen/logrus"
)

var itask *task.TaskInstance

type InfrastructureTask struct {
	ccfg *clusterdeployment.ClusterConfig
}

func (it *InfrastructureTask) Name() string {
	return "InfrastructureTask"
}

func (it *InfrastructureTask) Run(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	if hcg == nil {
		return fmt.Errorf("empty host config")
	}

	// TODO: prepare loadbalancer
	if err := check(hcg); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := InstallDependences(r, hcg, it.ccfg.PackageSrc); err != nil {
		logrus.Errorf("install dependences failed: %v", err)
		return err
	}

	if err := setHostname(r, hcg); err != nil {
		logrus.Errorf("set hostname failed: %v", err)
		return err
	}

	if err := addFirewallPort(r, hcg); err != nil {
		logrus.Errorf("add firewall port failed: %v", err)
		return err
	}

	return nil
}

func check(hcg *clusterdeployment.HostConfig) error {
	if !utils.IsX86Arch(hcg.Arch) && !utils.IsArmArch(hcg.Arch) {
		return fmt.Errorf("invalid Arch %s for %s", hcg.Arch, hcg.Address)
	}

	if hcg.Type == 0 {
		return fmt.Errorf("no role for %s", hcg.Address)
	}

	return nil
}

func setHostname(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	if hcg.Name == "" {
		logrus.Warnf("no name for %s", hcg.Address)
		return nil
	}

	_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"hostnamectl sethostname %s\"", hcg.Name))
	if err != nil {
		return fmt.Errorf("set Hostname %s for %s failed: %v", hcg.Name, hcg.Address, err)
	}

	return nil
}

func addFirewallPort(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	var ports []string
	masterPorts := []string{"6443/tcp", "10252/tcp", "10251/tcp"}
	workPorts := []string{"10250/tcp", "10256/tcp"}
	etcdPosts := []string{"2379-2381/tcp"}

	if hcg.Type&clusterdeployment.Master != 0 {
		ports = append(ports, masterPorts...)
	}

	if hcg.Type&clusterdeployment.Worker != 0 {
		ports = append(ports, workPorts...)
	}

	if hcg.Type&clusterdeployment.ETCD != 0 {
		ports = append(ports, etcdPosts...)
	}

	for _, p := range hcg.OpenPorts {
		port := strconv.Itoa(p.Port) + "/" + p.Protocol
		ports = append(ports, port)
	}

	if err := exposePorts(r, ports...); err != nil {
		return err
	}

	return nil
}

func exposePorts(r runner.Runner, ports ...string) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for _, p := range ports {
		sb.WriteString(fmt.Sprintf("firewall-cmd --zone=public --add-port=%s && ", p))
	}

	sb.WriteString("firewall-cmd --runtime-to-permanent \"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}

	return nil
}

func PrepareInfrastructure(config *clusterdeployment.ClusterConfig) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}

	itask = task.NewTaskInstance(
		&InfrastructureTask{
			ccfg: config,
		})

	if err := nodemanager.RunTaskOnAll(itask); err != nil {
		return fmt.Errorf("infrastructure Task failed: %v", err)
	}

	if err := nodemanager.WaitTaskOnAllFinished(itask, time.Second*120); err != nil {
		return fmt.Errorf("wait Infrastructure Task failed: %v", err)
	}

	return nil
}
