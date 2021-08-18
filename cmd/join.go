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

package cmd

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment"
	"isula.org/eggo/pkg/utils"
)

func checkConflict(joinYaml string, host *HostConfig, joinType string, clusterID string) error {
	if joinYaml != "" {
		if joinType != "" {
			return fmt.Errorf("conflict option --file and --type")
		}
		if host.Ip != "" {
			return fmt.Errorf("please do not specify ip address with option --file")
		}
		if host.Port != 0 {
			return fmt.Errorf("conflict option --file and --port")
		}
		if host.Name != "" {
			return fmt.Errorf("conflict option --file and --name")
		}
	} else {
		if host.Ip == "" {
			return fmt.Errorf("please specify ip address")
		}
	}

	return nil
}

func parseJoinInput(joinYaml string, host *HostConfig, joinType string, clusterID string) (*DeployConfig, error) {
	if err := checkConflict(joinYaml, host, joinType, clusterID); err != nil {
		return nil, err
	}

	if clusterID == "" {
		return nil, fmt.Errorf("please specify cluster id")
	}

	if joinType == "" {
		joinType = "worker"
	}

	var err error
	conf := &DeployConfig{}
	if joinYaml != "" {
		conf, err = loadDeployConfig(joinYaml)
		if err != nil {
			return nil, fmt.Errorf("load join config failed: %v", err)
		}
	} else {
		types := strings.Split(joinType, ",")
		for _, t := range types {
			if t == "master" {
				conf.Masters = append(conf.Masters, host)
				conf.Etcds = append(conf.Etcds, host)
			} else if t == "worker" {
				conf.Workers = append(conf.Workers, host)
			} else {
				return nil, fmt.Errorf("join type %v unsupported", t)
			}
		}
	}
	conf.ClusterID = clusterID

	if len(conf.Masters) == 0 && len(conf.Workers) == 0 {
		return nil, fmt.Errorf("no join ip address found")
	}

	return conf, nil
}

func getMergedAndDiffConfigs(conf *DeployConfig, joinConf *DeployConfig) (*DeployConfig, []*api.HostConfig, error) {
	allHostConfigs := getAllHostConfigs(conf)

	mergedConfig := *conf
	diffConfig := *conf
	diffConfig.Masters, diffConfig.Workers, diffConfig.Etcds, diffConfig.LoadBalance = nil, nil, nil, LoadBalance{}
	for i, host := range joinConf.Masters {
		if getHostConfigByIp(mergedConfig.Masters, host.Ip) != nil {
			continue
		}

		h := createHostConfig(getHostConfigByIp(allHostConfigs, host.Ip), host,
			defaultHostName(conf.ClusterID, "master", len(conf.Masters)+i))
		mergedConfig.Masters = append(mergedConfig.Masters, h)
		diffConfig.Masters = append(diffConfig.Masters, h)
		if etcd := getHostConfigByIp(conf.Etcds, host.Ip); etcd == nil {
			mergedConfig.Etcds = append(mergedConfig.Etcds, h)
			diffConfig.Etcds = append(diffConfig.Etcds, h)
		}
	}

	for i, host := range joinConf.Workers {
		if getHostConfigByIp(mergedConfig.Workers, host.Ip) != nil {
			continue
		}
		h := createHostConfig(getHostConfigByIp(allHostConfigs, host.Ip), host,
			defaultHostName(conf.ClusterID, "worker", len(conf.Workers)+i))
		mergedConfig.Workers = append(mergedConfig.Workers, h)
		diffConfig.Workers = append(diffConfig.Workers, h)
	}

	return &mergedConfig, toClusterdeploymentConfig(&diffConfig).Nodes, nil
}

func getFailedConfigs(diffConfigs []*api.HostConfig, cstatus api.ClusterStatus) []*api.HostConfig {
	var failedConfigs []*api.HostConfig
	for _, h := range diffConfigs {
		if success, ok := cstatus.StatusOfNodes[h.Address]; ok && success {
			continue
		}
		failedConfigs = append(failedConfigs, h)
	}

	return failedConfigs
}

func isFailed(failedInfos map[string]uint16, hostconfig *HostConfig, t uint16) bool {
	failedType, ok := failedInfos[hostconfig.Ip]
	if ok && utils.IsType(failedType, t) {
		return true
	}
	return false
}

func getFailedInfos(failedConfigs []*api.HostConfig) map[string]uint16 {
	mapSize := 1
	if len(failedConfigs) != 0 {
		mapSize = len(failedConfigs)
	}

	failedInfos := make(map[string]uint16, mapSize)
	for _, h := range failedConfigs {
		failedInfos[h.Address] = h.Type
	}

	return failedInfos
}

func dropFailedConfigs(conf *DeployConfig, failedConfigs []*api.HostConfig) {
	var masters []*HostConfig

	failedInfos := getFailedInfos(failedConfigs)
	for _, n := range conf.Masters {
		if isFailed(failedInfos, n, api.Master) {
			continue
		}
		masters = append(masters, n)
	}
	conf.Masters = masters

	var workers []*HostConfig
	for _, n := range conf.Workers {
		if isFailed(failedInfos, n, api.Worker) {
			continue
		}
		workers = append(workers, n)
	}
	conf.Workers = workers

	var etcds []*HostConfig
	for _, n := range conf.Etcds {
		if isFailed(failedInfos, n, api.ETCD) {
			continue
		}
		etcds = append(etcds, n)
	}
	conf.Etcds = etcds
}

func joinCluster(cmd *cobra.Command, args []string) error {
	if opts.debug {
		initLog()
	}

	if len(args) != 0 {
		opts.joinHost.Ip = args[0]
	}
	var err error

	joinConf, err := parseJoinInput(opts.joinYaml, &opts.joinHost, opts.joinType, opts.joinClusterID)
	if err != nil {
		return err
	}

	conf, err := loadDeployConfig(savedDeployConfigPath(joinConf.ClusterID))
	if err != nil {
		return fmt.Errorf("load saved deploy config failed: %v", err)
	}

	// check saved config
	if err = RunChecker(conf); err != nil {
		return err
	}

	holder, err := NewProcessPlaceHolder(eggoPlaceHolderPath(conf.ClusterID))
	if err != nil {
		return fmt.Errorf("create process holder failed: %v, mayebe other eggo is running with cluster: %s", err, conf.ClusterID)
	}
	defer holder.Remove()

	mergedConf, diffConfigs, err := getMergedAndDiffConfigs(conf, joinConf)
	if mergedConf == nil || diffConfigs == nil || err != nil {
		return fmt.Errorf("get merged and diff config failed")
	}

	// check joined config
	if err = RunChecker(mergedConf); err != nil {
		return err
	}

	cstatus, err := clusterdeployment.JoinNodes(toClusterdeploymentConfig(conf), diffConfigs)
	if err != nil {
		failedConfigs := getFailedConfigs(diffConfigs, cstatus)
		// rollback
		if err1 := clusterdeployment.DeleteNodes(toClusterdeploymentConfig(mergedConf), failedConfigs); err1 != nil {
			logrus.Errorf("delete nodes failed when join failed: %v", err1)
		}

		dropFailedConfigs(mergedConf, failedConfigs)
	}

	if err = saveDeployConfig(mergedConf, savedDeployConfigPath(joinConf.ClusterID)); err != nil {
		return err
	}

	fmt.Print(cstatus.Show())

	return nil
}

func NewJoinCmd() *cobra.Command {
	joinCmd := &cobra.Command{
		Use:   "join IP",
		Short: "join master or worker to cluster",
		RunE:  joinCluster,
	}

	setupJoinCmdOpts(joinCmd)

	return joinCmd
}
