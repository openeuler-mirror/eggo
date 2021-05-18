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

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
)

const (
	SUCCESS = "success"
	FAILED  = "failed"
)

type Task interface {
	Name() string
	Run(runner.Runner, *clusterdeployment.HostConfig) error
	AddLabel(key, lable string)
	GetLable(key string) string
}

type Labels struct {
	data map[string]string
	l    sync.RWMutex
}

func (l *Labels) AddLabel(key, lable string) {
	l.l.Lock()
	defer l.l.Unlock()
	l.data[key] = lable
}

func (l *Labels) GetLabel(key string) string {
	l.l.RLock()
	defer l.l.RUnlock()

	val, ok := l.data[key]
	if ok {
		return val
	}

	return ""
}

func NewLabel() *Labels {
	return &Labels{data: make(map[string]string)}
}

func IsSuccess(lable string) bool {
	return strings.HasPrefix(lable, SUCCESS)
}

func IsFailed(lable string) bool {
	return strings.HasPrefix(lable, FAILED)
}
