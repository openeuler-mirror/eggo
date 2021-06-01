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
	"sync"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

const (
	BeginLabel   = "[Starting]"
	FinishPrefix = "[Finish]"
)

type Node struct {
	host *api.HostConfig
	r    runner.Runner
	stop chan bool
	// work on up to 10 tasks at a time
	queue      chan task.Task
	labelsLock sync.RWMutex
	lables     map[string]string
}

func (n *Node) addLabel(key, label string) {
	n.labelsLock.Lock()
	n.lables[key] = label
	n.labelsLock.Unlock()
}

func (n *Node) CheckProgress() string {
	n.labelsLock.RLock()
	taskCnt := len(n.lables)
	finishCnt := 0
	for _, v := range n.lables {
		if v != BeginLabel {
			finishCnt++
		}
	}
	n.labelsLock.RUnlock()

	return fmt.Sprintf("%d/%d", finishCnt, taskCnt)
}

func (n *Node) GetTaskStatus(task string) (string, error) {
	n.labelsLock.RLock()
	v, ok := n.lables[task]
	n.labelsLock.RUnlock()
	if ok {
		return v, nil
	}
	return "", fmt.Errorf("cannot found task: %s", task)
}

func (n *Node) PushTask(task task.Task) bool {
	select {
	case n.queue <- task:
		n.addLabel(task.Name(), BeginLabel)
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

func NewNode(hcf *api.HostConfig, r runner.Runner) (*Node, error) {
	// TODO: maybe we need deap copy hostconfig
	n := &Node{
		host:   hcf,
		r:      r,
		stop:   make(chan bool),
		queue:  make(chan task.Task, 10),
		lables: make(map[string]string, 10),
	}
	go func(n *Node) {
		for {
			select {
			case <-n.stop:
				return
			case t := <-n.queue:
				// set task status on node before run task
				err := t.Run(n.r, n.host)
				if err != nil {
					label := fmt.Sprintf("%s: run task: %s on node: %s fail: %v", task.FAILED, t.Name(), n.host.Address, err)
					t.AddLabel(n.host.Address, label)
					// set task status on node after task
					n.addLabel(t.Name(), fmt.Sprintf("%s with err: %v", FinishPrefix, err))
					logrus.Errorf("%s", label)
				} else {
					t.AddLabel(n.host.Address, task.SUCCESS)
					// set task status on node after task
					n.addLabel(t.Name(), fmt.Sprintf("%s success", FinishPrefix))
					logrus.Infof("run task: %s success on %s\n", t.Name(), n.host.Address)
				}
			}
		}
	}(n)
	return n, nil
}
