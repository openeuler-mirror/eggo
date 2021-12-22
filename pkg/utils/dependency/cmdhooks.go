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
 * Author: jikui
 * Create: 2021-12-11
 * Description: eggo cmd hooks implement
 ******************************************************************************/

package dependency

import (
	"fmt"
	"path"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

type CopyHooksTask struct {
	hooks *api.ClusterHookConf
}

func (ch *CopyHooksTask) Name() string {
	return "CopyHooksTask"
}

func (ch *CopyHooksTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	dstDir := path.Join(constants.DefaultPackagePath, constants.DefaultHookPath)

	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"test -d %s || mkdir -p %s\"", dstDir, dstDir)); err != nil {
		return err
	}

	if err := r.Copy(ch.hooks.HookSrcDir, dstDir); err != nil {
		return fmt.Errorf("copy from %s to %s for %s failed:%v", ch.hooks.HookSrcDir, dstDir, hcg.Address, err)
	}

	return nil
}

func ExecuteCmdHooks(ccfg *api.ClusterConfig, nodes []*api.HostConfig, op api.HookOperator, ty api.HookType) error {
	for _, hooks := range ccfg.HooksConf {
		for _, node := range nodes {
			if !utils.IsType(node.Type, hooks.Target) {
				continue
			}

			shell := getCmdShell(hooks, hooks.Target, op, ty)
			if shell == nil {
				return nil
			}
			if err := doCopyHooks(hooks, node); err != nil {
				return err
			}
			if err := executeCmdHooks(ccfg, hooks, node, shell); err != nil {
				return err
			}
		}
	}
	return nil
}

func executeCmdHooks(ccfg *api.ClusterConfig, hooks *api.ClusterHookConf, hcf *api.HostConfig, shell []*api.PackageConfig) error {
	hookConf := &api.HookRunConfig{
		ClusterID:          ccfg.Name,
		ClusterApiEndpoint: ccfg.APIEndpoint.GetUrl(),
		ClusterConfigDir:   ccfg.ConfigDir,
		HookType:           hooks.Type,
		Operator:           hooks.Operator,
		Node:               hcf,
		HookDir:            path.Join(ccfg.PackageSrc.GetPkgDstPath(), constants.DefaultHookPath),
		Hooks:              shell,
	}

	return ExecuteHooks(hookConf)
}

func getCmdShell(hooks *api.ClusterHookConf, target uint16, op api.HookOperator, ty api.HookType) []*api.PackageConfig {
	res := make([]*api.PackageConfig, len(hooks.HookFiles))

	if hooks.Target != target || hooks.Operator != op || hooks.Type != ty {
		return nil
	}
	for i, v := range hooks.HookFiles {
		res[i] = &api.PackageConfig{
			Name:    v,
			TimeOut: "120s",
		}
	}
	return res
}

func doCopyHooks(hcc *api.ClusterHookConf, node *api.HostConfig) error {
	copyHooksTask := task.NewTaskInstance(&CopyHooksTask{
		hooks: hcc,
	})

	if err := nodemanager.RunTaskOnNodes(copyHooksTask, []string{node.Address}); err != nil {
		logrus.Errorf("Copy hooks failed with:%v", err)
		return err
	}
	return nil
}
