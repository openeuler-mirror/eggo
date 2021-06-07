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
 * Description: testcase for token
 ******************************************************************************/
package commontools

import (
	"regexp"
	"testing"
)

func TestGenerateBootstrapToken(t *testing.T) {
	idPattern := `[a-z0-9]{6}`
	secretPattern := `[a-z0-9]{16}`
	tokenPattern := `\A([a-z0-9]{6})\.([a-z0-9]{16})\z`
	token, id, secret, err := ParseBootstrapTokenStr("")
	if err != nil {
		t.Fatalf("run GenerateBootstrapToken failed: %v", err)
	}
	if ok, _ := regexp.Match(idPattern, []byte(id)); !ok {
		t.Fatalf("invalid token id: %s", id)
	}
	if ok, _ := regexp.Match(secretPattern, []byte(secret)); !ok {
		t.Fatalf("invalid token secret: %s", secret)
	}
	if ok, _ := regexp.Match(tokenPattern, []byte(token)); !ok {
		t.Fatalf("invalid token: %s", token)
	}
}
