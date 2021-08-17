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
 * Author: zhangxiaoyu
 * Create: 2021-05-12
 * Description: eggo infrastructure binary implement
 ******************************************************************************/

package infrastructure

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/cleanupcluster"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/dependency"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"

	"github.com/sirupsen/logrus"
)

var (
	pmd *packageMD5 = &packageMD5{
		MD5s: make(map[string]string),
	}
)

type SetupInfraTask struct {
	packageSrc *api.PackageSrcConfig
	roleInfra  *api.RoleInfra
}

func (it *SetupInfraTask) Name() string {
	return "SetupInfraTask"
}

func (it *SetupInfraTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	if err := check(r, hcg, it.packageSrc); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := setNetBridge(r); err != nil {
		logrus.Errorf("set net bridge nf call iptables failed: %v", err)
		return err
	}

	if err := copyPackage(r, hcg, it.packageSrc); err != nil {
		logrus.Errorf("prepare package failed: %v", err)
		return err
	}

	if err := dependency.InstallDependency(r, it.roleInfra, hcg, it.packageSrc.GetPkgDstPath()); err != nil {
		logrus.Errorf("install dependency failed: %v", err)
		return err
	}

	if err := addHostNameIP(r, hcg); err != nil {
		logrus.Errorf("add host name ip failed: %v", err)
		return err
	}

	if err := addFirewallPort(r, it.roleInfra.OpenPorts); err != nil {
		logrus.Errorf("add firewall port failed: %v", err)
		return err
	}

	return nil
}

func check(r runner.Runner, hcg *api.HostConfig, packageSrc *api.PackageSrcConfig) error {
	if hcg == nil {
		return fmt.Errorf("empty host config")
	}

	if packageSrc == nil {
		return fmt.Errorf("empty package source config")
	}

	if !utils.IsX86Arch(hcg.Arch) && !utils.IsArmArch(hcg.Arch) {
		return fmt.Errorf("invalid Arch %s for %s", hcg.Arch, hcg.Address)
	}

	if _, err := r.RunCommand("sudo -E /bin/sh -c \"which md5sum\""); err != nil {
		return fmt.Errorf("no command md5sum on %s", hcg.Address)
	}

	return nil
}

func setNetBridge(r runner.Runner) error {
	const netBridgeNfCallIptablesConf = `net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
vm.swappiness=0
`
	base64Str := base64.StdEncoding.EncodeToString([]byte(netBridgeNfCallIptablesConf))

	confTmpl := `
#!/bin/bash
echo {{ .Config }} | base64 -d > /etc/sysctl.d/k8s.conf
if [ $? -ne 0 ]; then
	echo "set sysctl file failed" 1>&2
	exit 1
fi

modprobe br_netfilter
if [ $? -ne 0 ]; then
	echo "modprobe br_netfilter failed" 1>&2
	exit 1
fi

sysctl -p /etc/sysctl.d/k8s.conf
if [ $? -ne 0 ]; then
	echo "sysctl -p /etc/sysctl.d/k8s.conf failed" 1>&2
	exit 1
fi

exit 0
`

	datastore := make(map[string]interface{})
	datastore["Config"] = base64Str

	cmdStr, err := template.TemplateRender(confTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "k8s.conf")
	if err != nil {
		return err
	}

	return nil
}

func getPackageSrcPath(arch string, pcfg *api.PackageSrcConfig) string {
	if utils.IsX86Arch(arch) {
		return pcfg.X86Src
	}

	return pcfg.ArmSrc
}

func copyPackage(r runner.Runner, hcg *api.HostConfig, pcfg *api.PackageSrcConfig) error {
	src := getPackageSrcPath(hcg.Arch, pcfg)
	if src == "" {
		logrus.Warnf("no package source path")
		return nil
	}

	// 1. calculate package MD5
	md5, err := pmd.getMD5(src)
	if err != nil {
		return fmt.Errorf("get MD5 failed: %v", err)
	}

	// 2. package exist on remote host
	file, dstDir := filepath.Base(src), pcfg.GetPkgDstPath()
	dstPath := filepath.Join(dstDir, file)
	if checkMD5(r, md5, dstPath) {
		logrus.Warnf("package already exist on remote host")
		return nil
	}

	// 3. copy package
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s\"", dstDir)); err != nil {
		return err
	}
	if err := r.Copy(src, dstPath); err != nil {
		return fmt.Errorf("copy from %s to %s for %s failed: %v", src, dstPath, hcg.Address, err)
	}

	// 4. check package MD5
	if !checkMD5(r, md5, dstPath) {
		return fmt.Errorf("%s MD5 has changed after copy, maybe it is corrupted", file)
	}

	// 5. uncompress package
	// TODO: support other compress method
	switch pcfg.Type {
	case "tar.gz", "":
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"cd %s && tar -zxvf %s\"", dstDir, file))
		if err != nil {
			return fmt.Errorf("uncompress %s failed for %s: %v", src, hcg.Address, err)
		}
	default:
		return fmt.Errorf("cannot support uncompress %s", pcfg.Type)
	}

	return nil
}

func addHostNameIP(r runner.Runner, hcg *api.HostConfig) error {
	shell := `
#!/bin/bash
cat /etc/hosts | grep "{{ .Address }}" | grep "{{ .Name }}"
if [ $? -eq 0 ]; then
	exit 0
fi

echo "{{ .Address }} {{ .Name }}" >> /etc/hosts
exit 0
`

	if hcg.Name == "" || hcg.Address == "" {
		logrus.Warnf("no name or address")
		return nil
	}

	datastore := make(map[string]interface{})
	datastore["Address"] = hcg.Address
	datastore["Name"] = hcg.Name

	cmdStr, err := template.TemplateRender(shell, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "addHostNameIP")
	if err != nil {
		return err
	}

	return nil
}

