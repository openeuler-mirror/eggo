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
 * Create: 2021-05-24
 * Description: eggo cleanup cluster binary implement
 ******************************************************************************/

package cleanupcluster

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

var (
	LoadBalanceService = []string{"nginx"}
)

type cleanupLoadBalanceTask struct {
	ccfg *api.ClusterConfig
}

func (t *cleanupLoadBalanceTask) Name() string {
	return "cleanupLoadBalanceTask"
}

func getLoadBalancePathes() []string {
	return []string{"/etc/nginx", "/usr/lib/systemd/system/nginx.service"}
}

func (t *cleanupLoadBalanceTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	// stop service before remove dependences
	if err := stopServices(r, LoadBalanceService); err != nil {
		logrus.Errorf("stop loadbalance service failed: %v", err)
	}

	removePathes(r, getLoadBalancePathes())

	PostCleanup(r)

	return nil
}

func CleanupLoadBalance(conf *api.ClusterConfig, lb *api.HostConfig) error {
	taskCleanupLoadBalance := task.NewTaskIgnoreErrInstance(
		&cleanupLoadBalanceTask{
			ccfg: conf,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskCleanupLoadBalance, []string{lb.Address}); err != nil {
		return fmt.Errorf("run task for cleanup loadbalance failed: %v", err)
	}

	return nil
}
