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
 * Author: wangfengtu
 * Create: 2021-05-26
 * Description: cleanup cluster testcase
 ******************************************************************************/

package cleanupcluster

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
)

const (
	memberListOutput = `868b499159f00586, started, workder0, https://192.168.0.1:2380, https://192.168.0.1:2379, false
6787454327e00766, started, workder1, https://192.168.0.2:2380, https://192.168.0.2:2379, true`
)

type fakeRunner struct {
}

func (r *fakeRunner) Copy(src, dst string) error {
	logrus.Infof("copy %v to %v", src, dst)
	return nil
}

func (r *fakeRunner) RunCommand(cmd string) (string, error) {
	logrus.Infof("run command:[%v]", cmd)

	if strings.Contains(cmd, "which yum") {
		return "/usr/bin/yum", nil
	}

	if strings.Contains(cmd, "member list") {
		return memberListOutput, nil
	}

	return "", nil
}

func (m *fakeRunner) RunShell(shell string, name string) (string, error) {
	logrus.Infof("run shell: %s", name)
	return "", nil
}

func (r *fakeRunner) Reconnect() error {
	// nothing to do
	return nil
}

func (r *fakeRunner) Close() {
	// nothing to do
}

func TestRemoveWorkers(t *testing.T) {
	nodes := []*api.HostConfig{
		{
			Arch:    "amd64",
			Name:    "node0",
			Address: "192.168.0.1",
			Type:    0x07,
		},
	}
	conf := &api.ClusterConfig{
		Certificate: api.CertificateConfig{SavePath: "/tmp/test"},
		Nodes:       nodes,
	}

	task := &cleanupNodeTask{ccfg: conf}
	if err := task.Run(&fakeRunner{}, nodes[0]); err != nil {
		t.Fatalf("task execute failed for cleanup workers")
	}
}

func TestRemoveEtcds(t *testing.T) {
	nodes := []*api.HostConfig{
		{
			Arch:    "amd64",
			Name:    "node0",
			Address: "192.168.0.1",
			Type:    0x07,
		},
	}
	conf := &api.ClusterConfig{
		Certificate: api.CertificateConfig{SavePath: "/tmp/test"},
		Nodes:       nodes,
	}

	task := &cleanupEtcdMemberTask{ccfg: conf}
	if err := task.Run(&fakeRunner{}, nodes[0]); err != nil {
		t.Fatalf("task execute failed for cleanup etcds")
	}
}
