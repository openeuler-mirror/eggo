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
 * Description: nodemanager testcase
 ******************************************************************************/

package etcdcluster

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
)

func TestDeployEtcd(t *testing.T) {
	certsTempDir, err := ioutil.TempDir("", "etcd-test-src-certs-")
	if err != nil {
		t.Fatalf("create tempdir for etcd certs failed: %v", err)
	}
	defer os.RemoveAll(certsTempDir)

	configsTempDir, err := ioutil.TempDir("", "etcd-test-src-configs-")
	if err != nil {
		t.Fatalf("create tempdir for etcd config failed: %v", err)
	}
	defer os.RemoveAll(configsTempDir)

	dstTempDir, err := ioutil.TempDir("", "etcd-test-dst-")
	if err != nil {
		t.Fatalf("create tempdir for dst etcd configs and certs failed: %v", err)
	}
	defer os.RemoveAll(dstTempDir)

	nodes := []*clusterdeployment.HostConfig{
		&clusterdeployment.HostConfig{
			Arch:    "amd64",
			Name:    "node0",
			Address: "192.168.0.1",
		},
		&clusterdeployment.HostConfig{
			Arch:    "aarch64",
			Name:    "node1",
			Address: "192.168.0.2",
		},
	}
	conf := &clusterdeployment.ClusterConfig{
		Certificate: clusterdeployment.CertificateConfig{SavePath: dstTempDir},
		EtcdCluster: clusterdeployment.EtcdClusterConfig{
			Token:     "etcd-cluster",
			Nodes:     nodes,
			CertsDir:  certsTempDir,
			ExtraArgs: map[string]string{"TESTARG": "testval", "ETCD_UNSUPPORTED_ARCH": "testarch"},
		},
		Nodes: nodes,
	}

	if err := prepareEtcdConfigs(conf, configsTempDir); err != nil {
		t.Fatalf("prepare etcd configs failed: %v", err)
	}

	if err := generateCerts(&runner.LocalRunner{}, conf); err != nil {
		t.Fatalf("generate etcd certs failed: %v", err)
	}

	if err := copyCertsAndConfigs(conf, &runner.LocalRunner{}, &clusterdeployment.HostConfig{
		Arch:    "aarch64",
		Name:    "node0",
		Address: "192.168.0.1",
	}, configsTempDir, filepath.Join(dstTempDir, "etcd.conf"), filepath.Join(dstTempDir, "etcd.service")); err != nil {
		t.Fatalf("copy etcd certs and configs failed: %v", err)
	}

	for _, file := range []string{
		"ca.crt", "healthcheck-client.crt", "peer.crt", "server.crt",
		"ca.key", "healthcheck-client.key", "peer.key", "server.key",
	} {
		if _, err := os.Stat(filepath.Join(dstTempDir, "etcd", file)); err != nil {
			t.Fatalf("etcd file %v not found in dst dir", file)
		}
	}

	for _, file := range []string{
		"etcd.service", "etcd.conf",
	} {
		if _, err := os.Stat(filepath.Join(dstTempDir, file)); err != nil {
			t.Fatalf("etcd file %v not found in dst dir", file)
		}
	}

	envStr, err := ioutil.ReadFile(filepath.Join(dstTempDir, "etcd.conf"))
	if err != nil {
		t.Fatalf("read etcd env config file etcd.conf failed: %v", err)
	}

	if !strings.Contains(string(envStr), "ETCD_UNSUPPORTED_ARCH=testarch") ||
		!strings.Contains(string(envStr), "TESTARG=testval") ||
		strings.Contains(string(envStr), "ETCD_UNSUPPORTED_ARCH=aarch64") {
		t.Fatalf("etcd env config file not right")
	}

	return
}