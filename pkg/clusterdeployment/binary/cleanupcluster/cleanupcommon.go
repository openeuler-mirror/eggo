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
 * Create: 2021-05-24
 * Description: eggo cleanup cluster binary implement
 ******************************************************************************/

package cleanupcluster

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/runner"
)

func removePathes(r runner.Runner, pathes []string) {
	for _, path := range pathes {
		if output, err := r.RunCommand(utils.AddSudo("rm -rf " + path)); err != nil {
			logrus.Errorf("remove path %v failed: %v\noutput: %v",
				path, err, output)
		}
	}
}

func PostCleanup(r runner.Runner) {
	// daemon-reload
	if output, err := r.RunCommand(utils.AddSudo("systemctl daemon-reload")); err != nil {
		logrus.Errorf("daemon-reload failed: %v\noutput: %v", err, output)
	}
}

func stopServices(r runner.Runner, services []string) error {
	join := strings.Join(services, " ")
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"systemctl stop %s\"", join)); err != nil {
		logrus.Errorf("stop services failed: %v", err)
		return err
	}
	return nil
}
