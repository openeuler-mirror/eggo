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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	kkv1alpha1 "github.com/kubesphere/kubekey/apis/kubekey/v1alpha1"
	"github.com/kubesphere/kubekey/pkg/util/ssh"
	"github.com/sirupsen/logrus"
)

type Runner interface {
	// only copy file, do not support copy dir
	Copy(src, dst string) error
	CopyDir(srcDir, dstDir string) error
	RunCommand(cmd string) (string, error)
	Reconnect() error
	Close()
}

type LocalRunner struct {
}

func (r *LocalRunner) CopyDir(srcDir, dstDir string) error {
	output, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo cp -rf %v %v", srcDir, dstDir)).CombinedOutput()
	if err != nil {
		logrus.Errorf("copy %s to %s failed: %v\noutput: %v\n", srcDir, dstDir, err, string(output))
	}
	return err
}

func (r *LocalRunner) Copy(src, dst string) error {
	output, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo cp -f %v %v", src, dst)).CombinedOutput()
	if err != nil {
		logrus.Errorf("copy %s to %s failed: %v\noutput: %v\n", src, dst, err, string(output))
	}
	return err
}

func (r *LocalRunner) RunCommand(cmd string) (string, error) {
	output, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	return string(output), err
}

func (r *LocalRunner) Reconnect() error {
	// nothing to do
	return nil
}

func (r *LocalRunner) Close() {
	// nothing to do
	return
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

func HostConfigToKKCfg(hcfg *api.HostConfig) *kkv1alpha1.HostCfg {
	return &kkv1alpha1.HostCfg{
		User:           hcfg.UserName,
		Port:           hcfg.Port,
		Address:        hcfg.Address,
		Password:       hcfg.Password,
		PrivateKey:     hcfg.PrivateKey,
		PrivateKeyPath: hcfg.PrivateKeyPath,
	}
}

func NewSSHRunner(hcfg *api.HostConfig) (Runner, error) {
	host := HostConfigToKKCfg(hcfg)
	conn, err := connect(host)
	if err != nil {
		return nil, err
	}
	return &SSHRunner{Host: host, Conn: conn}, nil
}

func (ssh *SSHRunner) Close() {
	// TODO: wait kubekey support close for Connection
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

func (ssh *SSHRunner) CopyDir(srcDir, dstDir string) error {
	fi, err := os.Stat(srcDir)
	if err != nil {
		logrus.Errorf("check src dir: %s failed: %v", srcDir, err)
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("src dir %s is not a directory", srcDir)
	}
	tmpDir, err := ioutil.TempDir("", "eggo-certs-")
	if err != nil {
		return err
	}
	tmpCert := filepath.Join(tmpDir, "pki.tar.gz")
	lr := &LocalRunner{}
	// tar src dir
	_, err = lr.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s && cd %s && tar -cf %s *\"", tmpDir, srcDir, tmpCert))
	if err != nil {
		logrus.Errorf("create cert tmp tar failed: %v", err)
		return err
	}
	// scp to dist
	err = ssh.Copy(tmpCert, filepath.Join(dstDir, "pki.tar.gz"))
	if err != nil {
		logrus.Errorf("copy tmp tar failed: %v", err)
		return err
	}
	// untar tmp file
	_, err = ssh.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"cd %s && tar -xf %s && rm -f %s\"", dstDir, "pki.tar.gz", "pki.tar.gz"))
	if err != nil {
		logrus.Errorf("untar tmp tar failed: %v", err)
	}
	return err
}

func (ssh *SSHRunner) RunCommand(cmd string) (string, error) {
	if ssh.Conn == nil {
		return "", errors.New("SSH runner is not connected")
	}
	output, err := ssh.Conn.Exec(cmd, ssh.Host)
	if err != nil {
		logrus.Errorf("run '%s' on %s failed: %v\n", cmd, ssh.Host.Address, err)
		return "", err
	}

	logrus.Debugf("run '%s' on %s success, output: %s\n", cmd, ssh.Host.Address, output)
	return output, nil
}
