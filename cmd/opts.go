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
 * Create: 2021-05-28
 * Description: eggo command opts implement
 ******************************************************************************/

package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type eggoOptions struct {
	config  string
	version bool
}

var opts eggoOptions

func init() {
	if _, err := os.Stat(getEggoDir()); err == nil {
		return
	}

	if err := os.Mkdir(getEggoDir(), 0700); err != nil {
		logrus.Errorf("mkdir eggo directory %v failed", getEggoDir())
	}
}

func setupEggoCmdOpts(eggoCmd *cobra.Command) {
	flags := eggoCmd.Flags()
	flags.BoolVarP(&opts.version, "version", "v", false, "Print version information and quit")
}

func setupDeployCmdOpts(deployCmd *cobra.Command) {
	flags := deployCmd.Flags()
	flags.StringVarP(&opts.config, "file", "f", getDefaultDeployConfig(), "location of cluster deploy config file, default $HOME/.eggo/deploy.yaml")
}

func setupCleanupCmdOpts(cleanupCmd *cobra.Command) {
	flags := cleanupCmd.Flags()
	flags.StringVarP(&opts.config, "file", "f", getDefaultDeployConfig(), "location of cluster deploy config file for cleanup, default $HOME/.eggo/deploy.yaml")
}

func setupTemplateCmdOpts(templateCmd *cobra.Command) {
	flags := templateCmd.Flags()
	flags.StringVarP(&opts.config, "file", "f", "", "location of eggo's template config file, default $(current)/template.yaml")
}
