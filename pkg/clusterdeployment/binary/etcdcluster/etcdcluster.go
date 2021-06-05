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
 * Create: 2021-05-19
 * Description: eggo etcdcluster binary implement
 ******************************************************************************/

package etcdcluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
)

const (
	EtcdConfFile       = "/etc/etcd/etcd.conf"
	EtcdServiceFile    = "/usr/lib/systemd/system/etcd.service"
	DefaultEtcdDataDir = "/var/lib/etcd/default.etcd"
)

var (
	tempDir = ""
)

type copyInfo struct {
	src string
	dst string
}

type EtcdDeployEtcdsTask struct {
	ccfg *api.ClusterConfig
}

func (t *EtcdDeployEtcdsTask) Name() string {
	return "EtcdDeployEtcdsTask"
}

func getDstEtcdCertsDir(ccfg *api.ClusterConfig) string {
	return filepath.Join(ccfg.GetCertDir(), "etcd")
}

func copyCaAndConfigs(ccfg *api.ClusterConfig, r runner.Runner,
	hostConfig *api.HostConfig, tempConfigsDir string, dstConf string,
	dstService string) error {
	var copyInfos []*copyInfo

	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	// etcd configs
	envFile := filepath.Join(tempConfigsDir, hostConfig.Name+".conf")
	copyInfos = append(copyInfos, &copyInfo{src: envFile, dst: dstConf})

	serviceFile := filepath.Join(tempConfigsDir, "etcd.service")
	copyInfos = append(copyInfos, &copyInfo{src: serviceFile, dst: dstService})

	// certs
	certsDir := api.GetCertificateStorePath(ccfg.Name)
	etcdDir := filepath.Join(certsDir, "etcd")
	dstCertsDir := getDstEtcdCertsDir(ccfg)

	caCrt := filepath.Join(etcdDir, "ca.crt")
	copyInfos = append(copyInfos, &copyInfo{src: caCrt, dst: filepath.Join(dstCertsDir, "ca.crt")})
	caKey := filepath.Join(etcdDir, "ca.key")
	copyInfos = append(copyInfos, &copyInfo{src: caKey, dst: filepath.Join(dstCertsDir, "ca.key")})

	createDirsCmd := "sudo mkdir -p -m 0700 " + filepath.Dir(dstConf) +
		" && mkdir -p -m 0700 " + dstCertsDir
	if output, err := r.RunCommand(createDirsCmd); err != nil {
		return fmt.Errorf("run command on %v to create dirs failed: %v\noutput: %v",
			hostConfig.Address, err, output)
	}

	for output, info := range copyInfos {
		if err := r.Copy(info.src, info.dst); err != nil {
			return fmt.Errorf("copy %v to %v failed: %v\noutput: %v", info.src,
				hostConfig.Address, err, output)
		}
	}

	return nil
}

func (t *EtcdDeployEtcdsTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	if err := copyCaAndConfigs(t.ccfg, r, hostConfig, tempDir, EtcdConfFile, EtcdServiceFile); err != nil {
		return err
	}

	// generate etcd-server etcd-peer and etcd-health-check certificates on etcd nodes
	if err := generateEtcdCerts(r, t.ccfg, hostConfig); err != nil {
		return err
	}

	if output, err := r.RunCommand("sudo systemctl enable etcd.service && systemctl daemon-reload && systemctl restart etcd.service"); err != nil {
		return fmt.Errorf("run command on %v to enable etcd service failed: %v\noutput: %v",
			hostConfig.Address, err, output)
	}

	return nil
}

type EtcdPostDeployEtcdsTask struct {
	ccfg *api.ClusterConfig
}

func (t *EtcdPostDeployEtcdsTask) Name() string {
	return "EtcdPostDeployEtcdsTask"
}

