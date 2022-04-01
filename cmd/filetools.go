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
 * Create: 2021-08-18
 * Description: eggo utils for handle file
 ******************************************************************************/
package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"isula.org/eggo/pkg/constants"
)

func checkProcessRunning(pid int) bool {
	_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return err == nil
}

func checkProcessInFile(path string) error {
	bPid, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	pidStr := string(bPid)
	pidStr = strings.TrimSpace(pidStr)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return err
	}
	if !checkProcessRunning(pid) {
		return nil
	}
	return fmt.Errorf("process: %v is running", pid)
}

type ProcessPlaceHolder struct {
	file string
}

func NewProcessPlaceHolder(path string) (*ProcessPlaceHolder, error) {
	if err := checkProcessInFile(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), constants.EggoHomeDirMode); err != nil {
		return nil, err
	}
	pid := strconv.Itoa(os.Getpid())
	if err := ioutil.WriteFile(path, []byte(pid), constants.ProcessFileMode); err != nil {
		return nil, err
	}
	return &ProcessPlaceHolder{path}, nil
}

func (p ProcessPlaceHolder) Remove() error {
	return os.Remove(p.file)
}
