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
 * Create: 2021-05-31
 * Description: bootstrap testcase
 ******************************************************************************/

package bootstrap

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
)

type MockRunner struct {
}

func (m *MockRunner) Copy(src, dst string) error {
	logrus.Infof("copy %s to %s", src, dst)
	return nil
}

func (m *MockRunner) RunCommand(cmd string) (string, error) {
	logrus.Infof("run command: %s", cmd)
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

func TestJoinMaster(t *testing.T) {
	lr := &runner.LocalRunner{}
	masterNode := api.HostConfig{
		Arch:     "arm64",
		Name:     "master1",
		Address:  "192.168.1.2",
		Port:     22,
		UserName: "root",
		Password: "123456",
		Type:     api.Master,
	}
	conf := &api.ClusterConfig{
		Name: "test-cluster",
		APIEndpoint: api.APIEndpoint{
			AdvertiseAddress: "192.168.1.1",
			BindPort:         6443,
		},
		WorkerConfig: api.WorkerConfig{
			KubeletConf: &api.Kubelet{
				DNSVip:    "10.32.0.10",
				DNSDomain: "cluster.local",
				CniBinDir: "/opt/cni/bin",
			},
			ContainerEngineConf: &api.ContainerEngine{
				Runtime:         "iSulad",
				RuntimeEndpoint: "unix:///var/run/isulad.sock",
			},
		},
		Nodes: []*api.HostConfig{
			{
				Arch:     "x86_64",
				Name:     "master0",
				Address:  "192.168.1.1",
				Port:     22,
				UserName: "root",
				Password: "123456",
				Type:     api.Master,
			},
			&masterNode,
		},
		RoleInfra: map[uint16]*api.RoleInfra{
			api.Master: {},
		},
	}

	r := &MockRunner{}
	for _, node := range conf.Nodes {
		if err := nodemanager.RegisterNode(node, r); err != nil {
			t.Fatalf("register node failed: %v", err)
		}
	}
	defer func() {
		nodemanager.UnRegisterAllNodes()
	}()

	api.EggoHomePath = "/tmp/eggo"
	if _, err := lr.RunCommand(
		fmt.Sprintf("sudo mkdir -p -m 0777 %s/%s/pki", api.EggoHomePath, conf.Name)); err != nil {
		t.Fatalf("run command failed: %v", err)
	}
	if err := JoinMaster(conf, &masterNode); err != nil {
		t.Fatalf("do bootstrap init failed: %v", err)
	}
	t.Logf("do bootstrap init success")
}

func TestJoinWorker(t *testing.T) {
	lr := &runner.LocalRunner{}
	controlplane := api.HostConfig{
		Arch:     "x86_64",
		Name:     "master0",
		Address:  "192.168.1.1",
		Port:     22,
		UserName: "root",
		Password: "123456",
		Type:     api.Master,
	}
	workerNode := api.HostConfig{
		Arch:     "arm64",
		Name:     "worker1",
		Address:  "192.168.1.3",
		Port:     22,
		UserName: "root",
		Password: "123456",
		Type:     api.Worker,
	}
	conf := &api.ClusterConfig{
		Name: "test-cluster",
		APIEndpoint: api.APIEndpoint{
			AdvertiseAddress: "192.168.1.1",
			BindPort:         6443,
		},
		WorkerConfig: api.WorkerConfig{
			KubeletConf: &api.Kubelet{
				DNSVip:    "10.32.0.10",
				DNSDomain: "cluster.local",
				CniBinDir: "/opt/cni/bin",
			},
			ContainerEngineConf: &api.ContainerEngine{
				Runtime:         "iSulad",
				RuntimeEndpoint: "unix:///var/run/isulad.sock",
			},
		},
		Nodes: []*api.HostConfig{
			&controlplane,
			&workerNode,
		},
		RoleInfra: map[uint16]*api.RoleInfra{
			api.Worker: {},
		},
	}

	r := &MockRunner{}
	for _, node := range conf.Nodes {
		if err := nodemanager.RegisterNode(node, r); err != nil {
			t.Fatalf("register node failed: %v", err)
		}
	}
	defer func() {
		nodemanager.UnRegisterAllNodes()
	}()

	api.EggoHomePath = "/tmp/eggo"
	if _, err := lr.RunCommand(
		fmt.Sprintf("sudo mkdir -p -m 0777 %s/%s/pki", api.EggoHomePath, conf.Name)); err != nil {
		t.Fatalf("run command failed: %v", err)
	}
	if err := JoinWorker(conf, &controlplane, &workerNode); err != nil {
		t.Fatalf("do bootstrap init failed: %v", err)
	}
	t.Logf("do bootstrap init success")
}
