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
 * Description: testcase for controlplane binary implement
 ******************************************************************************/
package controlplane

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils"
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

func TestInit(t *testing.T) {
	lr := &runner.LocalRunner{}
	conf := &api.ClusterConfig{
		Name: "test-cluster",
		ServiceCluster: api.ServiceClusterConfig{
			CIDR:    "10.32.0.0/16",
			Gateway: "10.32.0.1",
		},
		APIEndpoint: api.APIEndpoint{
			AdvertiseAddress: "192.168.1.1",
			BindPort:         6443,
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
			{
				Arch:     "arm64",
				Name:     "work1",
				Address:  "192.168.1.2",
				Port:     22,
				UserName: "root",
				Password: "123456",
				Type:     api.Worker,
			},
		},
	}

	r := &MockRunner{}
	var master string
	for _, node := range conf.Nodes {
		nodemanager.RegisterNode(node, r)
		if utils.IsType(node.Type, api.Master) {
			master = node.Address
		}
	}
	defer func() {
		nodemanager.UnRegisterAllNodes()
	}()

	api.EggoHomePath = "/tmp/eggo"
	// generate api server etcd client ceritifaces for testing
	lr.RunCommand(fmt.Sprintf("sudo mkdir -p -m 0777 %s/%s/pki/etcd", api.EggoHomePath, conf.Name))
	lr.RunCommand(fmt.Sprintf("sudo chmod -R 0777 %s/%s", api.EggoHomePath, conf.Name))
	lr.RunCommand(fmt.Sprintf("sudo touch %s/%s/pki/apiserver-etcd-client.crt", api.EggoHomePath, conf.Name))
	lr.RunCommand(fmt.Sprintf("sudo touch %s/%s/pki/apiserver-etcd-client.key", api.EggoHomePath, conf.Name))
	lr.RunCommand(fmt.Sprintf("sudo touch %s/%s/pki/etcd/ca.crt", api.EggoHomePath, conf.Name))
	if err := Init(conf, master); err != nil {
		t.Fatalf("do control plane init failed: %v", err)
	}
	//lr.RunCommand(fmt.Sprintf("sudo rm -rf 0777 %s/%s", api.EggoHomePath, conf.Name))
	t.Logf("do control plane init success")
}
