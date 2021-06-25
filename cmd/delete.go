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
	"strings"

	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/clusterdeployment"
)

func typeStrToInt(delType string) (uint16, error) {
	if delType == "" {
		return 0, fmt.Errorf("invalid none type")
	}
	types := strings.Split(delType, ",")
	var resType uint16
	for _, t := range types {
		typeInt, ok := toTypeInt[t]
		if !ok {
			return 0, fmt.Errorf("invalid node type %v", t)
		}
		resType |= typeInt
	}
	return resType, nil
}

func deleteCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	// TODO: support delete multi-nodes at one time
	if len(args) != 1 {
		return fmt.Errorf("delete command need exactly one argument")
	}

	opts.delName = args[0]

	conf, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load backuped deploy config failed: %v", err)
	}

	delType, err := typeStrToInt(opts.delType)
	if err != nil {
		return err
	}

	// TODO: make sure config valid
	if err = clusterdeployment.DeleteNode(toClusterdeploymentConfig(conf), opts.delName, delType); err != nil {
		return err
	}

	if err = deleteConfig(conf, opts.delType, opts.delName); err != nil {
		return fmt.Errorf("delete config from userconfig failed")
	}

	if err = backupDeployConfig(conf); err != nil {
		return err
	}

	return nil
}

func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "delete a node from cluster",
		RunE:  deleteCluster,
	}

	setupDeleteCmdOpts(deleteCmd)

	return deleteCmd
}
