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
 * Description: eggo cleanup command implement
 ******************************************************************************/

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment"
)

func cleanup(ccfg *api.ClusterConfig) error {
	return clusterdeployment.RemoveCluster(ccfg)
}

func cleanupCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	if opts.cleanupConfig == "" && opts.cleanupClusterID == "" {
		return fmt.Errorf("please specify cluster id")
	}

	confPath := opts.cleanupConfig
	if confPath == "" {
		confPath = savedDeployConfigPath(opts.cleanupClusterID)
		_, err := os.Stat(confPath)
		if os.IsNotExist(err) {
			confPath = defaultDeployConfigPath()
		} else if err != nil {
			return fmt.Errorf("stat %v failed: %v", confPath, err)
		}
	}

	conf, err := loadDeployConfig(confPath)
	if err != nil {
		return fmt.Errorf("load deploy config file %v failed: %v", confPath, err)
	}

	// TODO: make sure config valid

	if err = cleanup(toClusterdeploymentConfig(conf)); err != nil {
		return err
	}

	return nil
}

func NewCleanupCmd() *cobra.Command {
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup a kubernetes cluster",
		RunE:  cleanupCluster,
	}

	setupCleanupCmdOpts(cleanupCmd)

	return cleanupCmd
}
