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
 * Create: 2021-07-27
 * Description: eggo etcd join binary implement
 ******************************************************************************/

package etcdcluster

import (
	"fmt"
	"time"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/task"
)

func AddMember(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	initialCluster, err := ExecAddMemberTask(conf, hostconfig)
	if err != nil {
		return err
	}

	tasks := []task.Task{
		task.NewTaskInstance(
			&EtcdDeployEtcdsTask{
				ccfg:           conf,
				initialCluster: initialCluster,
			},
		),
		task.NewTaskInstance(
			&EtcdPostDeployEtcdsTask{
				ccfg: conf,
			},
		),
	}

	if err := nodemanager.RunTasksOnNode(tasks, hostconfig.Address); err != nil {
		return fmt.Errorf("run task on nodes failed: %v", err)
	}

	if err := nodemanager.WaitNodesFinish([]string{hostconfig.Address}, 5*time.Minute); err != nil {
		return fmt.Errorf("wait for post deploy etcds task finish failed: %v", err)
	}

	return nil
}
