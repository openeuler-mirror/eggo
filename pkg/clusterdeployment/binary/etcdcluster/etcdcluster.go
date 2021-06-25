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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/commontools"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

const (
	EtcdConfFile       = "/etc/etcd/etcd.conf"
	EtcdServiceFile    = "/usr/lib/systemd/system/etcd.service"
	DefaultEtcdDataDir = "/var/lib/etcd/default.etcd"
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

func copyCa(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig) error {
	var copyInfos []*copyInfo

	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	// certs
	certsDir := api.GetCertificateStorePath(ccfg.Name)
	etcdDir := filepath.Join(certsDir, "etcd")
	dstCertsDir := getDstEtcdCertsDir(ccfg)

	caCrt := filepath.Join(etcdDir, "ca.crt")
	copyInfos = append(copyInfos, &copyInfo{src: caCrt, dst: filepath.Join(dstCertsDir, "ca.crt")})
	caKey := filepath.Join(etcdDir, "ca.key")
	copyInfos = append(copyInfos, &copyInfo{src: caKey, dst: filepath.Join(dstCertsDir, "ca.key")})

	createDirsCmd := "mkdir -p -m 0700 " + dstCertsDir
	if output, err := r.RunCommand(utils.AddSudo(createDirsCmd)); err != nil {
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

	// prepare etcd dir
	if err := prepareEtcdDir(r); err != nil {
		logrus.Errorf("prepare etcd dir failed: %v", err)
		return err
	}

	// prepare config
	if err := prepareEtcdConfigs(t.ccfg, r, hostConfig, EtcdConfFile, EtcdServiceFile); err != nil {
		return err
	}

	if err := copyCa(t.ccfg, r, hostConfig); err != nil {
		return err
	}

	// generate etcd-server etcd-peer and etcd-health-check certificates on etcd nodes
	if err := generateEtcdCerts(r, t.ccfg, hostConfig); err != nil {
		return err
	}

	shell, err := commontools.GetSystemdServiceShell("etcd", "", true)
	if err != nil {
		logrus.Errorf("get etcd systemd service shell failed: %v", err)
		return err
	}
	if output, err := r.RunShell(shell, "etcd"); err != nil {
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
	cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl endpoint health --endpoints=https://%v:2379 --cacert=%v/ca.crt --cert=%v/server.crt --key=%v/server.key", ip, etcdCertsDir, etcdCertsDir, etcdCertsDir)
	if output, err := r.RunCommand(utils.AddSudo(cmd)); err != nil {
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

func prepareEtcdDir(r runner.Runner) error {
	dirs := []string{filepath.Dir(EtcdConfFile), filepath.Dir(DefaultEtcdDataDir)}

	// create etcd working dir
	join := strings.Join(dirs, " ")
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s\"", join)); err != nil {
		return err
	}

	return nil
}

func prepareEtcdConfigs(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig,
	confPath string, servicePath string) error {
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

	conf := &etcdEnvConfig{
		Arch:          hostConfig.Arch,
		Ip:            hostConfig.Address,
		Token:         ccfg.EtcdCluster.Token,
		Hostname:      hostConfig.Name,
		State:         "new",
		PeerAddresses: peerAddresses,
		DataDir:       dataDir,
		CertsDir:      ccfg.GetCertDir(),
		ExtraArgs:     ccfg.EtcdCluster.ExtraArgs,
	}

	base64Str := base64.StdEncoding.EncodeToString([]byte(createEtcdEnv(conf)))
	cmd := fmt.Sprintf("echo %v | base64 -d > %v", base64Str, confPath)
	if output, err := r.RunCommand(utils.AddSudo(cmd)); err != nil {
		return fmt.Errorf("run command on %v to create etcd config file failed: %v\noutput: %v",
			hostConfig.Address, err, output)
	}

	base64Str = base64.StdEncoding.EncodeToString([]byte(createEtcdService()))
	cmd = fmt.Sprintf("echo %v | base64 -d > %v", base64Str, servicePath)
	if output, err := r.RunCommand(utils.AddSudo(cmd)); err != nil {
		return fmt.Errorf("run command on %v to create etcd service file failed: %v\noutput: %v",
			hostConfig.Address, err, output)
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

	if err := nodemanager.WaitNodesFinish(nodes, time.Second*60*5); err != nil {
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

	if err := nodemanager.WaitNodesFinish(nodes, time.Second*60*5); err != nil {
		return fmt.Errorf("wait for post deploy etcds task finish failed: %v", err)
	}

	return nil
}
