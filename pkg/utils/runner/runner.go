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

	kkv1alpha1 "github.com/kubesphere/kubekey/apis/kubekey/v1alpha1"
	"github.com/kubesphere/kubekey/pkg/util/ssh"
	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
)

const (
	RunnerShellPrefix = "eggo-shell-"
)

type Runner interface {
	// only copy file, do not support copy dir
	Copy(src, dst string) error
	RunCommand(cmd string) (string, error)
	// content is what shell contain, name is file name of shell
	RunShell(content string, name string) (string, error)
	Reconnect() error
	Close()
}

type LocalRunner struct {
}

func (r *LocalRunner) copyDir(srcDir, dstDir string) error {
	output, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -rf %v %v", srcDir, dstDir)).CombinedOutput()
	if err != nil {
		logrus.Errorf("[local] copy %s to %s failed: %v\noutput: %v\n", srcDir, dstDir, err, string(output))
		return err
	}
	logrus.Debugf("[local] copy %s to %s success", srcDir, dstDir)
	return nil
}

func (r *LocalRunner) Copy(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		logrus.Errorf("[local] check src dir: %s failed: %v", src, err)
		return err
	}
	if !fi.IsDir() {
		// just copy file
		return r.copyDir(src, dst)
	}
	output, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -f %v %v", src, dst)).CombinedOutput()
	if err != nil {
		logrus.Errorf("[local] copy %s to %s failed: %v\noutput: %v\n", src, dst, err, string(output))
	} else {
		logrus.Debugf("[local] copy %s to %s success", src, dst)
	}
	return err
}

func (r *LocalRunner) RunCommand(cmd string) (string, error) {
	output, err := exec.Command("/bin/sh", "-c", cmd).CombinedOutput()
	if err != nil {
		logrus.Errorf("[local] run command: %s, failed: %v", cmd, err)
	} else {
		logrus.Debugf("[local] run command: %s, success", cmd)
	}
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
		Name:           hcfg.Name,
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
	if err = prepareUserTempDir(conn, host); err != nil {
		logrus.Errorf("[%s] prepare user temp dir failed: %v", host.Name, err)
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

func prepareUserTempDir(conn ssh.Connection, host *kkv1alpha1.HostCfg) error {
	// scp to tmp file
	dir := api.GetUserTempDir(host.User)
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s", dir))
	// chown .eggo dir
	sb.WriteString(fmt.Sprintf(" && chown -R %s:%s %s", host.User, host.User, filepath.Dir(dir)))
	sb.WriteString("\"")
	_, err := conn.Exec(sb.String(), host)
	if err != nil {
		logrus.Errorf("[%s] prepare temp dir: %s failed: %v", host.Name, dir, err)
		return err
	}
	logrus.Debugf("[%s] prepare temp dir: %s success", host.Name, dir)
	return nil
}

func (ssh *SSHRunner) copyFile(src, dst string) error {
	if ssh.Conn == nil {
		return fmt.Errorf("[%s] SSH runner is not connected", ssh.Host.Name)
	}
	tempDir := api.GetUserTempDir(ssh.Host.User)
	// scp to tmp file
	tempCpyFile := filepath.Join(tempDir, filepath.Base(src))
	err := ssh.Conn.Scp(src, tempCpyFile)
	if err != nil {
		logrus.Errorf("[%s] Copy %s to tempfile %s failed: %v", ssh.Host.Name, src, tempCpyFile, err)
		return err
	}
	_, err = ssh.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"mv %s %s\"", tempCpyFile, dst))
	if err != nil {
		logrus.Errorf("[%s] untar tmp tar failed: %v", ssh.Host.Name, err)
		return err
	}
	logrus.Debugf("[%s] Copy %s to %s success\n", ssh.Host.Name, src, dst)
	return nil
}

func (ssh *SSHRunner) Copy(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		logrus.Errorf("[%s] check src dir: %s failed: %v", ssh.Host.Name, src, err)
		return err
	}
	if !fi.IsDir() {
		// just copy file
		return ssh.copyFile(src, dst)
	}

	// copy dir
	return ssh.copyDir(src, dst)
}

func (ssh *SSHRunner) copyDir(srcDir, dstDir string) error {
	tmpDir, err := ioutil.TempDir("", "eggo-certs-")
	if err != nil {
		logrus.Errorf("[%s] create tempdir failed: %v", ssh.Host.Name, err)
		return err
	}
	tmpPkgFile := filepath.Join(tmpDir, "pkg.tar")
	lr := &LocalRunner{}
	// tar src dir
	_, err = lr.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s && cd %s && tar -cf %s *\"", tmpDir, srcDir, tmpPkgFile))
	if err != nil {
		logrus.Errorf("[%s] create cert tmp tar failed: %v", ssh.Host.Name, err)
		return err
	}
	tmpCpyDir := api.GetUserTempDir(ssh.Host.User)
	tmpPkiFile := filepath.Join(tmpCpyDir, "remote-pkg.tar")
	// scp to user home directory
	err = ssh.Copy(tmpPkgFile, tmpPkiFile)
	if err != nil {
		logrus.Errorf("[%s] copy tmp tar failed: %v", ssh.Host.Name, err)
		return err
	}
	// untar tmp file
	_, err = ssh.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"cd %s && mv %s . && tar -xf %s && rm -rf %s\"", dstDir, tmpPkiFile, "remote-pkg.tar", "remote-pkg.tar"))
	if err != nil {
		logrus.Errorf("[%s] untar tmp tar failed: %v", ssh.Host.Name, err)
		return err
	}
	logrus.Debugf("[%s] copy dir '%s' to '%s' success", ssh.Host.Name, srcDir, dstDir)
	return nil
}

func (ssh *SSHRunner) RunCommand(cmd string) (string, error) {
	if ssh.Conn == nil {
		return "", errors.New("SSH runner is not connected")
	}
	output, err := ssh.Conn.Exec(cmd, ssh.Host)
	if err != nil {
		logrus.Errorf("[%s] run '%s' failed: %v\n", ssh.Host.Name, cmd, err)
		return "", err
	}

	logrus.Debugf("[%s] run '%s' success, output: %s\n", ssh.Host.Name, cmd, output)
	return output, nil
}

func (ssh *SSHRunner) RunShell(shell string, name string) (string, error) {
	tmpDir, err := ioutil.TempDir("", RunnerShellPrefix)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s", tmpDir))
	roleBase64 := base64.StdEncoding.EncodeToString([]byte(shell))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/%s", roleBase64, tmpDir, name))
	sb.WriteString(fmt.Sprintf(" && chmod +x %s/%s", tmpDir, name))
	sb.WriteString(fmt.Sprintf(" && %s/%s > /dev/null", tmpDir, name))
	sb.WriteString("\"")

	output, err := ssh.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("[%s] run shell '%s' failed: %v", ssh.Host.Name, name, err)
		return "", err
	}
	logrus.Debugf("[%s] run shell '%s' success, output: %s", ssh.Host.Name, name, output)
	return output, nil
}
