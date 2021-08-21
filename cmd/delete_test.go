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
 * Description: cmd delete testcase
 ******************************************************************************/

package cmd

import (
	"testing"
)

func TestSplitDeletedConfigs(t *testing.T) {
	// test delete name
	diff, deleted := splitDeletedConfigs([]*HostConfig{
		&HostConfig{
			Name: "test1",
		},
		&HostConfig{
			Name: "test2",
		},
	}, []string{"test1"})
	if len(diff) != 1 || len(deleted) != 1 || diff[0].Name != "test1" || deleted[0].Name != "test2" {
		t.Fatalf("split deleted configs by name failed")
	}

	// test delete ip
	diff, deleted = splitDeletedConfigs([]*HostConfig{
		&HostConfig{
			Ip: "192.168.0.2",
		},
		&HostConfig{
			Ip: "192.168.0.3",
		},
	}, []string{"192.168.0.2"})
	if len(diff) != 1 || len(deleted) != 1 || diff[0].Ip != "192.168.0.2" ||
		deleted[0].Ip != "192.168.0.3" {
		t.Fatalf("split deleted configs by ip failed")
	}

	// test not found
	diff, deleted = splitDeletedConfigs([]*HostConfig{
		&HostConfig{
			Ip: "192.168.0.2",
		},
		&HostConfig{
			Ip: "192.168.0.3",
		},
	}, []string{"192.168.0.4"})
	if len(diff) != 0 || len(deleted) != 2 || deleted[0].Ip != "192.168.0.2" ||
		deleted[1].Ip != "192.168.0.3" {
		t.Fatalf("split deleted configs not found failed")
	}
}

func TestGetDeletedAndDiffConfigs(t *testing.T) {
	deployConfig := &DeployConfig{
		Masters: []*HostConfig{
			&HostConfig{
				Name: "test1",
			},
			&HostConfig{
				Name: "test2",
			},
		},
		Workers: []*HostConfig{
			&HostConfig{
				Name: "test1",
			},
			&HostConfig{
				Name: "test2",
			},
		},
		Etcds: []*HostConfig{
			&HostConfig{
				Name: "test1",
			},
			&HostConfig{
				Name: "test2",
			},
		},
	}

	conf, diff, err := getDeletedAndDiffConfigs(deployConfig, []string{"test1"})
	if err != nil || len(conf.Masters) != 1 || len(conf.Workers) != 1 || len(conf.Etcds) != 1 ||
		conf.Masters[0].Name != "test2" || conf.Workers[0].Name != "test2" || conf.Etcds[0].Name != "test2" ||
		diff[0].Name != "test1" {
		t.Fatalf("test get deleted and diff configs failed")
	}
}
