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
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// content is what shell contain, name is file name of shell
	RunShell(content string, name string) (string, error)
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

func (r *LocalRunner) RunShell(shell string, name string) (string, error) {
	return "", nil
}

func (r *LocalRunner) Reconnect() error {
	// nothing to do
	return nil
}

func (r *LocalRunner) Close() {
	// nothing to do
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
		Timeout:    30 * time.Minute,
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
	// cleanup output by connection
	output, err := conn.Exec("date", host)
	if err != nil {
		logrus.Warnf("run command on new connection of host: %s failed: %v", host.Name, err)
	} else {
		logrus.Debugf("host: %s, output: %s", host.Name, output)
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

func (ssh *SSHRunner) RunShell(shell string, name string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "eggo-shell-")
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s", tmpDir))
	roleBase64 := base64.StdEncoding.EncodeToString([]byte(shell))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/%s", roleBase64, tmpDir, name))
	sb.WriteString(fmt.Sprintf(" && chmod +x %s/%s", tmpDir, name))
	sb.WriteString(fmt.Sprintf(" && %s/%s > /dev/null", tmpDir, name))
	sb.WriteString(fmt.Sprintf(" && rm -rf %s", tmpDir))
	sb.WriteString("\"")

	output, err := ssh.RunCommand(sb.String())
	if err != nil {
		return "", err
	}
	return output, nil
}
