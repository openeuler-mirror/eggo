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

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"github.com/sirupsen/logrus"
)

const (
	prmT   = "sudo -E /bin/sh -c \"which apt 1>/dev/null ; if [ $? -eq 0 ]; then echo apt ; elif which yum 1>/dev/null ; if [ $? -eq 0 ]; then echo yum ; fi\""
	pmT    = "sudo -E /bin/sh -c \"which dpkg 1>/dev/null ; if [ $? -eq 0 ]; then echo dpkg ; elif which rpm 1>/dev/null ; if [ $? -eq 0 ]; then echo rpm ; fi\""
	tmpDir = "/etc/.eggo/"
)

type Dependences interface {
	Check(r runner.Runner, hcg *api.HostConfig) error
	DoInstall(r runner.Runner, hcg *api.HostConfig) error
	PostInstall(r runner.Runner) error
	Remove(r runner.Runner, hcg *api.HostConfig) error
}

type InstallByRepo struct {
	prmanager   string
	dependences []string
}

func NewInstallByRepo(dependences []string) Dependences {
	return &InstallByRepo{
		dependences: dependences,
	}
}

func (ir *InstallByRepo) Check(r runner.Runner, hcg *api.HostConfig) error {
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

func (ir *InstallByRepo) DoInstall(r runner.Runner, hcg *api.HostConfig) error {
	join := strings.Join(ir.dependences, " ")
	_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s install -y %s\"", ir.prmanager, join))
	if err != nil {
		return fmt.Errorf("install dependences by repo failed for %s: %v", hcg.Address, err)
	}

	return nil
}

func (ir *InstallByRepo) PostInstall(r runner.Runner) error {
	return nil
}

func (ir *InstallByRepo) Remove(r runner.Runner, hcg *api.HostConfig) error {
	join := strings.Join(ir.dependences, " ")
	_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s remove -y %s\"", ir.prmanager, join))
	if err != nil {
		return fmt.Errorf("remove dependences by repo failed for %s: %v", hcg.Address, err)
	}

	return nil
}

type InstallByLocal struct {
	pmanager string
	pcfg     *api.PackageSrcConfig
	pkg      []string
	binary   map[string]string
}

func NewInstallByLocal(pcfg *api.PackageSrcConfig, pkg []string, binary map[string]string) Dependences {
	return &InstallByLocal{
		pcfg:   pcfg,
		pkg:    pkg,
		binary: binary,
	}
}

func (il *InstallByLocal) Check(r runner.Runner, hcg *api.HostConfig) error {
	pmanager, err := r.RunCommand(pmT)
	if err != nil {
		return fmt.Errorf("get package manager failed: %v", err)
	}
	if pmanager == "" {
		return fmt.Errorf("no package manager for %s", hcg.Address)
	}
	il.pmanager = pmanager

	if il.pcfg == nil {
		return fmt.Errorf("empty package source config")
	}
	if err := copySource(r, hcg, il.pcfg); err != nil {
		return err
	}

	return nil
}

