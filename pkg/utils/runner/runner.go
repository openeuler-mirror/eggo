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
 * Create: 2021-05-11
 * Description: runner interface and SSHRunner implements
 ******************************************************************************/

package runner

import (
	"errors"
	"time"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	kkv1alpha1 "github.com/kubesphere/kubekey/apis/kubekey/v1alpha1"
	"github.com/kubesphere/kubekey/pkg/util/ssh"
	"github.com/sirupsen/logrus"
)

type Runner interface {
	Copy(src, dst string) error
	RunCommand(cmd string) error
	Reconnect() error
	Close()
}

type SSHRunner struct {
	Host *kkv1alpha1.HostCfg
	Conn ssh.Connection
}

func connect(host *kkv1alpha1.HostCfg) (ssh.Connection, error) {
	opts := ssh.Cfg{
		Username:   host.User,
		Port:       host.Port,
		Address:    host.Address,
		Password:   host.Password,
		PrivateKey: host.PrivateKey,
		KeyFile:    host.PrivateKeyPath,
		Timeout:    30 * time.Second,
	}
	return ssh.NewConnection(opts)
}

func HostConfigToKKCfg(hcfg *clusterdeployment.HostConfig) *kkv1alpha1.HostCfg {
	return &kkv1alpha1.HostCfg{
		User:           hcfg.UserName,
		Port:           hcfg.Port,
		Address:        hcfg.Address,
		Password:       hcfg.Password,
		PrivateKey:     hcfg.PrivateKey,
		PrivateKeyPath: hcfg.PrivateKeyPath,
	}
}

func NewSSHRunner(hcfg *clusterdeployment.HostConfig) (Runner, error) {
	host := HostConfigToKKCfg(hcfg)
	conn, err := connect(host)
	if err != nil {
		return nil, err
	}
	return &SSHRunner{Host: host, Conn: conn}, nil
}

func (ssh *SSHRunner) Close() {
	logrus.Debugf("TODO: wait kubekey support close for Connection")
}

func (ssh *SSHRunner) Reconnect() error {
	conn, err := connect(ssh.Host)
	if err != nil {
		return nil
	}
	ssh.Conn = conn
	return nil
}

func (ssh *SSHRunner) Copy(src, dst string) error {
	if ssh.Conn == nil {
		return errors.New("SSH runner is not connected")
	}
	err := ssh.Conn.Scp(src, dst)
	if err != nil {
		logrus.Errorf("Copy %s to %s:%s failed\n", src, ssh.Host.Address, dst)
		return err
	}
	logrus.Debugf("Copy %s to %s:%s success\n", src, ssh.Host.Address, dst)
	return nil
}

func (ssh *SSHRunner) RunCommand(cmd string) error {
	if ssh.Conn == nil {
		return errors.New("SSH runner is not connected")
	}
	output, err := ssh.Conn.Exec(cmd, ssh.Host)
	if err != nil {
		logrus.Errorf("run '%s' on %s failed: %v, output: %s\n", cmd, ssh.Host.Address, err, output)
		return err
	}

	logrus.Debugf("run '%s' on %s success, output: %s\n", cmd, ssh.Host.Address, output)
	return nil
}
