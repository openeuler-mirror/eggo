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
 * Create: 2021-06-25
 * Description: eggo install implement
 ******************************************************************************/

package dependency

import (
	"fmt"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/runner"
)

const (
	PrmTest = "if [ x != x$(which apt 2>/dev/null) ]; then echo apt ; elif [ x != x$(which yum 2>/dev/null) ]; then echo yum ; fi"
	PmTest  = "if [ x != x$(which dpkg 2>/dev/null) ]; then echo dpkg ; elif [ x != x$(which rpm 2>/dev/null) ]; then echo rpm ; fi"
)

func installRepo(r runner.Runner, software []*api.PackageConfig, hcf *api.HostConfig) error {
	if len(software) == 0 {
		return nil
	}

	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", PrmTest))
	if err != nil {
		logrus.Errorf("package repo manager test failed: %v", err)
		return err
	}

	var dp dependency
	if strings.Contains(output, "apt") {
		dp = &dependencyApt{
			software: software,
		}
	} else if strings.Contains(output, "yum") {
		dp = &dependencyYum{
			software: software,
		}
	}

	if dp == nil {
		return fmt.Errorf("invalid package repo manager %s", output)
	}

	if err := dp.Install(r); err != nil {
		logrus.Errorf("install failed for %s: %v", hcf.Address, err)
		return err
	}

	return nil
}

func installPkg(r runner.Runner, software []*api.PackageConfig, hcf *api.HostConfig, packagePath string) error {
	if len(software) == 0 {
		return nil
	}

	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", PmTest))
	if err != nil {
		logrus.Errorf("package manager test failed: %v", err)
		return err
	}

	var dp dependency
	if strings.Contains(output, "dpkg") {
		dp = &dependencyDeb{
			srcPath:  path.Join(packagePath, constants.DefaultPkgPath),
			software: software,
		}
	} else if strings.Contains(output, "rpm") {
		dp = &dependencyRpm{
			srcPath:  path.Join(packagePath, constants.DefaultPkgPath),
			software: software,
		}
	}

	if dp == nil {
		return fmt.Errorf("invalid package manager %s", output)
	}

	if err := dp.Install(r); err != nil {
		logrus.Errorf("install failed for %s: %v", hcf.Address, err)
		return err
	}

	return nil
}

func installFD(r runner.Runner, bin, file, dir []*api.PackageConfig, hcf *api.HostConfig, packagePath string) error {
	dp := []dependency{}

	if len(bin) != 0 {
		dp = append(dp, &dependencyFD{
			srcPath:  path.Join(packagePath, constants.DefaultBinPath),
			software: bin,
		})
	}

	if len(file) != 0 {
		dp = append(dp, &dependencyFD{
			srcPath:  path.Join(packagePath, constants.DefaultFilePath),
			software: file,
		})
	}

	if len(dir) != 0 {
		dp = append(dp, &dependencyFD{
			srcPath:  path.Join(packagePath, constants.DefaultDirPath),
			software: dir,
		})
	}

	if len(dp) == 0 {
		return nil
	}

	for _, d := range dp {
		if err := d.Install(r); err != nil {
			logrus.Errorf("install failed for %s: %v", hcf.Address, err)
			return err
		}
	}

	return nil
}

func uninstallRepo(r runner.Runner, software []*api.PackageConfig, hcf *api.HostConfig) error {
	if len(software) == 0 {
		return nil
	}

	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", PrmTest))
	if err != nil {
		logrus.Errorf("package repo manager test failed: %v", err)
		return err
	}

	var dp dependency
	if strings.Contains(output, "apt") {
		dp = &dependencyApt{
			software: software,
		}
	} else if strings.Contains(output, "yum") {
		dp = &dependencyYum{
			software: software,
		}
	}

	if dp == nil {
		return fmt.Errorf("invalid package repo manager %s", output)
	}

	if err := dp.Remove(r); err != nil {
		logrus.Errorf("uninstall failed for %s: %v", hcf.Address, err)
		return err
	}

	return nil
}

