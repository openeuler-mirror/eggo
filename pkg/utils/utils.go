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
	"os"
	"os/user"
	"path/filepath"

	"isula.org/eggo/pkg/api"
)

func GetSysHome() string {
	if user, err := user.Current(); err == nil {
		return user.HomeDir
	}
	return "/root"
}

func GetEggoDir() string {
	return filepath.Join(GetSysHome(), ".eggo")
}

func IsType(curType uint16, expectedType uint16) bool {
	return (curType & expectedType) == expectedType
}

func ClearType(curType uint16, clearType uint16) uint16 {
	return (curType & ^clearType)
}

func AddSudo(cmd string) string {
	return "sudo -E /bin/sh -c \"" + cmd + "\""
}

func GetMasterIPList(c *api.ClusterConfig) []string {
	var masters []string
	for _, n := range c.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
			continue
		}
	}

	return masters
}

func GetAllIPs(nodes []*api.HostConfig) []string {
	var ips []string

	for _, node := range nodes {
		ips = append(ips, node.Address)
	}

	return ips
}

func RemoveDupString(str []string) []string {
	strMap := map[string]bool{}
	result := []string{}

	for _, s := range str {
		if _, ok := strMap[s]; !ok {
			strMap[s] = true
			result = append(result, s)
		}
	}

	return result
}

func CheckPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
