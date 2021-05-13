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
 * Create: 2021-05-13
 * Description: provide node implements
 ******************************************************************************/

package nodemanager

import (
	"fmt"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

type Node struct {
	host *clusterdeployment.HostConfig
	r    runner.Runner
	stop chan bool
	// work on up to 10 tasks at a time
	queue chan task.Task
}

func (n *Node) PushTask(task task.Task) bool {
	select {
	case n.queue <- task:
		return true
	default:
		logrus.Error("node task queue is full")
		return false
	}
}

func (n *Node) Finish() {
	n.stop <- true
	n.r.Close()
	logrus.Infof("node: %s is finished", n.host.Address)
}

func NewNode(hcf *clusterdeployment.HostConfig, r runner.Runner) (*Node, error) {
	// TODO: maybe we need deap copy hostconfig
	n := &Node{
		host:  hcf,
		r:     r,
		stop:  make(chan bool),
		queue: make(chan task.Task, 10),
	}
	go func(n *Node) {
		for {
			select {
			case <-n.stop:
				return
			case t := <-n.queue:
				err := t.Run(n.r)
				if err != nil {
					label := fmt.Sprintf("%s: run task: %s on node: %s fail", task.FAILED, t.Name(), n.host.Address)
					t.AddLabels(n.host.Address, label)
				} else {
					t.AddLabels(n.host.Address, task.SUCCESS)
					logrus.Infof("run task: %s success on %s\n", t.Name(), n.host.Address)
				}
			}
		}
	}(n)
	return n, nil
}
