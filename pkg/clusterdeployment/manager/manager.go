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
 * Create: 2021-06-07
 * Description: manage cluster deploy drivers
 ******************************************************************************/
package manager

import (
	"fmt"
	"sync"

	"isula.org/eggo/pkg/api"
)

type ClusterDeploymentCreator func(*api.ClusterConfig) (api.ClusterDeploymentAPI, error)

type clusterDeploymentFactory struct {
	registry map[string]ClusterDeploymentCreator
	m        sync.Mutex
}

func (df *clusterDeploymentFactory) register(name string, c ClusterDeploymentCreator) error {
	df.m.Lock()
	defer df.m.Unlock()
	if _, ok := df.registry[name]; ok {
		return fmt.Errorf("driver %s is already registered", name)
	}
	df.registry[name] = c
	return nil
}

func (df *clusterDeploymentFactory) get(name string) (ClusterDeploymentCreator, error) {
	df.m.Lock()
	defer df.m.Unlock()
	c, ok := df.registry[name]
	if ok {
		return c, nil
	}
	return nil, fmt.Errorf("driver %s cannot be found", name)
}

// global factory instance
var factory = &clusterDeploymentFactory{registry: make(map[string]ClusterDeploymentCreator)}

func RegisterClusterDeploymentDriver(name string, c ClusterDeploymentCreator) error {
	if c == nil {
		return fmt.Errorf("creator is nil")
	}
	return factory.register(name, c)
}

func GetClusterDeploymentDriver(name string) (ClusterDeploymentCreator, error) {
	return factory.get(name)
}
