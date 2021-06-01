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

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
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
		LocalEndpoint: api.APIEndpoint{
			AdvertiseAddress: "192.168.1.1",
			BindPort:         6443,
		},
		ControlPlane: api.ControlPlaneConfig{
			Endpoint: "eggo.com:6443",
			KubeletConf: &api.Kubelet{
				DnsVip:          "10.32.0.10",
				DnsDomain:       "cluster.local",
				CniBinDir:       "/opt/cni/bin",
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
			{
				Arch:     "arm64",
				Name:     "master1",
				Address:  "192.168.1.2",
				Port:     22,
				UserName: "root",
				Password: "123456",
				Type:     api.Master | api.Worker,
			},
		},
	}

	r := &MockRunner{}
	for _, node := range conf.Nodes {
		nodemanager.RegisterNode(node, r)
	}
	defer func() {
		nodemanager.UnRegisterAllNodes()
	}()

	api.EggoHomePath = "/tmp/eggo"
	lr.RunCommand(fmt.Sprintf("sudo mkdir -p -m 0777 %s/%s/pki", api.EggoHomePath, conf.Name))
	if err := Init(conf); err != nil {
		t.Fatalf("do bootstrap init failed: %v", err)
	}
	t.Logf("do bootstrap init success")
}