func uninstallPkg(r runner.Runner, software []*api.PackageConfig, hcf *api.HostConfig, packagePath string) error {
	if len(software) == 0 {
		return nil
	}

	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", PmTest))
	if err != nil {
		logrus.Errorf("package manager test failed: %v", err)
		return err
	}

	var dp dependency
	if strings.Contains(output, "dpkg") {
		dp = &dependencyDeb{
			srcPath:  path.Join(packagePath, constants.DefaultPkgPath),
			software: software,
		}
	} else if strings.Contains(output, "rpm") {
		dp = &dependencyRpm{
			srcPath:  path.Join(packagePath, constants.DefaultPkgPath),
			software: software,
		}
	}

	if dp == nil {
		return fmt.Errorf("invalid package manager %s", output)
	}

	if err := dp.Remove(r); err != nil {
		logrus.Errorf("uninstall failed for %s: %v", hcf.Address, err)
		return err
	}

	return nil
}

func uninstallFD(r runner.Runner, bin, file, dir []*api.PackageConfig, hcf *api.HostConfig) error {
	dp := []dependency{}

	if len(bin) != 0 {
		dp = append(dp, &dependencyFD{
			software: bin,
		})
	}

	if len(file) != 0 {
		dp = append(dp, &dependencyFD{
			software: file,
		})
	}

	if len(dir) != 0 {
		dp = append(dp, &dependencyFD{
			software: dir,
		})
	}

	if len(dp) == 0 {
		return nil
	}

	for _, d := range dp {
		if err := d.Remove(r); err != nil {
			logrus.Errorf("uninstall failed for %s: %v", hcf.Address, err)
			return err
		}
	}

	return nil
}

func separateSofeware(softwares []*api.PackageConfig) ([]*api.PackageConfig, []*api.PackageConfig, []*api.PackageConfig, []*api.PackageConfig, []*api.PackageConfig) {
	repo := []*api.PackageConfig{}
	pkg := []*api.PackageConfig{}
	bin := []*api.PackageConfig{}
	file := []*api.PackageConfig{}
	dir := []*api.PackageConfig{}

	for _, p := range softwares {
		switch p.Type {
		case "repo":
			repo = append(repo, p)
		case "pkg":
			pkg = append(pkg, p)
		case "bin":
			bin = append(bin, p)
		case "file":
			file = append(file, p)
		case "dir":
			dir = append(dir, p)
		}
	}

	return repo, pkg, bin, file, dir
}

func InstallDependency(r runner.Runner, roleInfra *api.RoleInfra, hcf *api.HostConfig, packagePath string) error {
	repo, pkg, bin, file, dir := separateSofeware(roleInfra.Softwares)

	if err := installRepo(r, repo, hcf); err != nil {
		return fmt.Errorf("install repo failed: %v", err)
	}

	if err := installPkg(r, pkg, hcf, packagePath); err != nil {
		return fmt.Errorf("install pkg failed: %v", err)
	}

	if err := installFD(r, bin, file, dir, hcf, packagePath); err != nil {
		return fmt.Errorf("install file failed: %v", err)
	}

	return nil
}

func RemoveDependency(r runner.Runner, roleInfra *api.RoleInfra, hcf *api.HostConfig, packagePath string) {
	repo, pkg, bin, file, dir := separateSofeware(roleInfra.Softwares)

	if err := uninstallRepo(r, repo, hcf); err != nil {
		logrus.Errorf("uninstall repo failed: %v", err)
	}

	if err := uninstallPkg(r, pkg, hcf, packagePath); err != nil {
		logrus.Errorf("uninstall pkg failed: %v", err)
	}

	if err := uninstallFD(r, bin, file, dir, hcf); err != nil {
		logrus.Errorf("uninstall file failed: %v", err)
	}
}

func CheckDependency(r runner.Runner, softwares []string) error {
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
