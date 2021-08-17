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
 * Create: 2021-05-22
 * Description: etcd cluster testcase
 ******************************************************************************/

package etcdcluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
)

const (
	memberListOutput = `868b499159f00586, started, worker0, https://192.168.0.1:2380, https://192.168.0.1:2379, false
6787454327e00766, started, worker1, https://192.168.0.2:2380, https://192.168.0.2:2379, true`
	memberAddOutput = `added member 6787454327e00766 to cluster

ETCD_NAME="worker1"
ETCD_INITIAL_CLUSTER="worker0=http://192.168.0.1:2380,worker1=http://192.168.0.2:2380"
ETCD_INITIAL_ADVERTISE_PEER_URLS="http://192.168.0.2:2380"
ETCD_INITIAL_CLUSTER_STATE="existing"
`
)

var (
	nodes = []*api.HostConfig{
		{
			Arch:    "amd64",
			Name:    "worker0",
			Address: "192.168.0.1",
			Type:    api.Master | api.Worker | api.ETCD,
		},
		{
			Arch:    "amd64",
			Name:    "worker1",
			Address: "192.168.0.2",
			Type:    api.Master | api.Worker | api.ETCD,
		},
	}
	conf = &api.ClusterConfig{
		Certificate: api.CertificateConfig{SavePath: "/tmp/test"},
		Nodes:       nodes,
		WorkerConfig: api.WorkerConfig{
			ContainerEngineConf: &api.ContainerEngine{Runtime: "docker"},
		},
		EtcdCluster: api.EtcdClusterConfig{
			Nodes: nodes,
		},
		RoleInfra: map[uint16]*api.RoleInfra{
			api.Master: {},
			api.Worker: {},
			api.ETCD:   {},
		},
	}
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

	if strings.Contains(cmd, "member add") {
		return memberAddOutput, nil
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

func registerFakeRunner(t *testing.T) {
	if err := nodemanager.RegisterNode(nodes[0], &fakeRunner{}); err != nil {
		t.Fatalf("register fakerunner for worker0 failed")
	}
	if err := nodemanager.RegisterNode(nodes[1], &fakeRunner{}); err != nil {
		t.Fatalf("register fakerunner for worker1 failed")
	}
}

func TestEtcdCertsAndConfig(t *testing.T) {
	certsTempDir, err := ioutil.TempDir("", "etcd-test-eggo-")
	if err != nil {
		t.Fatalf("create eggo config dir failed: %v", err)
	}
	defer os.RemoveAll(certsTempDir)

	api.EggoHomePath = certsTempDir
	certsTempDir = api.GetCertificateStorePath("test-cluster")

	dstTempDir, err := ioutil.TempDir("", "etcd-test-dst-")
	if err != nil {
		t.Fatalf("create tempdir for dst etcd configs and certs failed: %v", err)
	}
	defer os.RemoveAll(dstTempDir)

	allNodes := []*api.HostConfig{
		{
			Arch:    "amd64",
			Name:    "node0",
			Address: "192.168.0.1",
		},
		{
			Arch:    "aarch64",
			Name:    "node1",
			Address: "192.168.0.2",
		},
	}
	deployConf := &api.ClusterConfig{
		Name:        "test-cluster",
		Certificate: api.CertificateConfig{SavePath: dstTempDir},
		EtcdCluster: api.EtcdClusterConfig{
			Token:     "etcd-cluster",
			Nodes:     allNodes,
			CertsDir:  certsTempDir,
			ExtraArgs: map[string]string{"TESTARG": "testval", "ETCD_UNSUPPORTED_ARCH": "testarch"},
		},
		Nodes: allNodes,
	}

	r := &runner.LocalRunner{}
	if err = prepareEtcdConfigs(deployConf, r, &api.HostConfig{
		Arch:    "aarch64",
		Name:    "node0",
		Address: "192.168.0.1",
	}, "", filepath.Join(dstTempDir, "etcd.conf"), filepath.Join(dstTempDir, "etcd.service")); err != nil {
		t.Fatalf("prepare etcd configs failed: %v", err)
	}

	if err = generateCaAndApiserverEtcdCerts(r, deployConf); err != nil {
		t.Fatalf("generate ca and apiserver etcd certs failed: %v", err)
	}

	if err = copyCa(deployConf, r, &api.HostConfig{
		Arch:    "aarch64",
		Name:    "node0",
		Address: "192.168.0.1",
	}); err != nil {
		t.Fatalf("copy etcd certs and configs failed: %v", err)
	}

	if err = generateEtcdCerts(r, deployConf, allNodes[0]); err != nil {
		t.Fatalf("generate etcd certs failed: %v", err)
	}

	// change temp dir to 777, ensure ut success
	if _, err = r.RunCommand(utils.AddSudo(fmt.Sprintf("chmod -R 777 %s/etcd", dstTempDir))); err != nil {
		t.Fatalf("chmod etcd dir failed: %v", err)
	}

	for _, file := range []string{
		"ca.crt", "healthcheck-client.crt", "peer.crt", "server.crt",
		"ca.key", "healthcheck-client.key", "peer.key", "server.key",
	} {
		if _, err = os.Stat(filepath.Join(dstTempDir, "etcd", file)); err != nil {
			t.Fatalf("check file: %s failed: %v", file, err)
		}
	}

	for _, file := range []string{
		"etcd.service", "etcd.conf",
	} {
		if _, err = os.Stat(filepath.Join(dstTempDir, file)); err != nil {
			t.Fatalf("etcd file %v not found in dst dir", file)
		}
	}

	envStr, err := r.RunCommand("sudo cat " + filepath.Join(dstTempDir, "etcd.conf"))
	if err != nil {
		t.Fatalf("read etcd env config file etcd.conf failed: %v", err)
	}

	if !strings.Contains(string(envStr), "ETCD_UNSUPPORTED_ARCH=testarch") ||
		!strings.Contains(string(envStr), "TESTARG=testval") ||
		strings.Contains(string(envStr), "ETCD_UNSUPPORTED_ARCH=aarch64") {
		t.Fatalf("etcd env config file not right")
	}
}

func TestDeployEtcd(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "etcdcluster-test-")
	if err != nil {
		t.Fatalf("create tempdir for cmd configs failed: %v", err)
	}
	defer os.RemoveAll(tempdir)
	api.EggoHomePath = tempdir

	registerFakeRunner(t)

	if err := Init(conf); err != nil {
		t.Fatalf("deploy etcd cluster failed")
	}
}
