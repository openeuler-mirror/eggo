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
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

const (
	WorkingStatus = iota
	FinishStatus
	IgnoreStatus
	ErrorStatus
)

type NodeStatus struct {
	Status         int
	Message        string
	TaskTotalCnt   int
	TaskSuccessCnt int
	TaskIgnoreCnt  int
	TaskFailCnt    int
}

func (ns NodeStatus) HasError() bool {
	return ns.TaskFailCnt > 0
}

func (s NodeStatus) TasksFinished() bool {
	return s.TaskTotalCnt == s.TaskSuccessCnt+s.TaskFailCnt+s.TaskIgnoreCnt
}

func (ns NodeStatus) ShowCounts() string {
	return fmt.Sprintf("{ total: %d, success: %d, fail: %d, ignore: %d }", ns.TaskTotalCnt, ns.TaskSuccessCnt, ns.TaskFailCnt, ns.TaskIgnoreCnt)
}

type Node struct {
	host *api.HostConfig
	r    runner.Runner
	stop chan bool
	// work on up to 10 tasks at a time
	queue  chan task.Task
	lock   sync.RWMutex
	status NodeStatus
}

func (n *Node) GetStatus() NodeStatus {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.status
}

func (n *Node) WaitNodeTasksFinish(timeout time.Duration) error {
	finish := time.After(timeout)
	for {
		select {
		case t := <-finish:
			return fmt.Errorf("timeout %s for wait node: %s", t.String(), n.host.Name)
		default:
			n.lock.RLock()
			s := n.status
			msg := s.Message
			n.lock.RUnlock()
			if !s.TasksFinished() {
				time.Sleep(time.Millisecond * 200)
				continue
			}
			if s.HasError() {
				return fmt.Errorf("%s", msg)
			}
			return nil
		}
	}
}

func (n *Node) updateTotalCnt() {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.status.TaskTotalCnt += 1
}

func (n *Node) updateNodeStatus(message string, status int) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.status.Message = message
	n.status.Status = status
	if status == FinishStatus {
		n.status.TaskSuccessCnt += 1
	}
	if status == ErrorStatus {
		n.status.TaskFailCnt += 1
	}
	if status == IgnoreStatus {
		n.status.TaskIgnoreCnt += 1
	}
}

func (n *Node) PushTask(t task.Task) bool {
	// only run ignore error tasks to cleanup node
	if n.status.HasError() && task.IsIgnoreError(t) {
		logrus.Debugf("node finished with error: %v", n.status.Message)
		return false
	}

	select {
	case n.queue <- t:
		n.updateTotalCnt()
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
		host:  hcf,
		r:     r,
		stop:  make(chan bool),
		queue: make(chan task.Task, 16),
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
					if task.IsIgnoreError(t) {
						logrus.Warnf("ignore: %s", label)
						n.updateNodeStatus("", IgnoreStatus)
					} else {
						logrus.Errorf("%s", label)
						// set task status on node after task
						n.updateNodeStatus(label, ErrorStatus)
					}
				} else {
					t.AddLabel(n.host.Address, task.SUCCESS)
					// set task status on node after task
					n.updateNodeStatus("", FinishStatus)
					logrus.Infof("run task: %s success on %s\n", t.Name(), n.host.Address)
				}
			}
		}
	}(n)
	return n, nil
}
