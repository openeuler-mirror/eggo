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
 * Create: 2021-07-27
 * Description: util for run shell
 ******************************************************************************/
package commontools

import (
	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/runner"
)

type RunShellTask struct {
	ShellName string
	Shell     string
}

func (ct *RunShellTask) Name() string {
	return "RunShellTask"
}

func (ct *RunShellTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	out, err := r.RunShell(ct.Shell, ct.ShellName)
	if err != nil {
		return err
	}
	logrus.Debugf("run shell: %s, get out:\n%s", ct.ShellName, out)
	return nil
}
