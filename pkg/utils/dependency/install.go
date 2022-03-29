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
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

func newBaseDependency(roleInfra *api.RoleInfra, packagePath string) map[string]dependency {
	packages := map[string][]*api.PackageConfig{
		"repo": {},
		"pkg":  {},
		"bin":  {},
		"file": {},
		"dir":  {},
	}

	for _, p := range roleInfra.Softwares {
		if _, exist := packages[p.Type]; !exist {
			continue
		}
		packages[p.Type] = append(packages[p.Type], p)
	}

	baseDependency := map[string]dependency{
		"repo": &dependencyRepo{
			software: packages["repo"],
		},
		"pkg": &dependencyPkg{
			srcPath:  path.Join(packagePath, constants.DefaultPkgPath),
			software: packages["pkg"],
		},
		"bin": &dependencyFileDir{
			executable: true,
			srcPath:    path.Join(packagePath, constants.DefaultBinPath),
			software:   packages["bin"],
		},
		"file": &dependencyFileDir{
			executable: false,
			srcPath:    path.Join(packagePath, constants.DefaultFilePath),
		},
		"dir": &dependencyFileDir{
			executable: false,
			srcPath:    path.Join(packagePath, constants.DefaultDirPath),
			software:   packages["dir"],
		},
	}

	return baseDependency
}

// install base dependency, include repo, pkg, bin, file, dir
func InstallBaseDependency(r runner.Runner, roleInfra *api.RoleInfra, hcf *api.HostConfig, packagePath string) error {
	baseDependency := newBaseDependency(roleInfra, packagePath)

	for _, dep := range baseDependency {
		if err := dep.Install(r); err != nil {
			logrus.Errorf("install failed for %s: %v", hcf.Address, err)
			return err
		}
	}

	return nil
}

func RemoveBaseDependency(r runner.Runner, roleInfra *api.RoleInfra, hcf *api.HostConfig, packagePath string) {
	baseDependency := newBaseDependency(roleInfra, packagePath)

	for _, dep := range baseDependency {
		if err := dep.Remove(r); err != nil {
			logrus.Errorf("uninstall failed for %s: %v", hcf.Address, err)
		}
	}
}

func getImages(workerInfra *api.RoleInfra) []*api.PackageConfig {
	images := []*api.PackageConfig{}
	for _, s := range workerInfra.Softwares {
		if s.Type == "image" {
			images = append(images, s)
		}
	}

	return images
}

// install image dependency
func InstallImageDependency(r runner.Runner, workerInfra *api.RoleInfra, packageSrc *api.PackageSrcConfig,
	runtime, runtimeClient, runtimeCommand string) error {
	images := getImages(workerInfra)
	if len(images) == 0 {
		logrus.Warn("no images load")
		return nil
	}

	logrus.Info("do load images...")

	imageDependency := &dependencyImage{
		srcPath: filepath.Join(packageSrc.GetPkgDstPath(), constants.DefaultImagePath),
		client:  runtimeClient,
		command: runtimeCommand,
		image:   images,
	}

	if err := imageDependency.Install(r); err != nil {
		if utils.IsContainerd(runtime) {
			logrus.Warnf("%s not support load images", runtime)
			return nil
		}
		return err
	}

	logrus.Info("load images success")
	return nil
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

func getShell(roleInfra *api.RoleInfra, schedule api.ScheduleType) []*api.PackageConfig {
	shell := []*api.PackageConfig{}
	for _, s := range roleInfra.Softwares {
		if s.Type == "shell" && s.Schedule == schedule {
			shell = append(shell, s)
		}
	}

	return shell
}

func ExecuteHooks(hookConf *api.HookRunConfig) error {
	if hookConf == nil || len(hookConf.Hooks) == 0 {
		return nil
	}

	var hookStr []string
	for _, h := range hookConf.Hooks {
		hookStr = append(hookStr, h.Name)
	}
	logrus.Debugf("run %s shell %v on %v\n", string(hookConf.Scheduler), hookStr, hookConf.Node.Address)

	dp := &dependencyShell{
		srcPath: hookConf.HookDir,
		shell:   hookConf.Hooks,
	}

	const envsSize = 9
	envs := make([]string, envsSize)
	envs[0] = fmt.Sprintf("EGGO_CLUSTER_ID=%s", hookConf.ClusterID)
	envs[1] = fmt.Sprintf("EGGO_CLUSTER_API_ENDPOINT=%s", hookConf.ClusterAPIEndpoint)
	envs[2] = fmt.Sprintf("EGGO_CLUSTER_CONFIG_DIR=%s", hookConf.ClusterConfigDir)
	envs[3] = fmt.Sprintf("EGGO_NODE_IP=%s", hookConf.Node.Address)
	envs[4] = fmt.Sprintf("EGGO_NODE_NAME=%s", hookConf.Node.Name)
	envs[5] = fmt.Sprintf("EGGO_NODE_ARCH=%s", hookConf.Node.Arch)
	envs[6] = fmt.Sprintf("EGGO_NODE_ROLE=%s", strings.Join(api.GetRoleString(hookConf.Node.Type), ","))
	envs[7] = fmt.Sprintf("EGGO_HOOK_TYPE=%s", hookConf.HookType)
	envs[8] = fmt.Sprintf("EGGO_OPERATOR=%s", hookConf.Operator)
	dp.envs = envs

	dependencyTask := task.NewTaskInstance(&DependencyTask{
		dp: dp,
	})

	if api.IsCleanupSchedule(hookConf.Scheduler) {
		task.SetIgnoreErrorFlag(dependencyTask)
	}
	if err := nodemanager.RunTaskOnNodes(dependencyTask, []string{hookConf.Node.Address}); err != nil {
		logrus.Errorf("Hook %s failed for %s: %v", string(api.SchedulePreJoin), hookConf.Node.Address, err)
		return err
	}

	return nil
}

func executeShell(ccfg *api.ClusterConfig, role uint16, hcf *api.HostConfig, schedule api.ScheduleType) error {
	shell := getShell(ccfg.RoleInfra[role], schedule)
	if len(shell) == 0 {
		return nil
	}

	htype := api.PreHookType
	if strings.HasPrefix(string(schedule), "post") {
		htype = api.PostHookType
	}
	oper := api.HookOpJoin
	if strings.HasSuffix(string(schedule), "cleanup") {
		oper = api.HookOpCleanup
	}

	hookConf := &api.HookRunConfig{
		ClusterID:          ccfg.Name,
		ClusterAPIEndpoint: ccfg.APIEndpoint.GetURL(),
		ClusterConfigDir:   ccfg.ConfigDir,
		HookType:           htype,
		Operator:           oper,
		Node:               hcf,
		HookDir:            path.Join(ccfg.PackageSrc.GetPkgDstPath(), constants.DefaultFilePath),
		Hooks:              shell,
	}

	return ExecuteHooks(hookConf)
}

func HookSchedule(ccfg *api.ClusterConfig, nodes []*api.HostConfig, role []uint16, schedule api.ScheduleType) error {
	for _, n := range nodes {
		for _, r := range role {
			if !utils.IsType(n.Type, r) {
				continue
			}

			if err := executeShell(ccfg, r, n, schedule); err != nil {
				if api.IsCleanupSchedule(schedule) {
					logrus.Errorf("execute shell failed for %s at %s: %v", n.Address, string(schedule), err)
				} else {
					return err
				}
			}
		}
	}

	return nil
}
