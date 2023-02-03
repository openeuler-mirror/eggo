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
	"isula.org/eggo/pkg/utils"
)

func removeFailedNodes(cstatus *api.ClusterStatus, conf *DeployConfig) {
	// if partial success, just update config of cluster, remove failed nodes
	if cstatus.FailureCnt == 0 {
		return
	}

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

	if err := saveDeployConfig(conf, savedDeployConfigPath(conf.ClusterID)); err != nil {
		fmt.Printf("Warn: failed to save config!!!\n")
		fmt.Printf("	you can call \"eggo delete --id %s [failed nodes id]\" to remove failed node from your cluster.\n", conf.ClusterID)
		return
	}
	fmt.Printf("update config of cluster: %s", conf.ClusterID)
}

func deploy(conf *DeployConfig) error {
	if err := saveDeployConfig(conf, savedDeployConfigPath(conf.ClusterID)); err != nil {
		return fmt.Errorf("save deploy config failed: %v", err)
	}

	hooksConf, err := getClusterHookConf(api.HookOpDeploy)
	if err != nil {
		return fmt.Errorf("get cmd hooks config failed:%v", err)
	}
	ccfg := toClusterdeploymentConfig(conf, hooksConf)

	cstatus, err := clusterdeployment.CreateCluster(ccfg, opts.deployEnableRollback)
	if err != nil {
		return err
	}

	// if disable rollback, just ignore error, and wait user to cleanup
	if opts.deployEnableRollback {
		removeFailedNodes(&cstatus, conf)
	} else {
		if cstatus.FailureCnt > 0 {
			fmt.Printf("Warn: you can call \"eggo delete --id %s [failed nodes id]\" to remove failed node from your cluster.\n", conf.ClusterID)
		}
	}

	fmt.Print(cstatus.Show())

	if cstatus.Working {
		fmt.Printf("To start using cluster: %s, you need following as a regular user:\n\n", ccfg.Name)
		fmt.Printf("\texport KUBECONFIG=%s/admin.conf\n\n", api.GetClusterHomePath(ccfg.Name))
	}

	return err
}

func checkClusterExist(ClusterID string) error {
	clusterHomeDir := api.GetClusterHomePath(ClusterID)
	if exist, err := utils.CheckPathExist(clusterHomeDir); err != nil || exist {
		return fmt.Errorf("cluster: %s exist, please check it", ClusterID)
	}
	return nil
}

func deployCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}
	var err error

	conf, err := loadDeployConfig(opts.deployConfig)
	if err != nil {
		return fmt.Errorf("load deploy config file failed: %v", err)
	}

	if err = checkCmdHooksParameter(opts.clusterPrehook, opts.clusterPosthook); err != nil {
		return err
	}
	if err = RunChecker(conf); err != nil {
		return err
	}

	// check cluster home dir
	if err = checkClusterExist(conf.ClusterID); err != nil {
		return err
	}

	holder, err := NewProcessPlaceHolder(eggoPlaceHolderPath(conf.ClusterID))
	if err != nil {
		return fmt.Errorf("create process holder failed: %v, mayebe other eggo is running with cluster: %s", err, conf.ClusterID)
	}
	defer func() {
		if terr := holder.Remove(); terr != nil {
			fmt.Printf("remove process place holder failed: %v", terr)
		}
	}()

	if err = deploy(conf); err != nil {
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
