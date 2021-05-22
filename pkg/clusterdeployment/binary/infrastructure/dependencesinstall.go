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
 * Create: 2021-05-22
 * Description: eggo dependences install implement
 ******************************************************************************/

package infrastructure

import (
	"fmt"
	"strings"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
)

const (
	prmT   = "sudo -E /bin/sh -c \"which apt 1>/dev/null ; if [ $? -eq 0 ]; then echo apt ; elif which yum 1>/dev/null ; if [ $? -eq 0 ]; then echo yum ; fi\""
	pmT    = "sudo -E /bin/sh -c \"which dpkg 1>/dev/null ; if [ $? -eq 0 ]; then echo dpkg ; elif which rpm 1>/dev/null ; if [ $? -eq 0 ]; then echo rpm ; fi\""
	tmpDir = "/etc/.eggo/"
)

type DependencesInstall interface {
	PreInstall(r runner.Runner, hcg *clusterdeployment.HostConfig) error
	DoInstall(r runner.Runner, hcg *clusterdeployment.HostConfig) error
	PostInstall(r runner.Runner) error
}

type InstallByRepo struct {
	prmanager   string
	dependences []string
}

func NewInstallByRepo(dependences []string) DependencesInstall {
	return &InstallByRepo{
		dependences: dependences,
	}
}

func (ir *InstallByRepo) PreInstall(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	prmanager, err := r.RunCommand(prmT)
	if err != nil {
		return fmt.Errorf("get repo package manager failed: %v", err)
	}

	if len(prmanager) == 0 {
		return fmt.Errorf("no package repo manager for %s", hcg.Address)
	}

	ir.prmanager = prmanager
	return nil
}

func (ir *InstallByRepo) DoInstall(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	join := strings.Join(ir.dependences, " ")
	_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s -y %s\"", ir.prmanager, join))
	if err != nil {
		return fmt.Errorf("install dependences by repo failed for %s: %v", hcg.Address, err)
	}

	return nil
}

func (ir *InstallByRepo) PostInstall(r runner.Runner) error {
	return nil
}

type InstallByLocal struct {
	pmanager string
	pcfg     *clusterdeployment.PackageSrcConfig
	pkg      []string
	binary   map[string]string
}

func NewInstallByLocal(pcfg *clusterdeployment.PackageSrcConfig, pkg []string, binary map[string]string) DependencesInstall {
	return &InstallByLocal{
		pcfg:   pcfg,
		pkg:    pkg,
		binary: binary,
	}
}

func (il *InstallByLocal) PreInstall(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	pmanager, err := r.RunCommand(pmT)
	if err != nil {
		return fmt.Errorf("get package manager failed: %v", err)
	}

	if pmanager == "" {
		return fmt.Errorf("no package manager for %s", hcg.Address)
	}
	il.pmanager = pmanager

	if err := copySource(r, hcg, il.pcfg); err != nil {
		return err
	}

	return nil
}

func (il *InstallByLocal) DoInstall(r runner.Runner, hcg *clusterdeployment.HostConfig) error {
	if err := installByLocalPkg(r, hcg, il.pkg); err != nil {
		return err
	}

	if err := installByLocalBinary(r, hcg, il.binary); err != nil {
		return err
	}

	return nil
}

func (il *InstallByLocal) PostInstall(r runner.Runner) error {
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"rm -rf %s\"", tmpDir)); err != nil {
		return err
	}

	return nil
}

func copySource(r runner.Runner, hcg *clusterdeployment.HostConfig, pcfg *clusterdeployment.PackageSrcConfig) error {
	var src string
	if utils.IsX86Arch(hcg.Arch) {
		src = pcfg.X86Src
	} else if utils.IsArmArch(hcg.Arch) {
		src = pcfg.ArmSrc
	}
	if src == "" {
		return fmt.Errorf("invalid srcpath for %s", hcg.Address)
	}

	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s\"", tmpDir)); err != nil {
		return err
	}

	if err := r.Copy(src, tmpDir); err != nil {
		return fmt.Errorf("copy from %s to %s for %s failed: %v", src, tmpDir, hcg.Address, err)
	}

	switch pcfg.Type {
	case "tar.gz":
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"cd %s && tar -zxvf *.tar.gz\"", tmpDir))
		if err != nil {
			return fmt.Errorf("uncompress %s failed for %s: %v", src, hcg.Address, err)
		}
	default:
		return fmt.Errorf("cannot support uncompress %s", pcfg.Type)
	}

	return nil
}

func installByLocalPkg(r runner.Runner, hcg *clusterdeployment.HostConfig, pkg []string) error {
	if len(pkg) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for _, p := range pkg {
		sb.WriteString(fmt.Sprintf("rpm -ivh %s/%s* && ", tmpDir, p))
	}

	sb.WriteString("echo success\"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("install local pkg failed: %v", err)
	}

	return nil
}

func installByLocalBinary(r runner.Runner, hcg *clusterdeployment.HostConfig, binary map[string]string) error {
	if len(binary) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for b, d := range binary {
		sb.WriteString(fmt.Sprintf("cp %s/%s* %s && ", tmpDir, b, d))
	}

	sb.WriteString("echo success\"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("cp binary failed: %v", err)
	}

	return nil
}