func (il *InstallByLocal) DoInstall(r runner.Runner, hcg *api.HostConfig) error {
	if err := installByLocalPkg(r, hcg, il.pmanager, il.pkg); err != nil {
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

func (il *InstallByLocal) Remove(r runner.Runner, hcg *api.HostConfig) error {
	if err := removePkg(r, hcg, il.pmanager, il.pkg); err != nil {
		return err
	}

	if err := removeBinary(r, hcg, il.binary); err != nil {
		return err
	}

	return nil
}

func copySource(r runner.Runner, hcg *api.HostConfig, pcfg *api.PackageSrcConfig) error {
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

func installByLocalPkg(r runner.Runner, hcg *api.HostConfig, pmanager string, pkg []string) error {
	if len(pkg) == 0 {
		return nil
	}

	var pmCommand string
	if pmanager == "rpm" {
		pmCommand = "rpm -ivh"
	} else {
		pmCommand = "dpkg -i"
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for _, p := range pkg {
		sb.WriteString(fmt.Sprintf("%s %s/%s* && ", pmCommand, tmpDir, p))
	}

	sb.WriteString("echo success\"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("install local pkg failed: %v", err)
	}

	return nil
}

func removePkg(r runner.Runner, hcg *api.HostConfig, pmanager string, pkg []string) error {
	if len(pkg) == 0 {
		return nil
	}

	var sb strings.Builder
	if pmanager == "rpm" {
		sb.WriteString("sudo -E /bin/sh -c \"yum remove -y ")
	} else {
		sb.WriteString("sudo -E /bin/sh -c \"apt remove -y ")
	}

	for _, p := range pkg {
		sb.WriteString(fmt.Sprintf("%s* && ", p))
	}

	sb.WriteString("echo success\"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("remove dependences by pkg failed: %v", err)
	}

	return nil
}

func installByLocalBinary(r runner.Runner, hcg *api.HostConfig, binary map[string]string) error {
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

func removeBinary(r runner.Runner, hcg *api.HostConfig, binary map[string]string) error {
	if len(binary) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for b, d := range binary {
		sb.WriteString(fmt.Sprintf("rm -f %s/%s && ", d, b))
	}

	sb.WriteString("echo success\"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("remove binary failed: %v", err)
	}

	return nil
}

func separateDependences(hcg *api.HostConfig) ([]string, []string, map[string]string, error) {
	// repo, pkg, binary
	repo := []string{}
	pkg := []string{}
	binary := make(map[string]string)

	for p, c := range hcg.Packages {
		switch c.Type {
		case "repo":
			repo = append(repo, p)
		case "pkg":
			pkg = append(pkg, p)
		case "binary":
			if c.Dst == "" {
				return nil, nil, nil, fmt.Errorf("no dst for binary %s", p)
			}
			binary[p] = c.Dst
		default:
			return nil, nil, nil, fmt.Errorf("invalid type %s for %s", c.Type, p)
		}
	}

	return repo, pkg, binary, nil
}

func doInstallDependences(r runner.Runner, hcg *api.HostConfig, dp Dependences) error {
	if err := dp.Check(r, hcg); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := dp.DoInstall(r, hcg); err != nil {
		logrus.Errorf("do install failed: %v", err)
		return err
	}

	if err := dp.PostInstall(r); err != nil {
		logrus.Errorf("post install failed: %v", err)
		return err
	}

	return nil
}

func InstallDependences(r runner.Runner, hcg *api.HostConfig, pcfg *api.PackageSrcConfig) error {
	repo, pkg, binary, err := separateDependences(hcg)
	if err != nil {
		return err
	}

	if len(repo) != 0 {
		ir := NewInstallByRepo(repo)
		if err := doInstallDependences(r, hcg, ir); err != nil {
			return err
		}
	}

	if len(pkg) != 0 || len(binary) != 0 {
		il := NewInstallByLocal(pcfg, pkg, binary)
		if err := doInstallDependences(r, hcg, il); err != nil {
			return err
		}
	}

	return nil
}

func doRemoveDependences(r runner.Runner, hcg *api.HostConfig, dp Dependences) error {
	if err := dp.Check(r, hcg); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := dp.Remove(r, hcg); err != nil {
		logrus.Errorf("remove dependences failed: %v", err)
		return err
	}

	return nil
}

func RemoveDependences(r runner.Runner, hcg *api.HostConfig) error {
	repo, pkg, binary, err := separateDependences(hcg)
	if err != nil {
		return err
	}

	if len(repo) != 0 {
		ir := NewInstallByRepo(repo)
		if err := doRemoveDependences(r, hcg, ir); err != nil {
			return err
		}
	}

	if len(pkg) != 0 || len(binary) != 0 {
		// do remove dependences without package source config
		il := NewInstallByLocal(nil, pkg, binary)
		if err := doRemoveDependences(r, hcg, il); err != nil {
			return err
		}
	}

	return nil
}

func CheckDependences(r runner.Runner, softwares []string) error {
	for _, s := range softwares {
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"which %s\"", s))
		if err != nil {
			logrus.Errorf("check software: %s, failed: %v\n", s, err)
			return err
		}
		logrus.Debugf("check software: %s success\n", s)
	}
	return nil
}
