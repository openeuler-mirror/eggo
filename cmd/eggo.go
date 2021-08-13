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
 * Description: eggo command implement
 ******************************************************************************/

package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func showVersion() {
	fmt.Println("eggo version 0.0.1")
}

func initLog() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return fmt.Sprintf("%s()", path.Base(f.Function)), fmt.Sprintf("%s:%d", path.Base(f.File), f.Line)
		},
	})
}

func preCheck() {
	proxies := []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"}
	var sb strings.Builder
	sb.WriteString("Warning:\n")
	flag := false
	for _, p := range proxies {
		if v := os.Getenv(p); v != "" {
			flag = true
			sb.WriteString(fmt.Sprintf("\tproxy is setted: %s=%s\n", p, v))
		}
	}
	if flag {
		sb.WriteString("Maybe cause to failure!!!\n")
		fmt.Println(sb.String())
	}
}

func NewEggoCmd() *cobra.Command {
	eggoCmd := &cobra.Command{
		Short:         "eggo is a tool built to provide standard multi-ways for creating Kubernetes clusters",
		Use:           "eggo",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			preCheck()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.version {
				showVersion()
				return nil
			}
			cmd.Help()
			return nil
		},
	}
	eggoCmd.PersistentFlags().BoolVarP(&opts.debug, "debug", "d", false, "Run debug mode")

	setupEggoCmdOpts(eggoCmd)

	eggoCmd.AddCommand(NewDeployCmd())
	eggoCmd.AddCommand(NewCleanupCmd())
	eggoCmd.AddCommand(NewTemplateCmd())
	eggoCmd.AddCommand(NewJoinCmd())
	eggoCmd.AddCommand(NewDeleteCmd())

	return eggoCmd
}
