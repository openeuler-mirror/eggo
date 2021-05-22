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
 * Description: eggo utils implement
 ******************************************************************************/

package utils

import (
	"strings"
)

func IsX86Arch(arch string) bool {
	if strings.HasPrefix(arch, "x86") || strings.HasPrefix(arch, "X86") {
		return true
	}

	return false
}

func IsArmArch(arch string) bool {
	if strings.HasPrefix(arch, "arm") || strings.HasPrefix(arch, "Arm") || strings.HasPrefix(arch, "ARM") {
		return true
	}

	return false
}