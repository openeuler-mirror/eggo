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
 * Create: 2021-08-18
 * Description: etcd join testcase
 ******************************************************************************/

package etcdcluster

import (
	"io/ioutil"
	"os"
	"testing"

	"isula.org/eggo/pkg/api"
)

func TestAddMember(t *testing.T) {
	node0 := &api.HostConfig{
		Arch:    "amd64",
		Name:    "worker0",
		Address: "192.168.0.1",
		Type:    api.Master | api.Worker | api.ETCD,
	}
	conf = &api.ClusterConfig{
		Certificate: api.CertificateConfig{SavePath: "/tmp/test"},
		Nodes:       []*api.HostConfig{node0},
		WorkerConfig: api.WorkerConfig{
			ContainerEngineConf: &api.ContainerEngine{Runtime: "docker"},
		},
		EtcdCluster: api.EtcdClusterConfig{
			Nodes: []*api.HostConfig{node0},
		},
		RoleInfra: map[uint16]*api.RoleInfra{
			api.Master: {},
			api.Worker: {},
			api.ETCD:   {},
		},
	}

	tempdir, err := ioutil.TempDir("", "etcdcluster-test-")
	if err != nil {
		t.Fatalf("create tempdir for cmd configs failed: %v", err)
	}
	defer os.RemoveAll(tempdir)
	api.EggoHomePath = tempdir

	node1 := &api.HostConfig{
		Arch:    "amd64",
		Name:    "worker1",
		Address: "192.168.0.2",
		Type:    api.Master | api.Worker | api.ETCD,
	}

	registerFakeRunner(t)

	if err := AddMember(conf, node1); err != nil {
		t.Fatalf("deploy etcd cluster failed")
	}
}
