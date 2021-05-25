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

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/certs"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
)

const (
	EtcdConfFile       = "/etc/etcd/etcd.conf"
	EtcdServiceFile    = "/usr/lib/systemd/system/etcd.service"
	DefaultEtcdDataDir = "/var/lib/etcd"
)

var (
	tempDir = ""
)

type copyInfo struct {
	src string
	dst string
}

type EtcdDeployEtcdsTask struct {
	ccfg *clusterdeployment.ClusterConfig
}

func (t *EtcdDeployEtcdsTask) Name() string {
	return "EtcdDeployEtcdsTask"
}

func getDstCertsDir(ccfg *clusterdeployment.ClusterConfig) string {
	if ccfg.Certificate.SavePath != "" {
		return ccfg.Certificate.SavePath
	} else {
		return certs.DefaultCertPath
	}
}

func getDstEtcdCertsDir(ccfg *clusterdeployment.ClusterConfig) string {
	return filepath.Join(getDstCertsDir(ccfg), "etcd")
}

func copyCertsAndConfigs(ccfg *clusterdeployment.ClusterConfig, r runner.Runner,
	hostConfig *clusterdeployment.HostConfig, tempConfigsDir string, dstConf string,
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
	certsDir := ccfg.EtcdCluster.CertsDir
	etcdDir := filepath.Join(certsDir, "etcd")
	dstCertsDir := getDstEtcdCertsDir(ccfg)

	caCrt := filepath.Join(etcdDir, "ca.crt")
	copyInfos = append(copyInfos, &copyInfo{src: caCrt, dst: filepath.Join(dstCertsDir, "ca.crt")})
	caKey := filepath.Join(etcdDir, "ca.key")
	copyInfos = append(copyInfos, &copyInfo{src: caKey, dst: filepath.Join(dstCertsDir, "ca.key")})

	serverCrt := filepath.Join(etcdDir, hostConfig.Name+"-server.crt")
	copyInfos = append(copyInfos, &copyInfo{src: serverCrt, dst: filepath.Join(dstCertsDir, "server.crt")})
	serverKey := filepath.Join(etcdDir, hostConfig.Name+"-server.key")
	copyInfos = append(copyInfos, &copyInfo{src: serverKey, dst: filepath.Join(dstCertsDir, "server.key")})

	peerCrt := filepath.Join(etcdDir, hostConfig.Name+"-peer.crt")
	copyInfos = append(copyInfos, &copyInfo{src: peerCrt, dst: filepath.Join(dstCertsDir, "peer.crt")})
	peerKey := filepath.Join(etcdDir, hostConfig.Name+"-peer.key")
	copyInfos = append(copyInfos, &copyInfo{src: peerKey, dst: filepath.Join(dstCertsDir, "peer.key")})

	healthcheckCrt := filepath.Join(etcdDir, hostConfig.Name+"-healthcheck-client.crt")
	copyInfos = append(copyInfos, &copyInfo{src: healthcheckCrt, dst: filepath.Join(dstCertsDir, "healthcheck-client.crt")})
	healthcheckKey := filepath.Join(etcdDir, hostConfig.Name+"-healthcheck-client.key")
	copyInfos = append(copyInfos, &copyInfo{src: healthcheckKey, dst: filepath.Join(dstCertsDir, "healthcheck-client.key")})

	createDirsCmd := "sudo mkdir -p " + filepath.Dir(dstConf) + " && chmod 700 " + filepath.Dir(dstConf) +
		" && mkdir -p " + dstCertsDir + " && chmod 700 " + dstCertsDir + " && mkdir -p " + filepath.Dir(dstService)
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

func (t *EtcdDeployEtcdsTask) Run(r runner.Runner, hostConfig *clusterdeployment.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	if err := copyCertsAndConfigs(t.ccfg, r, hostConfig, tempDir, EtcdConfFile, EtcdServiceFile); err != nil {
		return err
	}

	if output, err := r.RunCommand("sudo systemctl enable etcd.service"); err != nil {
		return fmt.Errorf("run command on %v to enable etcd service failed: %v\noutput: %v",
			hostConfig.Address, err, output)
	}

	return nil
}

type EtcdPostDeployEtcdsTask struct {
	ccfg *clusterdeployment.ClusterConfig
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

func (t *EtcdPostDeployEtcdsTask) Run(r runner.Runner, hostConfig *clusterdeployment.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	if err := healthcheck(r, getDstEtcdCertsDir(t.ccfg), hostConfig.Address); err != nil {
		return err
	}

	return nil
}

func prepareEtcdConfigs(ccfg *clusterdeployment.ClusterConfig, tempConfigsDir string) error {
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
			CertsDir:      getDstCertsDir(ccfg),
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

func getAllIps(nodes []*clusterdeployment.HostConfig) []string {
	var ips []string

	for _, node := range nodes {
		ips = append(ips, node.Address)
	}

	return ips
}

func Init(conf *clusterdeployment.ClusterConfig) error {
	tempDir, err := ioutil.TempDir("", "etcd-conf-")
	if err != nil {
		return fmt.Errorf("create tempdir for etcd config failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// prepare config
	if err := prepareEtcdConfigs(conf, tempDir); err != nil {
		return err
	}

	// generate certificates
	if err := generateCerts(&runner.LocalRunner{}, conf); err != nil {
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
