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
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/dependency"
	"isula.org/eggo/pkg/utils/nodemanager"
)

type MockRunner struct {
}

func (m *MockRunner) Copy(src, dst string) error {
	logrus.Infof("copy %s to %s", src, dst)
	return nil
}

func (m *MockRunner) RunCommand(cmd string) (string, error) {
	logrus.Infof("run command: %s", cmd)
	if cmd == fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", dependency.PmTest) {
		return "dpkg", nil
	} else if cmd == fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", dependency.PrmTest) {
		return "apt", nil
	}

	return "", nil
}

func (m *MockRunner) RunShell(shell string, name string) (string, error) {
	logrus.Infof("run shell: %s", name)
	return "", nil
}

func (m *MockRunner) Reconnect() error {
	logrus.Infof("reconnect")
	return nil
}

func (m *MockRunner) Close() {
	logrus.Infof("close")
}

func addNodes(hcfs []*api.HostConfig) {
	r := &MockRunner{}
	for _, hcf := range hcfs {
		nodemanager.RegisterNode(hcf, r)
	}
}

func TestPrepareInfrastructure(t *testing.T) {
	ccfg := &api.ClusterConfig{
		Nodes: []*api.HostConfig{
			{
				Arch:     "x86_64",
				Name:     "master",
				Address:  "192.168.0.1",
				Port:     22,
				UserName: "root",
				Password: "123456",
				Type:     api.Master,
			},
			{
				Arch:     "arm64",
				Name:     "work",
				Address:  "192.168.0.2",
				Port:     22,
				UserName: "root",
				Password: "123456",
				Type:     api.Worker,
			},
			{
				Arch:     "x86_64",
				Name:     "etcd",
				Address:  "192.168.0.3",
				Port:     22,
				UserName: "root",
				Password: "123456",
				Type:     api.Master | api.ETCD,
			},
		},
		PackageSrc: api.PackageSrcConfig{
			Type:   "",
			ArmSrc: "",
			X86Src: "",
		},
		RoleInfra: map[uint16]*api.RoleInfra{
			api.Master: {
				OpenPorts: []*api.OpenPorts{
					{
						Port:     1234,
						Protocol: "tcp",
					},
				},
				Softwares: []*api.PackageConfig{
					{
						Name: "openssl",
						Type: "repo",
					},
					{
						Name: "kubernetes-client",
						Type: "repo",
					},
					{
						Name: "kubernetes-master",
						Type: "repo",
					},
					{
						Name: "coredns",
						Type: "repo",
					},
				},
			},
			api.Worker: {
				OpenPorts: []*api.OpenPorts{
					{
						Port:     2345,
						Protocol: "udp",
					},
				},
				Softwares: []*api.PackageConfig{
					{
						Name: "hostname",
						Type: "repo",
					},
					{
						Name: "kubectl",
						Type: "repo",
					},
					{
						Name: "kube-proxy",
						Type: "repo",
					},
				},
			},
			api.ETCD: {
				OpenPorts: []*api.OpenPorts{
					{
						Port:     12345,
						Protocol: "tcp",
					},
					{
						Port:     23456,
						Protocol: "udp",
					},
				},
				Softwares: []*api.PackageConfig{
					{
						Name: "etcd",
						Type: "repo",
					},
				},
			},
		},
	}

	addNodes(ccfg.Nodes)
	if err := NodeInfrastructureSetup(ccfg, ccfg.Nodes[0].Address, ccfg.Nodes[0].Type); err != nil {
		t.Fatalf("test NodeInfrastructureSetup failed: %v\n", err)
	}

	if err := NodeInfrastructureDestroy(ccfg, ccfg.Nodes[0]); err != nil {
		t.Fatalf("test NodeInfrastructureDestroy failed: %v\n", err)
	}

	nodemanager.UnRegisterAllNodes()
}
