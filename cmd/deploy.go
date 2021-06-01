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
 * Create: 2021-05-28
 * Description: eggo deploy command implement
 ******************************************************************************/

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
)

func deploy(ccfg *api.ClusterConfig) error {
	// TODO: save or load config on disk
	return clusterdeployment.CreateCluster(ccfg)
}

func deployCluster(cmd *cobra.Command, args []string) error {
	conf, err := loadDeployConfig(opts.deployConfig)
	if err != nil {
		return fmt.Errorf("load deploy config file failed: %v", err)
	}

	// TODO: make sure config valid

	if err := deploy(toClusterdeploymentConfig(conf)); err != nil {
		return err
	}

	return nil
}

func NewDeployCmd() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "deploy a kubernetes cluster",
		RunE:  deployCluster,
	}

	setupDeployCmdOpts(deployCmd)

	return deployCmd
}
