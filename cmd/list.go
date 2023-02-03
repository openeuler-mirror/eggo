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
 * Author: haozi007
 * Create: 2021-09-09
 * Description: eggo list command implement
 ******************************************************************************/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
)

type clusterInfo struct {
	name      string
	masterCnt int
	workerCnt int
	status    string
}

var (
	infos []clusterInfo
)

func addClusterInfo(name string, conf *DeployConfig, err error) {
	info := clusterInfo{
		name: name,
	}
	if err != nil {
		info.status = "unknow"
		logrus.Debugf("%s: %s", info.name, err.Error())
		infos = append(infos, info)
		return
	}
	if conf.Masters != nil {
		info.masterCnt = len(conf.Masters)
	}
	if conf.Workers != nil {
		info.workerCnt = len(conf.Workers)
	}

	if terr := RunChecker(conf); terr != nil {
		info.status = "broken"
		logrus.Debugf("%s: %s", info.name, terr.Error())
	} else {
		info.status = "success"
	}

	infos = append(infos, info)
}

func checkFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !info.IsDir() {
		logrus.Debugf("ingore non-dir: %q", path)
		return nil
	}

	if path == api.GetEggoClusterPath() {
		return nil
	}

	conf, err := loadDeployConfig(savedDeployConfigPath(info.Name()))
	addClusterInfo(info.Name(), conf, err)
	return filepath.SkipDir
}

func showClustersInfo() {
	maxLen := 8
	for _, info := range infos {
		if len(info.name) > maxLen {
			maxLen = len(info.name)
		}
	}
	fmt.Printf("Name%*s\tMasters\tWorkers\tStatus\n", maxLen, "")
	for _, info := range infos {
		fmt.Printf("%s%*s\t%d\t%d\t%s\n", info.name, len(info.name)-maxLen, "", info.masterCnt, info.workerCnt, info.status)
	}
}

func listClusters(cmd *cobra.Command, args []string) error {
	infos = nil
	if opts.debug {
		initLog()
	}

	eggoDir := api.GetEggoClusterPath()

	if err := filepath.Walk(eggoDir, checkFile); err != nil {
		logrus.Debugf("walk eggo cluster dir: %s, err: %v\n", eggoDir, err)
	}

	showClustersInfo()

	return nil
}

func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "list clusters which manager by eggo",
		RunE:  listClusters,
	}

	return listCmd
}
