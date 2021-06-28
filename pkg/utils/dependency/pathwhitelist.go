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
 * Create: 2021-06-26
 * Description: eggo removed path white list
 ******************************************************************************/

package dependency

import (
	"path/filepath"
	"strings"
)

var (
	// subdir path in white list
	SubWhiteList = []string{
		"/usr/bin", "/usr/local/bin",
		"/usr/lib/systemd/system", "/etc/systemd/system",
		"/tmp",
	}
	// dir and subdir path in white list
	WhiteList = []string{
		"/opt/cni/bin", "/usr/libexec/cni", "/etc/cni/net.d",
		"/etc/kubernetes",
	}
)

func CheckPath(path string) bool {
	cleanPath := filepath.Clean(path)

	for _, w := range WhiteList {
		if strings.HasPrefix(cleanPath, w) {
			return true
		}
	}

	for _, s := range SubWhiteList {
		if strings.HasPrefix(cleanPath, s) && cleanPath != s {
			return true
		}
	}

	return false
}
