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
 * Author: wangfengtu
 * Create: 2021-06-25
 * Description: eggo delete command implement
 ******************************************************************************/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment"
)

func splitDeletedConfigs(hosts []*HostConfig, delNames []string) ([]*HostConfig, []*HostConfig) {
	var diff []*HostConfig
	var deleted []*HostConfig
	for _, h := range hosts {
		found := false
		for _, delName := range delNames {
			if delName == h.Name || delName == h.Ip {
				found = true
				break
			}
		}

		if found {
			diff = append(diff, h)
		} else {
			deleted = append(deleted, h)
		}
	}

	return diff, deleted
}

func getDeletedAndDiffConfigs(conf *DeployConfig, delNames []string) (*DeployConfig, []*api.HostConfig, error) {
	if len(conf.Masters) == 0 {
		return nil, nil, fmt.Errorf("invalid cluster config, no master found")
	}

	deletedConfig := *conf
	deletedConfig.Masters, deletedConfig.Workers, deletedConfig.Etcds = nil, nil, nil
	diffConfig := *conf
	diffConfig.Masters, diffConfig.Workers, diffConfig.Etcds, diffConfig.LoadBalance = nil, nil, nil, LoadBalance{}
	diffConfig.Masters, deletedConfig.Masters = splitDeletedConfigs(conf.Masters, delNames)
	diffConfig.Etcds, deletedConfig.Etcds = splitDeletedConfigs(conf.Etcds, delNames)
	diffConfig.Workers, deletedConfig.Workers = splitDeletedConfigs(conf.Workers, delNames)

	if len(deletedConfig.Masters) == 0 || conf.Masters[0].Ip != deletedConfig.Masters[0].Ip {
		return nil, nil, fmt.Errorf("forbidden to delete first master")
	}

	clusterConfig := toClusterdeploymentConfig(&diffConfig, nil)
	if len(clusterConfig.Nodes) == 0 {
		return nil, nil, fmt.Errorf("no valid ip or name found")
	}

	return &deletedConfig, clusterConfig.Nodes, nil
}

func deleteCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	if len(args) == 0 {
		return fmt.Errorf("delete command need at least one argument")
	}

	if opts.delClusterID == "" {
		return fmt.Errorf("please specify cluster id")
	}

	conf, err := loadDeployConfig(savedDeployConfigPath(opts.delClusterID))
	if err != nil {
		return fmt.Errorf("load saved deploy config failed: %v", err)
	}

	if err := checkCmdHooksParameter(opts.prehook, opts.posthook); err != nil {
		return err
	}
	// check saved deploy config
	if err = RunChecker(conf); err != nil {
		return err
	}

	hooksConf, err := getClusterHookConf(api.HookOpDelete)
	if err != nil {
		return fmt.Errorf("get cmd hooks config failed:%v", err)
	}

	holder, err := NewProcessPlaceHolder(eggoPlaceHolderPath(conf.ClusterID))
	if err != nil {
		return fmt.Errorf("create process holder failed: %v, mayebe other eggo is running with cluster: %s", err, conf.ClusterID)
	}
	defer holder.Remove()

	deletedConfig, diffHostconfigs, err := getDeletedAndDiffConfigs(conf, args)
	if err != nil {
		return fmt.Errorf("get deleted and diff config failed: %v", err)
	}

	// check deleted config
	if err = RunChecker(deletedConfig); err != nil {
		return err
	}

	if err = clusterdeployment.DeleteNodes(toClusterdeploymentConfig(conf, hooksConf), diffHostconfigs); err != nil {
		return err
	}

	if err = saveDeployConfig(deletedConfig, savedDeployConfigPath(opts.delClusterID)); err != nil {
		return err
	}

	return nil
}

func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete NAME [NAME...]",
		Short: "delete nodes from cluster",
		RunE:  deleteCluster,
	}

	setupDeleteCmdOpts(deleteCmd)

	return deleteCmd
}
