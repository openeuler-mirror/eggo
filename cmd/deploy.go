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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment"
)

func deploy(conf *DeployConfig) error {
	if err := saveDeployConfig(conf, savedDeployConfigPath(conf.ClusterID)); err != nil {
		return fmt.Errorf("save deploy config failed: %v", err)
	}

	ccfg := toClusterdeploymentConfig(conf)

	cstatus, err := clusterdeployment.CreateCluster(ccfg)
	if err != nil {
		return err
	}

	if cstatus.FailureCnt > 0 {
		// if partial success, just update config of cluster, remove failed nodes
		var tmp []*HostConfig
		for _, n := range conf.Masters {
			if success, ok := cstatus.StatusOfNodes[n.Ip]; ok && !success {
				continue
			}
			tmp = append(tmp, n)
		}
		conf.Masters = tmp

		tmp = nil
		for _, n := range conf.Workers {
			if success, ok := cstatus.StatusOfNodes[n.Ip]; ok && !success {
				continue
			}
			tmp = append(tmp, n)
		}
		conf.Workers = tmp

		tmp = nil
		for _, n := range conf.Etcds {
			if success, ok := cstatus.StatusOfNodes[n.Ip]; ok && !success {
				continue
			}
			tmp = append(tmp, n)
		}
		conf.Etcds = tmp

		err = saveDeployConfig(conf, savedDeployConfigPath(conf.ClusterID))
		if err != nil {
			fmt.Printf("")
			clusterdeployment.RemoveCluster(ccfg)
			return fmt.Errorf("update config of cluster failed: %v", err)
		}
		fmt.Printf("update config of cluster: %s", conf.ClusterID)
	}

	fmt.Print(cstatus.Show())

	if cstatus.Working {
		fmt.Printf("To start using cluster: %s, you need following as a regular user:\n\n", ccfg.Name)
		fmt.Printf("\texport KUBECONFIG=%s/admin.conf\n\n", api.GetClusterHomePath(ccfg.Name))
	}

	return err
}

func deployCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	conf, err := loadDeployConfig(opts.deployConfig)
	if err != nil {
		return fmt.Errorf("load deploy config file failed: %v", err)
	}

	// TODO: make sure config valid

	if err := deploy(conf); err != nil {
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
