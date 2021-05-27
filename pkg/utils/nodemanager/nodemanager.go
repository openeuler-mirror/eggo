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
 * Description: provide nodemanager implement
 ******************************************************************************/

package nodemanager

import (
	"fmt"
	"sync"
	"time"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type NodeManager struct {
	// key is node.Address
	nodes map[string]*Node
	lock  sync.RWMutex
}

var manager = &NodeManager{
	nodes: make(map[string]*Node, 2),
}

func RegisterNode(hcf *clusterdeployment.HostConfig, r runner.Runner) error {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	if hcf == nil {
		return fmt.Errorf("empty hostconfig arguments")
	}
	if _, ok := manager.nodes[hcf.Address]; ok {
		logrus.Debugf("node %s is already registered", hcf.Address)
		return nil
	}
	n, err := NewNode(hcf, r)
	if err != nil {
		return err
	}
	manager.nodes[n.host.Address] = n
	return nil
}

func UnRegisterNode(nodeID string) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	n, ok := manager.nodes[nodeID]
	if !ok {
		logrus.Debugf("node %s do not registered", nodeID)
		return
	}
	delete(manager.nodes, nodeID)
	n.Finish()
}

func UnRegisterAllNodes() {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	for _, n := range manager.nodes {
		n.Finish()
	}
	manager.nodes = make(map[string]*Node, 2)
}

func doRetryPushTask(t task.Task, retryNodes []*Node) error {
	for _, n := range retryNodes {
		pushed := false
		for i := 0; i < 5 && !pushed; i++ {
			time.Sleep(time.Second)
			pushed = n.PushTask(t)
		}
		if !pushed {
			// retry failed, just return error
			return fmt.Errorf("node: %s work with too much tasks, failed to run new task", n.host.Address)
		}
	}
	return nil
}

func checkFinished(t task.Task, nodes []string) (bool, error) {
	finished := true
	var err error
	for _, id := range nodes {
		label := t.GetLabel(id)
		if label == "" {
			return false, nil
		}
		if task.IsFailed(label) {
			err = errors.Wrapf(err, "task: %s failed: %v", t.Name(), label)
			break
		}
	}

	return finished, err
}

func WaitTaskOnNodesFinished(t task.Task, nodes []string, timeout time.Duration) error {
	for {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timeout for wait task: %s finish", t.Name())
		default:
			f, err := checkFinished(t, nodes)
			if err != nil {
				return err
			}
			if f {
				return nil
			}
		}
	}
}

func checkAllFinished(t task.Task) (bool, error) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	finished := true
	var err error
	for id := range manager.nodes {
		label := t.GetLabel(id)
		if label == "" {
			return false, nil
		}
		if task.IsFailed(label) {
			err = errors.Wrapf(err, "task: %s failed: %v", t.Name(), label)
			continue
		}
	}

	return finished, err
}

func WaitTaskOnAllFinished(t task.Task, timeout time.Duration) error {
	for {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timeout for wait task: %s finish", t.Name())
		default:
			f, err := checkAllFinished(t)
			if err != nil {
				return err
			}
			if f {
				return nil
			}
		}
	}
}

func RunTaskOnNodes(t task.Task, nodes []string) error {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	var retryNodes []*Node
	for _, id := range nodes {
		if n, ok := manager.nodes[id]; ok {
			if n.PushTask(t) {
				continue
			}
			logrus.Warnf("node: %s work with too much tasks, will retry it", id)
			retryNodes = append(retryNodes, n)
		} else {
			return fmt.Errorf("unkown node %s", id)
		}
	}

	return doRetryPushTask(t, retryNodes)
}

func RunTaskOnAll(t task.Task) error {
	var retryNodes []*Node
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	for id, n := range manager.nodes {
		if n.PushTask(t) {
			continue
		}
		logrus.Warnf("node: %s work with too much tasks, will retry it", id)
		retryNodes = append(retryNodes, n)
	}

	return doRetryPushTask(t, retryNodes)
}

func RunTasksOnNode(tasks []task.Task, node string) error {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	for _, t := range tasks {
		if n, ok := manager.nodes[node]; ok {
			i := 0
			for ; i < 5; i++ {
				if n.PushTask(t) {
					break
				}
				time.Sleep(time.Second * 6)
			}
			if i == 5 {
				logrus.Errorf("node: %s work with too much tasks, will retry it", node)
				return fmt.Errorf("node: %s work with too much tasks, will retry it", node)
			}
		} else {
			logrus.Errorf("unkown node %s", node)
			return fmt.Errorf("unkown node %s", node)
		}
	}

	return nil
}

func checkTasksFinished(tasks []task.Task, node string) (bool, error) {
	finished := true
	var err error
	for _, t := range tasks {
		label := t.GetLabel(node)
		if label == "" {
			return false, nil
		}
		if task.IsFailed(label) {
			err = errors.Wrapf(err, "task: %s failed: %v", t.Name(), label)
			continue
		}
	}

	return finished, err
}

func WaitTasksOnNodeFinished(tasks []task.Task, node string, timeout time.Duration) error {
	for {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timeout for wait tasks finish at node: %s", node)
		default:
			f, err := checkTasksFinished(tasks, node)
			if err != nil {
				return err
			}
			if f {
				return nil
			}
		}
	}
}
