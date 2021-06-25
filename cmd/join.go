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
 * Description: eggo join command implement
 ******************************************************************************/

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment"
)

func join(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	return clusterdeployment.JoinNode(conf, hostconfig)
}

func joinCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	if len(args) != 1 {
		return fmt.Errorf("join command need exactly one argument")
	}

	opts.joinHost.Ip = args[0]

	conf, err := loadBackupedDeployConfig()
	if err != nil {
		return fmt.Errorf("load backuped deploy config failed: %v", err)
	}

	// TODO: make sure config valid

	hostconfig := joinConfig(conf, opts.joinType, &opts.joinHost)
	if hostconfig == nil {
		return fmt.Errorf("join userconfig failed")
	}

	if err = join(toClusterdeploymentConfig(conf), hostconfig); err != nil {
		return err
	}

	if err = backupDeployConfig(conf); err != nil {
		return err
	}

	return nil
}

func NewJoinCmd() *cobra.Command {
	joinCmd := &cobra.Command{
		Use:   "join NAME",
		Short: "join a node to cluster",
		RunE:  joinCluster,
	}

	setupJoinCmdOpts(joinCmd)

	return joinCmd
}
