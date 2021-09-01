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
 * Author: wangfengtu
 * Create: 2021-08-18
 * Description: etcd reconfig testcase
 ******************************************************************************/

package etcdcluster

import (
	"testing"
)

func TestExecRemoveEtcdsTask(t *testing.T) {
	registerFakeRunner(t)
	if err := ExecRemoveEtcdsTask(conf); err != nil {
		t.Fatalf("test exec remove etcds task failed")
	}
}

func TestExecRemoveMemberTask(t *testing.T) {
	registerFakeRunner(t)
	if err := ExecRemoveMemberTask(conf, nodes[1]); err != nil {
		t.Fatalf("test exec remove member task failed")
	}
}
