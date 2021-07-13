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
 * Create: 2021-06-21
 * Description: network functions
 ******************************************************************************/
package network

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/kubectl"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

const (
	defaultNetwork = "calico"
)

type ApplyNetworkTask struct {
	Cluster *api.ClusterConfig
}

func (ct *ApplyNetworkTask) Name() string {
	return "ApplyNetworkTask"
}

func (ct *ApplyNetworkTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	return applyNetwork(r, ct.Cluster)
}

func applyNetwork(r runner.Runner, cluster *api.ClusterConfig) error {
	plugin := defaultNetwork
	if cluster.Network.Plugin != "" {
		plugin = cluster.Network.Plugin
	}
	// TODO: network yaml maybe need to store in a excusive dir
	pluginYaml := filepath.Join(constants.DefaultK8SAddonsDir, fmt.Sprintf("%s.yaml", plugin))
	if f, ok := cluster.Network.PluginArgs[constants.NetworkPluginArgKeyYamlPath]; ok {
		pluginYaml = f
	}

	err := kubectl.OperatorByYaml(r, kubectl.ApplyOpKey, pluginYaml, cluster)
	if err != nil {
		return err
	}

	return nil
}

func SetupNetwork(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}
	t := task.NewTaskInstance(&ApplyNetworkTask{Cluster: cluster})
	var masters []string
	for _, n := range cluster.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
		}
	}

	useMaster, err := nodemanager.RunTaskOnOneNode(t, masters)
	if err != nil {
		return err
	}
	err = nodemanager.WaitNodesFinish([]string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}
	logrus.Infof("[cluster] apply network success")
	return nil
}

type CleanupNetworkTask struct {
	Cluster *api.ClusterConfig
}

func (ct *CleanupNetworkTask) Name() string {
	return "CleanupNetworkTask"
}

func (ct *CleanupNetworkTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	return deleteNetwork(r, ct.Cluster)
}

func deleteNetwork(r runner.Runner, cluster *api.ClusterConfig) error {
	plugin := defaultNetwork
	if cluster.Network.Plugin != "" {
		plugin = cluster.Network.Plugin
	}
	pluginYaml := filepath.Join(constants.DefaultK8SAddonsDir, fmt.Sprintf("%s.yaml", plugin))
	if f, ok := cluster.Network.PluginArgs[constants.NetworkPluginArgKeyYamlPath]; ok {
		pluginYaml = f
	}

	err := kubectl.OperatorByYaml(r, kubectl.DeleteOpKey, pluginYaml, cluster)
	if err != nil {
		return err
	}

	return nil
}

func CleanupNetwork(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}
	t := task.NewTaskInstance(&CleanupNetworkTask{Cluster: cluster})
	var masters []string
	for _, n := range cluster.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
		}
	}

	task.SetIgnoreErrorFlag(t)
	useMaster, err := nodemanager.RunTaskOnOneNode(t, masters)
	if err != nil {
		return err
	}
	err = nodemanager.WaitNodesFinish([]string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}
	logrus.Infof("[cluster] cleanup network success")
	return nil
}