func removeHostNameIP(r runner.Runner, hcg *api.HostConfig) error {
	shell := `
#!/bin/bash
cat /etc/hosts | grep "{{ .Address }}" | grep "{{ .Name }}"
if [ $? -ne 0 ]; then
	exit 0
fi

sed -i '/{{ .Address }} {{ .Name }}/d' /etc/hosts
exit 0
`

	if hcg.Name == "" || hcg.Address == "" {
		logrus.Warnf("no name or address")
		return nil
	}

	datastore := make(map[string]interface{})
	datastore["Address"] = hcg.Address
	datastore["Name"] = hcg.Name

	cmdStr, err := template.TemplateRender(shell, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "removeHostNameIP")
	if err != nil {
		return err
	}

	return nil
}

func checkMD5(r runner.Runner, md5, path string) bool {
	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"md5sum %s | awk '{print \\$1}'\"", path))
	if err != nil {
		logrus.Warnf("get %s MD5 failed: %v", path, err)
		return false
	}

	logrus.Debugf("package MD5 value: local %s, remote: %s", md5, output)
	return md5 == output
}

func NodeInfrastructureSetup(config *api.ClusterConfig, nodeID string, role uint16) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}

	roleInfra := config.RoleInfra[role]
	if roleInfra == nil {
		return fmt.Errorf("do not register %d roleinfra", role)
	}

	itask := task.NewTaskInstance(
		&SetupInfraTask{
			packageSrc: &config.PackageSrc,
			roleInfra:  roleInfra,
		})

	if err := nodemanager.RunTaskOnNodes(itask, []string{nodeID}); err != nil {
		return fmt.Errorf("setup infrastructure Task failed: %v", err)
	}

	return nil
}

type DestroyInfraTask struct {
	packageSrc *api.PackageSrcConfig
	roleInfra  *api.RoleInfra
}

func (it *DestroyInfraTask) Name() string {
	return "DestroyInfraTask"
}

func (it *DestroyInfraTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	if hcg == nil {
		return fmt.Errorf("empty host config")
	}

	dependency.RemoveDependency(r, it.roleInfra, hcg, it.packageSrc.GetPkgDstPath())

	if err := removeHostNameIP(r, hcg); err != nil {
		logrus.Errorf("remove host name ip failed: %v", err)
	}

	removeFirewallPort(r, it.roleInfra.OpenPorts)

	cleanupcluster.PostCleanup(r)

	dstDir := it.packageSrc.GetPkgDstPath()
	if !dependency.CheckPath(dstDir) {
		logrus.Errorf("path %s not in White List and cannot remove", dstDir)
		return nil
	}
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"rm -rf %s\"", dstDir)); err != nil {
		return fmt.Errorf("rm dependency failed: %v", err)
	}

	return nil
}

func deleteSoftwareIfExist(infras *api.RoleInfra, delSoftware *api.PackageConfig) {
	for {
		var found bool
		for i, software := range infras.Softwares {
			if software.Name == delSoftware.Name && software.Type == delSoftware.Type &&
				software.Dst == delSoftware.Dst {
				infras.Softwares = append(infras.Softwares[:i], infras.Softwares[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
}

func getRoleInfra(ccfg *api.ClusterConfig, ip string, delRoles uint16) *api.RoleInfra {
	var infras api.RoleInfra
	for _, r := range []uint16{api.Worker, api.Master, api.LoadBalance, api.ETCD} {
		if utils.IsType(delRoles, r) {
			roleInfra := ccfg.RoleInfra[r]
			if roleInfra == nil {
				logrus.Errorf("have not register %d roleinfra", delRoles)
				return nil
			}
			infras.OpenPorts = append(infras.OpenPorts, roleInfra.OpenPorts...)
			infras.Softwares = append(infras.Softwares, roleInfra.Softwares...)
		}
	}

	var allRoles uint16
	for _, node := range ccfg.Nodes {
		if node.Address == ip {
			allRoles = node.Type
			break
		}
	}
	remainRoles := allRoles &^ delRoles
	// if not found, it means no role remain, so delete all
	if remainRoles == 0 {
		return &infras
	}

	for _, r := range []uint16{api.Worker, api.Master, api.LoadBalance, api.ETCD} {
		if !utils.IsType(remainRoles, r) {
			continue
		}
		roleInfra := ccfg.RoleInfra[r]
		if roleInfra == nil {
			logrus.Errorf("have not register %d roleinfra", r)
			return nil
		}

		for _, software := range roleInfra.Softwares {
			deleteSoftwareIfExist(&infras, software)
		}
	}

	return &infras
}

func NodeInfrastructureDestroy(config *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}

	roleInfra := getRoleInfra(config, hostconfig.Address, hostconfig.Type)
	if roleInfra == nil {
		return fmt.Errorf("do not register %d roleinfra", hostconfig.Type)
	}

	itask := task.NewTaskIgnoreErrInstance(
		&DestroyInfraTask{
			packageSrc: &config.PackageSrc,
			roleInfra:  roleInfra,
		})

	if err := nodemanager.RunTaskOnNodes(itask, []string{hostconfig.Address}); err != nil {
		return fmt.Errorf("destroy infrastructure Task failed: %v", err)
	}

	return nil
}

type packageMD5 struct {
	MD5s map[string]string
	Lock sync.RWMutex
}

func (pm *packageMD5) getMD5(path string) (string, error) {
	pm.Lock.Lock()
	defer func() {
		pm.Lock.Unlock()
	}()

	md5str, ok := pm.MD5s[path]
	if ok {
		return md5str, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	md5str = fmt.Sprintf("%x", h.Sum(nil))
	pm.MD5s[path] = md5str

	return md5str, nil
}
