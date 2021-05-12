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
 * Create: 2021-05-11
 * Description: eggo controlplane binary implement
 ******************************************************************************/

package controlplane

import (
	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
)

func check() error {
	return nil
}

func installDependences() error {
	return nil
}

func generateCerts() error {
	return nil
}

func generateKubeconfigs() error {
	return nil
}

func runKubernetesServices() error {
	return nil
}

func Init(conf *clusterdeployment.ClusterConfig) error {
	check()
	installDependences()
	generateCerts()
	generateKubeconfigs()
	runKubernetesServices()
	return nil
}
