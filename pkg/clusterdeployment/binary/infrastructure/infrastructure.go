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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"

	"github.com/sirupsen/logrus"
)

var itask *task.TaskInstance

var (
	MasterPorts = []string{"6443/tcp", "10252/tcp", "10251/tcp"}
	WorkPorts   = []string{"10250/tcp", "10256/tcp"}
	EtcdPosts   = []string{"2379-2381/tcp"}
)

type InfrastructureTask struct {
	ccfg *api.ClusterConfig
}

func (it *InfrastructureTask) Name() string {
	return "InfrastructureTask"
}

func setNetBridge(r runner.Runner) error {
	const netBridgeNfCallIptablesConf = `net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
vm.swappiness=0
`
	base64Str := base64.StdEncoding.EncodeToString([]byte(netBridgeNfCallIptablesConf))

	confTmpl := `
#!/bin/bash
echo {{ .Config }} | base64 -d > /etc/sysctl.d/k8s.conf
if [ $? -ne 0 ]; then
	echo "set sysctl file failed"
	exit 1
fi

modprobe br_netfilter
if [ $? -ne 0 ]; then
	echo "modprobe br_netfilter failed"
	exit 1
fi

sysctl -p /etc/sysctl.d/k8s.conf
if [ $? -ne 0 ]; then
	echo "sysctl -p /etc/sysctl.d/k8s.conf failed"
	exit 1
fi

exit 0
`

	datastore := make(map[string]interface{})
	datastore["Config"] = base64Str

	cmdStr, err := template.TemplateRender(confTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "k8s.conf")
	if err != nil {
		return err
	}

	return nil
}

func (it *InfrastructureTask) Run(r runner.Runner, hcg *api.HostConfig) (err error) {
	if hcg == nil {
		return fmt.Errorf("empty host config")
	}

	// TODO: prepare loadbalancer
	if err = check(hcg); err != nil {
		logrus.Errorf("check failed: %v", err)
		return
	}

	if err = setNetBridge(r); err != nil {
		logrus.Errorf("set net bridge nf call iptables failed: %v", err)
		return
	}

	if err = InstallDependences(r, hcg, it.ccfg.PackageSrc); err != nil {
		logrus.Errorf("install dependences failed: %v", err)
		return err
	}

	if err = setHostname(r, hcg); err != nil {
		logrus.Errorf("set hostname failed: %v", err)
		return err
	}

	if err = addFirewallPort(r, hcg); err != nil {
		logrus.Errorf("add firewall port failed: %v", err)
		return err
	}

	return nil
}

func check(hcg *api.HostConfig) error {
	if !utils.IsX86Arch(hcg.Arch) && !utils.IsArmArch(hcg.Arch) {
		return fmt.Errorf("invalid Arch %s for %s", hcg.Arch, hcg.Address)
	}

	if hcg.Type == 0 {
		return fmt.Errorf("no role for %s", hcg.Address)
	}

	return nil
}

func setHostname(r runner.Runner, hcg *api.HostConfig) error {
	if hcg.Name == "" {
		logrus.Warnf("no name for %s", hcg.Address)
		return nil
	}

	_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"hostnamectl set-hostname %s\"", hcg.Name))
	if err != nil {
		return fmt.Errorf("set Hostname %s for %s failed: %v", hcg.Name, hcg.Address, err)
	}

	return nil
}

func addFirewallPort(r runner.Runner, hcg *api.HostConfig) error {
	ports := []string{}

	if hcg.Type&api.Master != 0 {
		ports = append(ports, MasterPorts...)
	}

	if hcg.Type&api.Worker != 0 {
		ports = append(ports, WorkPorts...)
	}

	if hcg.Type&api.ETCD != 0 {
		ports = append(ports, EtcdPosts...)
	}

	for _, p := range hcg.OpenPorts {
		port := strconv.Itoa(p.Port) + "/" + p.Protocol
		ports = append(ports, port)
	}

	if len(ports) == 0 {
		logrus.Warnf("no expose port for %s", hcg.Address)
	}

	if err := ExposePorts(r, ports...); err != nil {
		return err
	}

	return nil
}

func ExposePorts(r runner.Runner, ports ...string) error {
	if _, err := r.RunCommand(utils.AddSudo("systemctl status firewalld | grep running")); err != nil {
		logrus.Warnf("firewall is disable: %v, just ignore", err)
		return nil
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")

	rPorts := utils.RemoveDupString(ports)
	for _, p := range rPorts {
		sb.WriteString(fmt.Sprintf("firewall-cmd --zone=public --add-port=%s && ", p))
	}

	sb.WriteString("firewall-cmd --runtime-to-permanent \"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}

	return nil
}

func ShieldPorts(r runner.Runner, ports ...string) {
	if _, err := r.RunCommand(utils.AddSudo("systemctl status firewalld | grep running")); err != nil {
		logrus.Warnf("firewall is disable: %v, just ignore", err)
		return
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")

	rPorts := utils.RemoveDupString(ports)
	for _, p := range rPorts {
		sb.WriteString(fmt.Sprintf("firewall-cmd --zone=public --remove-port=%s ; ", p))
	}

	sb.WriteString("firewall-cmd --runtime-to-permanent \"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		logrus.Errorf("shield port failed: %v", err)
	}
}

func Init(config *api.ClusterConfig) error {
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

	return nil
}

func NodeInfrastructureSetup(config *api.ClusterConfig, nodeID string) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}

	itask = task.NewTaskInstance(
		&InfrastructureTask{
			ccfg: config,
		})

	if err := nodemanager.RunTaskOnNodes(itask, []string{nodeID}); err != nil {
		return fmt.Errorf("infrastructure Task failed: %v", err)
	}

	return nil
}

func NodeInfrastructureDestroy(config *api.ClusterConfig, nodeID string) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}

	// TODO: add implements

	return nil
}
