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

package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
)

type eggoOptions struct {
	name                 string
	templateConfig       string
	masters              []string
	nodes                []string
	etcds                []string
	loadbalance          string
	username             string
	password             string
	deployConfig         string
	deployEnableRollback bool
	cleanupConfig        string
	cleanupClusterID     string
	debug                bool
	version              bool
	joinType             string
	joinClusterID        string
	joinYaml             string
	joinHost             HostConfig
	delClusterID         string
	clusterPrehook       string
	clusterPosthook      string
	prehook              string
	posthook             string
}

var opts eggoOptions

func init() {
	if _, err := os.Stat(utils.GetEggoDir()); err == nil {
		return
	}

	if err := os.Mkdir(utils.GetEggoDir(), constants.EggoDirMode); err != nil {
		logrus.Errorf("mkdir eggo directory %v failed", utils.GetEggoDir())
	}
}

func setupEggoCmdOpts(eggoCmd *cobra.Command) {
	flags := eggoCmd.Flags()
	flags.BoolVarP(&opts.version, "version", "v", false, "Print version information and quit")
}

func setupDeployCmdOpts(deployCmd *cobra.Command) {
	flags := deployCmd.Flags()
	flags.StringVarP(&opts.deployConfig, "file", "f", defaultDeployConfigPath(), "location of cluster deploy config file, default $HOME/.eggo/deploy.yaml")
	flags.BoolVarP(&opts.deployEnableRollback, "rollback", "", true, "rollback failed node to cleanup")
	flags.StringVarP(&opts.clusterPrehook, "cluster-prehook", "", "", "cluser prehooks when deploy cluser")
	flags.StringVarP(&opts.clusterPosthook, "cluster-posthook", "", "", "cluster posthook when deploy cluster")
}

func setupCleanupCmdOpts(cleanupCmd *cobra.Command) {
	flags := cleanupCmd.Flags()
	flags.StringVarP(&opts.cleanupConfig, "file", "f", "", "location of cluster deploy config file")
	flags.StringVarP(&opts.cleanupClusterID, "id", "", "", "cluster id")
	flags.StringVarP(&opts.clusterPrehook, "cluster-prehook", "", "", "cluser prehooks when clenaup cluser")
	flags.StringVarP(&opts.clusterPosthook, "cluster-posthook", "", "", "cluster posthook when cleaup cluster")
}

func setupJoinCmdOpts(joinCmd *cobra.Command) {
	flags := joinCmd.Flags()
	flags.StringVarP(&opts.joinType, "type", "t", "", "join type, can be \"master,worker\", deault worker")
	flags.StringVarP(&opts.joinHost.Arch, "arch", "a", "", "host's architecture")
	flags.StringVarP(&opts.joinHost.Name, "name", "n", "", "host's name")
	flags.IntVarP(&opts.joinHost.Port, "port", "p", 0, "host's ssh port")
	flags.StringVarP(&opts.joinClusterID, "id", "", "", "cluster id")
	flags.StringVarP(&opts.joinYaml, "file", "f", "", "yaml file contain nodes information")
	flags.StringVarP(&opts.prehook, "prehook", "", "", "prehook when join cluster")
	flags.StringVarP(&opts.posthook, "posthook", "", "", "posthook when join cluster")
}

func setupDeleteCmdOpts(deleteCmd *cobra.Command) {
	flags := deleteCmd.Flags()
	flags.StringVarP(&opts.delClusterID, "id", "", "", "cluster id")
	flags.StringVarP(&opts.prehook, "prehook", "", "", "prehook when delete cluster")
	flags.StringVarP(&opts.posthook, "posthook", "", "", "posthook when delete cluster")
}

func setupTemplateCmdOpts(templateCmd *cobra.Command) {
	flags := templateCmd.Flags()
	flags.StringVarP(&opts.name, "name", "n", "k8s-cluster", "set cluster name")
	flags.StringVarP(&opts.username, "user", "u", "root", "user to login all node")
	flags.StringVarP(&opts.password, "password", "p", "123456", "password to login all node")
	flags.StringArrayVarP(&opts.masters, "masters", "", []string{"192.168.0.2"}, "set master ips")
	flags.StringArrayVarP(&opts.nodes, "workers", "", []string{"192.168.0.3", "192.168.0.4"}, "set worker ips")
	flags.StringArrayVarP(&opts.etcds, "etcds", "", nil, "set etcd node ips")
	flags.StringVarP(&opts.loadbalance, "loadbalance", "l", "192.168.0.1", "set loadbalance node")
	flags.StringVarP(&opts.templateConfig, "file", "f", "template.yaml", "location of eggo's template config file, default $(current)/template.yaml")
}
