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
 * Create: 2021-06-29
 * Description: eggo cleanup etcd member binary implement
 ******************************************************************************/

package cleanupcluster

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/etcdcluster"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

var (
	EtcdService = []string{"etcd"}
)

type cleanupEtcdMemberTask struct {
	ccfg *api.ClusterConfig
}

func (t *cleanupEtcdMemberTask) Name() string {
	return "cleanupEtcdMemberTask"
}

func getEtcdDataDir(dataDir string) string {
	if dataDir != "" {
		return dataDir
	}
	return etcdcluster.DefaultEtcdDataDir
}

func getEtcdPathes(ccfg *api.ClusterConfig) []string {
	return []string{
		filepath.Join(ccfg.GetCertDir(), "etcd"),
		getEtcdDataDir(ccfg.EtcdCluster.DataDir),
		"/etc/etcd",
		"/var/lib/etcd",
		"/usr/lib/systemd/system/etcd.service",
	}
}

func (t *cleanupEtcdMemberTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	stopServices(r, EtcdService)

	removePathes(r, getEtcdPathes(t.ccfg))

	postCleanup(r)

	return nil
}

func CleanupEtcdMember(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if conf.EtcdCluster.External {
		logrus.Info("external etcd, ignore remove etcds")
		return nil
	}

	// delete etcd member
	if err := etcdcluster.ExecRemoveMemberTask(conf, hostconfig); err != nil {
		return fmt.Errorf("remove etcd member %v failed", hostconfig.Name)
	}

	// cleanup remains
	taskCleanupEtcdMember := task.NewTaskInstance(
		&cleanupEtcdMemberTask{
			ccfg: conf,
		},
	)

	task.SetIgnoreErrorFlag(taskCleanupEtcdMember)
	if err := nodemanager.RunTaskOnNodes(taskCleanupEtcdMember, []string{hostconfig.Address}); err != nil {
		return fmt.Errorf("run task for cleanup etcd member failed: %v", err)
	}

	if err := nodemanager.WaitNodesFinish([]string{hostconfig.Address}, time.Minute*5); err != nil {
		return fmt.Errorf("wait for cleanup etcd member task finish failed: %v", err)
	}

	return nil
}

// cleanup all etcds
func CleanupAllEtcds(conf *api.ClusterConfig) error {
	if conf.EtcdCluster.External {
		logrus.Info("external etcd, ignore remove etcds")
		return nil
	}

	if err := etcdcluster.ExecRemoveEtcdsTask(conf); err != nil {
		return fmt.Errorf("remove etcds failed")
	}

	// cleanup remains
	taskCleanupAllEtcds := task.NewTaskInstance(
		&cleanupEtcdMemberTask{
			ccfg: conf,
		},
	)

	nodes := utils.GetAllIPs(conf.EtcdCluster.Nodes)
	task.SetIgnoreErrorFlag(taskCleanupAllEtcds)
	if err := nodemanager.RunTaskOnNodes(taskCleanupAllEtcds, nodes); err != nil {
		return fmt.Errorf("run task for cleanup all etcds failed: %v", err)
	}

	return nil
}
