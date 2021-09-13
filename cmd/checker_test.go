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
 * Description: cmd checker testcase
 ******************************************************************************/

package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestRunChecker(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "cmd-checker-test-")
	if err != nil {
		t.Fatalf("create tempdir for cmd checker failed: %v", err)
	}
	defer os.RemoveAll(tempdir)

	// init opts
	if NewEggoCmd() == nil {
		t.Fatalf("failed to create eggo command")
	}

	f := filepath.Join(tempdir, "config.yaml")
	if err = createDeployConfigTemplate(f); err != nil {
		t.Fatalf("create deploy template config file failed: %v", err)
	}

	conf, err := loadDeployConfig(f)
	if err != nil {
		t.Fatalf("load deploy config file failed: %v", err)
	}

	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid cluster config failed: %v", err)
	}

	for _, fn := range conf.InstallConfig.PackageSrc.SrcPath {
		os.MkdirAll(fn, 0755)
		defer os.RemoveAll(fn)
	}

	// test check success
	if err = RunChecker(conf); err != nil {
		t.Fatalf("test checker success failed: %v", err)
	}

	// test invalid cluster config
	tmpClusterID := conf.ClusterID
	conf.ClusterID = ""
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid cluster config failed: %v", err)
	}
	conf.ClusterID = tmpClusterID

	// test invalid nodes
	tmpBindPort := conf.LoadBalance.BindPort
	conf.LoadBalance.BindPort = 777777
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid nodes failed: %v", err)
	}
	conf.LoadBalance.BindPort = tmpBindPort

	// test invalid service cluster
	tmpGateway := conf.Service.Gateway
	conf.Service.Gateway = "192.168.0.777"
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid service cluster failed: %v", err)
	}
	conf.Service.Gateway = tmpGateway

	// test invalid network
	tmpPodCIDR := conf.NetWork.PodCIDR
	conf.NetWork.PodCIDR = "192.168.0.777"
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid network failed: %v", err)
	}
	conf.NetWork.PodCIDR = tmpPodCIDR

	// test invalid apiSan
	if len(conf.ApiServerCertSans.DNSNames) == 0 {
		conf.ApiServerCertSans.DNSNames = []string{"test"}
	}
	tmpDNSName := conf.ApiServerCertSans.DNSNames[0]
	conf.ApiServerCertSans.DNSNames[0] = "._-^(!#%"
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid apiSan failed: %v", err)
	}
	conf.ApiServerCertSans.DNSNames[0] = tmpDNSName

	// test invalid open port
	var tmpOpenPort int
	for _, v := range conf.OpenPorts {
		for _, port := range v {
			tmpOpenPort = port.Port
			port.Port = 777777
			break
		}
	}
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid open port failed: %v", err)
	}
	for _, v := range conf.OpenPorts {
		for _, port := range v {
			port.Port = tmpOpenPort
			break
		}
	}

	// test invalid install config
	conf.InstallConfig.PackageSrc.SrcPath["test-arch"] = "package-test-arch.tar.gz"
	if err = RunChecker(conf); err == nil {
		t.Fatalf("test invalid install config failed: %v", err)
	}
	delete(conf.InstallConfig.PackageSrc.SrcPath, "test-arch")
}