func healthcheck(r runner.Runner, etcdCertsDir string, ip string) error {
	cmd := fmt.Sprintf("sudo -E /bin/sh -c \"ETCDCTL_API=3 etcdctl endpoint health --endpoints=https://%v:2379 --cacert=%v/ca.crt --cert=%v/server.crt --key=%v/server.key\"", ip, etcdCertsDir, etcdCertsDir, etcdCertsDir)
	if output, err := r.RunCommand(cmd); err != nil {
		return fmt.Errorf("etcd in %v healthcheck failed: %v\noutput: %v", ip, err, output)
	}
	return nil
}

func (t *EtcdPostDeployEtcdsTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	if err := healthcheck(r, getDstEtcdCertsDir(t.ccfg), hostConfig.Address); err != nil {
		return err
	}

	return nil
}

func prepareEtcdConfigs(ccfg *api.ClusterConfig, tempConfigsDir string) error {
	var peerAddresses string
	dataDir := ccfg.EtcdCluster.DataDir
	if dataDir == "" {
		dataDir = DefaultEtcdDataDir
	}

	nodes := ccfg.EtcdCluster.Nodes
	if len(nodes) == 0 {
		return fmt.Errorf("no etcd node found in config")
	}

	for i, node := range nodes {
		if i != 0 {
			peerAddresses += ","
		}
		peerAddresses += node.Name + "=https://" + node.Address + ":2380"
	}

	for _, node := range nodes {
		conf := &etcdEnvConfig{
			Arch:          node.Arch,
			Ip:            node.Address,
			Token:         ccfg.EtcdCluster.Token,
			Hostname:      node.Name,
			State:         "new",
			PeerAddresses: peerAddresses,
			DataDir:       dataDir,
			CertsDir:      ccfg.GetCertDir(),
			ExtraArgs:     ccfg.EtcdCluster.ExtraArgs,
		}
		envFile := filepath.Join(tempConfigsDir, node.Name+".conf")
		if err := ioutil.WriteFile(envFile, []byte(createEtcdEnv(conf)), 0700); err != nil {
			return fmt.Errorf("write etcd env file to %s failed: %v", envFile, err)
		}
	}

	serviceFile := filepath.Join(tempConfigsDir, "etcd.service")
	if err := ioutil.WriteFile(serviceFile, []byte(createEtcdService()), 0700); err != nil {
		return fmt.Errorf("write etcd service file to %s failed: %v", serviceFile, err)
	}

	return nil
}

func getAllIps(nodes []*api.HostConfig) []string {
	var ips []string

	for _, node := range nodes {
		ips = append(ips, node.Address)
	}

	return ips
}

func Init(conf *api.ClusterConfig) error {
	var err error
	tempDir, err = ioutil.TempDir("", "etcd-conf-")
	if err != nil {
		return fmt.Errorf("create tempdir for etcd config failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// prepare config
	if err := prepareEtcdConfigs(conf, tempDir); err != nil {
		return err
	}

	// generate ca certificates and kube-apiserver-etcd-client certificates
	if err := generateCaAndApiserverEtcdCerts(&runner.LocalRunner{}, conf); err != nil {
		return err
	}

	taskDeployEtcds := task.NewTaskInstance(
		&EtcdDeployEtcdsTask{
			ccfg: conf,
		},
	)

	nodes := getAllIps(conf.EtcdCluster.Nodes)
	if err := nodemanager.RunTaskOnNodes(taskDeployEtcds, nodes); err != nil {
		return fmt.Errorf("run task on nodes failed: %v", err)
	}

	if err := nodemanager.WaitTaskOnNodesFinished(taskDeployEtcds, nodes, time.Second*60*5); err != nil {
		return fmt.Errorf("wait for deploy etcds task finish failed: %v", err)
	}

	taskPostDeployEtcds := task.NewTaskInstance(
		&EtcdPostDeployEtcdsTask{
			ccfg: conf,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskPostDeployEtcds, nodes); err != nil {
		return fmt.Errorf("run task on nodes failed: %v", err)
	}

	if err := nodemanager.WaitTaskOnNodesFinished(taskPostDeployEtcds, nodes, time.Second*60*5); err != nil {
		return fmt.Errorf("wait for post deploy etcds task finish failed: %v", err)
	}

	return nil
}
