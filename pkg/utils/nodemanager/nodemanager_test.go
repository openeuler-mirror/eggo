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
 * Create: 2021-05-13
 * Description: nodemanager testcase
 ******************************************************************************/

package nodemanager

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

type MockRunner struct {
}

func (m *MockRunner) Copy(src, dst string) error {
	logrus.Infof("copy %s to %s", src, dst)
	return nil
}

func (m *MockRunner) CopyDir(src, dst string) error {
	logrus.Infof("copydir %s to %s", src, dst)
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

type MockTask struct {
	// some need data
	name string
}

func (m *MockTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	rand.Seed(time.Now().UnixNano())

	err := r.Copy("/home/data", "/data")
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	_, err = r.RunCommand(m.name + " run 'top'")
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	r.Reconnect()

	return err
}

func (m *MockTask) Name() string {
	return m.name
}

func addNodes() {
	hcf1 := &api.HostConfig{
		Arch:     "x86_64",
		Name:     "master",
		Address:  "192.168.0.1",
		Port:     22,
		UserName: "root",
		Password: "123456",
		Type:     api.Master,
	}
	hcf2 := &api.HostConfig{
		Arch:     "arm64",
		Name:     "work",
		Address:  "192.168.0.2",
		Port:     22,
		UserName: "root",
		Password: "123456",
		Type:     api.Worker,
	}
	r := &MockRunner{}
	RegisterNode(hcf1, r)
	RegisterNode(hcf2, r)
}

func releaseNodes(nodes []string) {
	for _, id := range nodes {
		UnRegisterNode(id)
	}
}

func TestRunTaskOnNodes(t *testing.T) {
	addNodes()
	tt := task.NewTaskInstance(
		&MockTask{
			name: "precheck",
		})
	nodes := []string{"192.168.0.1", "192.168.0.2"}
	err := RunTaskOnNodes(tt, nodes)
	if err != nil {
		t.Fatalf("run task on ondes failed: %v\n", err)
	}

	err = WaitTaskOnNodesFinished(tt, nodes, time.Second*30)
	if err != nil {
		t.Fatalf("run task on ondes failed: %v\n", err)
	}

	errTask := task.NewTaskInstance(
		&ErrorTask{
			name: "ErrorTask",
		})
	err = RunTaskOnNodes(errTask, nodes)
	if err != nil {
		t.Fatalf("run err task failed: %v", err)
	}
	err = WaitTaskOnNodesFinished(errTask, nodes, time.Second*30)
	if err == nil {
		t.Fatal("run error task on ondes success")
	}
	releaseNodes(nodes)
}

func TestRunTaskOnAll(t *testing.T) {
	addNodes()
	tt := task.NewTaskInstance(
		&MockTask{
			name: "precheck",
		},
	)
	err := RunTaskOnAll(tt)
	if err != nil {
		t.Fatalf("run task on all node failed: %v\n", err)
	}
	err = WaitTaskOnAllFinished(tt, time.Second*30)
	if err != nil {
		t.Fatal("run task on all ondes failed\n")
	}
	UnRegisterAllNodes()
}

type ErrorTask struct {
	// some need data
	name string
}

func (m *ErrorTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	rand.Seed(time.Now().UnixNano())

	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	logrus.Errorf("run error task return error")

	return fmt.Errorf("ErrorTask is error")
}

func (m *ErrorTask) Name() string {
	return m.name
}
