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

package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment"
	"isula.org/eggo/pkg/utils"
)

func getDelType(hostconfig *api.HostConfig) string {
	var delType string
	if utils.IsType(hostconfig.Type, api.Master) {
		delType = "master"
	}
	if utils.IsType(hostconfig.Type, api.Worker) {
		if delType != "" {
			delType += ","
		}
		delType += "worker"
	}
	return delType
}

func deleteCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	// TODO: support delete multi-nodes at one time
	if len(args) != 1 {
		return fmt.Errorf("delete command need exactly one argument")
	}

	if opts.delClusterID == "" {
		return fmt.Errorf("please specify cluster id")
	}

	// support delete master and worker at one time only currently
	opts.delName = args[0]

	conf, err := loadDeployConfig(savedDeployConfigPath(opts.delClusterID))
	if err != nil {
		return fmt.Errorf("load saved deploy config failed: %v", err)
	}
	// TODO: make sure config valid

	clusterConfig := toClusterdeploymentConfig(conf)
	h := getHostConfigByName(clusterConfig.Nodes, opts.delName)
	if h == nil {
		return fmt.Errorf("cannot found host by %v in %v", opts.delName, opts.delClusterID)
	}

	delType := getDelType(h)
	if delType == "" {
		logrus.Errorf("no master or worker found by %v in %v, ignore delete", opts.delName, opts.delClusterID)
		return nil
	}

	_, diffHostconfig, err := joinConfig(conf, delType, &HostConfig{Ip: h.Address})
	if diffHostconfig == nil || err != nil {
		return fmt.Errorf("get diff config failed: %v", err)
	}

	if err = clusterdeployment.DeleteNode(toClusterdeploymentConfig(conf), diffHostconfig); err != nil {
		return err
	}

	if err = deleteConfig(conf, delType, opts.delName); err != nil {
		return fmt.Errorf("delete config from userconfig failed")
	}

	if err = saveDeployConfig(conf, savedDeployConfigPath(opts.delClusterID)); err != nil {
		return err
	}

	return nil
}

func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "delete the master and worker from cluster",
		RunE:  deleteCluster,
	}

	setupDeleteCmdOpts(deleteCmd)

	return deleteCmd
}
