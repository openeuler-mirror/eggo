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
 * Create: 2021-05-17
 * Description: infrastructure testcase
 ******************************************************************************/

package infrastructure

import (
	"testing"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"github.com/sirupsen/logrus"
)

type MockRunner struct {
}

func (m *MockRunner) Copy(src, dst string) error {
	logrus.Infof("copy %s to %s", src, dst)
	return nil
}

func (m *MockRunner) RunCommand(cmd string) (string, error) {
	logrus.Infof("run command: %s", cmd)
	if cmd == pmT {
		return "dpkg", nil
	} else if cmd == prmT {
		return "apt", nil
	}

	return "", nil
}

func (m *MockRunner) Reconnect() error {
	logrus.Infof("reconnect")
	return nil
}

func (m *MockRunner) Close() {
	logrus.Infof("close")
}

func addNodes() {
	hcfs := []*clusterdeployment.HostConfig{
		{
			Arch:     "x86_64",
			Name:     "master",
			Address:  "192.168.0.1",
			Port:     22,
			UserName: "root",
			Password: "123456",
			Type:     clusterdeployment.Master,
			OpenPorts: []*clusterdeployment.OpenPorts{
				{
					Port:     1234,
					Protocol: "tcp",
				},
			},
			Packages: map[string]clusterdeployment.Packages{
				"openssl": {
					Type: "repo",
				},
				"kubernetes-client": {
					Type: "repo",
				},
				"kubernetes-master": {
					Type: "repo",
				},
				"coredns": {
					Type: "pkg",
				},
			},
		},
		{
			Arch:     "arm64",
			Name:     "work",
			Address:  "192.168.0.2",
			Port:     22,
			UserName: "root",
			Password: "123456",
			Type:     clusterdeployment.Worker,
			OpenPorts: []*clusterdeployment.OpenPorts{
				{
					Port:     2345,
					Protocol: "udp",
				},
			},
			Packages: map[string]clusterdeployment.Packages{
				"hostname": {
					Type: "repo",
				},
				"kubectl": {
					Type: "binary",
					Dst:  "/usr/bin/",
				},
				"kubelet": {
					Type: "binary",
					Dst:  "/usr/bin/",
				},
				"kube-proxy": {
					Type: "binary",
					Dst:  "/usr/bin/",
				},
			},
		},
		{
			Arch:     "x86_64",
			Name:     "etcd",
			Address:  "192.168.0.3",
			Port:     22,
			UserName: "root",
			Password: "123456",
			Type:     clusterdeployment.Master | clusterdeployment.ETCD,
			OpenPorts: []*clusterdeployment.OpenPorts{
				{
					Port:     12345,
					Protocol: "tcp",
				},
				{
					Port:     23456,
					Protocol: "udp",
				},
			},
			Packages: map[string]clusterdeployment.Packages{
				"ipcalc": {
					Type: "repo",
				},
				"etcd": {
					Type: "pkg",
				},
				"kube-apiserver": {
					Type: "binary",
					Dst:  "/usr/bin/",
				},
				"kube-controller-manager": {
					Type: "binary",
					Dst:  "/usr/bin/",
				},
				"kube-scheduler": {
					Type: "binary",
					Dst:  "/usr/bin/",
				},
			},
		},
	}

	r := &MockRunner{}
	for _, hcf := range hcfs {
		nodemanager.RegisterNode(hcf, r)
	}
}

func TestPrepareInfrastructure(t *testing.T) {
	addNodes()

	ccfg := &clusterdeployment.ClusterConfig{
		PackageSrc: &clusterdeployment.PackageSrcConfig{
			Type:   "tar.gz",
			ArmSrc: "/etc/eggo/arm.tar.gz",
			X86Src: "/etc/eggo/x86.tar.gz",
		},
	}

	if err := PrepareInfrastructure(ccfg); err != nil {
		t.Fatalf("test PrepareInfrastructure failed: %v\n", err)
	}

	nodemanager.UnRegisterAllNodes()
}
