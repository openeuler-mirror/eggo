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
 * Description: task interface and common tools
 ******************************************************************************/

package task

import (
	"strings"
	"sync"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/runner"
)

const (
	SUCCESS = "success"
	FAILED  = "failed"
)

type TaskRun interface {
	Name() string
	Run(runner.Runner, *api.HostConfig) error
}

type Task interface {
	AddLabel(key, label string)
	GetLabel(key string) string
	TaskRun
}

type TaskInstance struct {
	data map[string]string
	l    sync.RWMutex
	TaskRun
}

func NewTaskInstance(t TaskRun) *TaskInstance {
	return &TaskInstance{
		data:    make(map[string]string),
		TaskRun: t,
	}
}

func (t *TaskInstance) AddLabel(key, label string) {
	t.l.Lock()
	defer t.l.Unlock()
	t.data[key] = label
}

func (t *TaskInstance) GetLabel(key string) string {
	t.l.RLock()
	defer t.l.RUnlock()

	val, ok := t.data[key]
	if ok {
		return val
	}

	return ""
}

func IsSuccess(label string) bool {
	return label == SUCCESS
}

func IsFailed(label string) bool {
	return strings.HasPrefix(label, FAILED)
}
