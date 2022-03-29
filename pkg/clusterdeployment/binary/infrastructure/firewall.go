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
 * Create: 2021-06-29
 * Description: eggo firewall implement
 ******************************************************************************/

package infrastructure

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/runner"
)

func getPorts(openPorts []*api.OpenPorts) []string {
	ports := []string{}

	for _, p := range openPorts {
		ports = append(ports, strconv.Itoa(p.Port)+"/"+p.Protocol)
	}

	return ports
}

func exposePorts(r runner.Runner, ports []string) error {
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

func shieldPorts(r runner.Runner, ports []string) {
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

func addFirewallPort(r runner.Runner, openPorts []*api.OpenPorts) error {
	ports := getPorts(openPorts)
	if len(ports) == 0 {
		logrus.Warnf("empty open ports")
		return nil
	}

	if err := exposePorts(r, ports); err != nil {
		return err
	}

	return nil
}

func removeFirewallPort(r runner.Runner, openPorts []*api.OpenPorts) {
	ports := getPorts(openPorts)
	if len(ports) == 0 {
		logrus.Warnf("empty open ports")
		return
	}

	shieldPorts(r, ports)
}
