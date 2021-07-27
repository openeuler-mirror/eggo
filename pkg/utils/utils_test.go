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
 * Author: zhangxiaoyu
 * Create: 2021-05-19
 * Description: eggo utils ut
 ******************************************************************************/

package utils

import (
	"sort"
	"testing"
)

func TestIsType(t *testing.T) {
	cs := []struct {
		name   string
		val1   uint16
		val2   uint16
		expect bool
	}{
		{
			"test1",
			1,
			1,
			true,
		},
		{
			"test2",
			7,
			7,
			true,
		},
		{
			"test3",
			123,
			123,
			true,
		},
		{
			"test11",
			8,
			1,
			false,
		},
		{
			"test12",
			0,
			1,
			false,
		},
		{
			"test13",
			120,
			121,
			false,
		},
	}

	for _, c := range cs {
		if IsType(c.val1, c.val2) != c.expect {
			t.Errorf("case: %s, expect: %v, get: %v", c.name, c.expect, IsType(c.val1, c.val2))
		}
	}
}

func TestRemoveDupString(t *testing.T) {
	cs := []struct {
		name   string
		val    []string
		expect []string
	}{
		{
			"test1",
			[]string{"abc", "bcd"},
			[]string{"abc", "bcd"},
		},
		{
			"test2",
			[]string{"abc", "bcd", "abc"},
			[]string{"abc", "bcd"},
		},
		{
			"test3",
			[]string{"xxx", "bcd"},
			[]string{"bcd", "xxx"},
		},
		{
			"test4",
			[]string{"xxx", "xxx", "xxx"},
			[]string{"xxx"},
		},
	}

	for _, c := range cs {
		flag := true
		got := RemoveDupString(c.val)
		if len(got) == len(c.expect) {
			sort.Strings(got)
			sort.Strings(c.expect)
			for i := 0; i < len(got); i++ {
				if got[i] != c.expect[i] {
					flag = false
				}
			}
		} else {
			flag = false
		}
		if !flag {
			t.Errorf("case: %s, expect: %v, get: %v", c.name, c.expect, got)
		}
	}
}
