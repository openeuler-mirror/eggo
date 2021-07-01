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
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

type NodeManager struct {
	// key is node.Address
	nodes map[string]*Node
	lock  sync.RWMutex
}

var manager = &NodeManager{
	nodes: make(map[string]*Node, 2),
}

func RegisterNode(hcf *api.HostConfig, r runner.Runner) error {
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

func RunTasksOnNodes(tasks []task.Task, nodes []string) error {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	for _, n := range nodes {
		if err := RunTasksOnNode(tasks, n); err != nil {
			logrus.Errorf("run tasks on node %s failed: %v", n, err)
			return fmt.Errorf("run tasks on node %s failed: %v", n, err)
		}
	}

	return nil
}

func RunTaskOnOneNode(t task.Task, nodes []string) (string, error) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	for _, id := range nodes {
		n, ok := manager.nodes[id]
		if !ok {
			logrus.Warnf("unkown node %s for task %s", id, t.Name())
			continue
		}
		if n.PushTask(t) {
			return n.host.Address, nil
		}
	}
	return "", fmt.Errorf("all nodes are busy for task %s", t.Name())
}

func checkNodeFinish(nodeID string) (bool, string, error) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	n, ok := manager.nodes[nodeID]
	if !ok {
		return true, fmt.Sprintf("unknow node: %s", nodeID), fmt.Errorf("unkown node %s", nodeID)
	}
	s := n.GetStatus()
	if s.TasksFinished() {
		if s.HasError() {
			return true, s.ShowCounts(), fmt.Errorf("%s", s.Message)
		}
		return true, s.ShowCounts(), nil
	}

	return false, s.ShowCounts(), nil
}

func WaitNodesFinishWithProgress(nodes []string, timeout time.Duration) error {
	var errmsg string
	unfinishedNodes := nodes

	finish := time.After(timeout)
outfor:
	for {
		select {
		case t := <-finish:
			return fmt.Errorf("timeout %s for WaitNodesFinishWithProgress", t.String())
		default:
			if len(unfinishedNodes) == 0 {
				break outfor
			}
			var sb strings.Builder
			var nextUnfinished []string
			for _, id := range unfinishedNodes {
				f, show, err := checkNodeFinish(id)
				if err != nil {
					errmsg = fmt.Sprintf("node: %s with error: %v\n%s", id, err, errmsg)
				}
				sb.WriteString("\nnode:")
				sb.WriteString(id + " ")
				sb.WriteString(show)
				if !f {
					nextUnfinished = append(nextUnfinished, id)
				}
			}
			logrus.Infof("Tasks progress: %s", sb.String())
			unfinishedNodes = nextUnfinished
			time.Sleep(time.Second)
		}
	}

	if errmsg != "" {
		return fmt.Errorf("%s", errmsg)
	}
	return nil
}

func WaitNodesFinish(nodes []string, timeout time.Duration) error {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	var errmsg string

	for _, id := range nodes {
		n, ok := manager.nodes[id]
		if !ok {
			return fmt.Errorf("unkown node %s", id)
		}
		err := n.WaitNodeTasksFinish(timeout)
		if err != nil {
			errmsg = fmt.Sprintf("node: %s with error: %v\n%s", id, err, errmsg)
		}
	}
	if errmsg != "" {
		return fmt.Errorf("%s", errmsg)
	}
	return nil
}

func WaitAllNodesFinished(timeout time.Duration) error {
	manager.lock.RLock()
	var nodes []string
	for id := range manager.nodes {
		nodes = append(nodes, id)
	}
	manager.lock.RUnlock()
	return WaitNodesFinish(nodes, timeout)
}
