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
 * Create: 2021-06-22
 * Description: coredns unified interfaces
 ******************************************************************************/
package coredns

import (
	"fmt"

	"isula.org/eggo/pkg/api"
)

const (
	CorednsTypeOfPod    = "pod"
	CorednsTypeOfBinary = "binary"
)

var cbs map[string]CorednsOps

func init() {
	cbs = make(map[string]CorednsOps)
	cbs[CorednsTypeOfPod] = &PodCoredns{}
	cbs[CorednsTypeOfBinary] = &BinaryCoredns{}
}

func getTypeOfCoredns(configType string) string {
	if configType != "" {
		return configType
	}

	return CorednsTypeOfBinary
}

func CorednsSetup(cluster *api.ClusterConfig) error {
	useType := getTypeOfCoredns(cluster.ServiceCluster.DNS.CorednsType)
	if cb, ok := cbs[useType]; ok {
		return cb.Setup(cluster)
	}
	return fmt.Errorf("unsupport coredns type %s", useType)
}

func CorednsCleanup(cluster *api.ClusterConfig) error {
	useType := getTypeOfCoredns(cluster.ServiceCluster.DNS.CorednsType)
	if cb, ok := cbs[useType]; ok {
		return cb.Cleanup(cluster)
	}
	return fmt.Errorf("unsupport coredns type %s", useType)
}

type CorednsOps interface {
	Setup(cluster *api.ClusterConfig) error
	Cleanup(cluster *api.ClusterConfig) error
	JoinNode(node string, cluster *api.ClusterConfig) error
	CleanNode(node string, cluster *api.ClusterConfig) error
}
