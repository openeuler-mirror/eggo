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
 * Create: 2021-08-17
 * Description: cmd join testcase
 ******************************************************************************/

package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"isula.org/eggo/pkg/api"
)

func TestParseJoinInput(t *testing.T) {
	// test parse join input by command
	conf, err := parseJoinInput("", &HostConfig{Ip: "192.168.0.5"}, "master,worker", "clusterid")
	if err != nil || len(conf.Masters) != 1 || len(conf.Workers) != 1 || len(conf.Etcds) != 1 ||
		conf.Masters[0].Ip != "192.168.0.5" || conf.Workers[0].Ip != "192.168.0.5" ||
		conf.Etcds[0].Ip != "192.168.0.5" {
		t.Fatalf("test parse join input by command failed")
	}

	// test invalid type failed
	_, err = parseJoinInput("", &HostConfig{Ip: "192.168.0.5"}, "kkk", "clusterid")
	if err == nil {
		t.Fatalf("test parse join input invalid type failed")
	}

	tempdir, err := ioutil.TempDir("", "cmd-join-test-")
	if err != nil {
		t.Fatalf("create tempdir for cmd join test failed: %v", err)
	}
	defer os.RemoveAll(tempdir)

	f := filepath.Join(tempdir, "config.yaml")
	if err = createDeployConfigTemplate(f); err != nil {
		t.Fatalf("create deploy template config file failed: %v", err)
	}

	// test parse join input by file
	conf, err = parseJoinInput(f, &HostConfig{}, "", "k8s-cluster")
	if err != nil || len(conf.Masters) != 1 || len(conf.Workers) != 2 || len(conf.Etcds) != 1 ||
		conf.Masters[0].Ip != "192.168.0.2" || conf.Etcds[0].Ip != "192.168.0.2" ||
		conf.Workers[0].Ip != "192.168.0.3" || conf.Workers[1].Ip != "192.168.0.4" {
		t.Fatalf("test parse join input by file failed")
	}

	// test parse join input conflict
	_, err = parseJoinInput(f, &HostConfig{}, "worker", "k8s-cluster")
	if err == nil {
		t.Fatalf("test parse join input conflict failed")
	}
}

func TestGetMergedAndDiffConfigs(t *testing.T) {
	deployConfig := &DeployConfig{
		Masters: []*HostConfig{
			&HostConfig{
				Name: "test1",
				Ip:   "192.168.0.2",
				Arch: "arm64",
				Port: 22,
			},
		},
		Workers: []*HostConfig{
			&HostConfig{
				Name: "test1",
				Ip:   "192.168.0.2",
				Arch: "arm64",
				Port: 22,
			},
		},
		Etcds: []*HostConfig{
			&HostConfig{
				Name: "test1",
				Ip:   "192.168.0.2",
				Arch: "arm64",
				Port: 22,
			},
		},
	}

	joinConf := &DeployConfig{
		Masters: []*HostConfig{
			&HostConfig{
				Name: "test2",
				Ip:   "192.168.0.3",
				Arch: "arm64",
				Port: 22,
			},
		},
		Workers: []*HostConfig{
			&HostConfig{
				Name: "test2",
				Ip:   "192.168.0.3",
				Arch: "arm64",
				Port: 22,
			},
			&HostConfig{
				Name: "test3",
				Ip:   "192.168.0.4",
				Arch: "arm64",
				Port: 22,
			},
		},
	}

	conf, diff, err := getMergedAndDiffConfigs(deployConfig, joinConf)
	if err != nil || len(conf.Masters) != 2 || len(conf.Workers) != 3 || len(conf.Etcds) != 2 ||
		conf.Masters[0].Name != "test1" || conf.Workers[0].Name != "test1" || conf.Etcds[0].Name != "test1" ||
		conf.Masters[1].Name != "test2" || conf.Workers[1].Name != "test2" || conf.Etcds[1].Name != "test2" ||
		conf.Workers[2].Name != "test3" || diff[0].Name != "test2" || diff[1].Name != "test3" ||
		diff[0].Type != (api.Master|api.Worker|api.ETCD) || diff[1].Type != api.Worker {
		t.Fatalf("test get deleted and diff configs failed")
	}
}
