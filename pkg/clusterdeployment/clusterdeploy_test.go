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
 * Description: cluster deploy test
 ******************************************************************************/

package clusterdeployment

import (
	"testing"
)

func TestRegisterControlPlaneDriver(t *testing.T) {
	err := RegisterClusterDeploymentDriver("test", nil)
	if err == nil {
		t.Fatal("expect err is not nil")
	}
}

func MyCreator(*ClusterConfig) (ClusterDeploymentAPI, error) {
	return nil, nil
}
func TestGetControlPlaneDriver(t *testing.T) {
	err := RegisterClusterDeploymentDriver("test", MyCreator)
	if err != nil {
		t.Fatal("expect err is nil")
	}

	if _, err = GetClusterDeploymentDriver("test"); err != nil {
		t.Fatal("expect err is not nil")
	}
}
