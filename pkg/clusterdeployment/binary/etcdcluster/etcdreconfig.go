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
 * Create: 2021-05-19
 * Description: eggo etcdcluster binary implement
 ******************************************************************************/

package etcdcluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

type EtcdEtcdReconfigTask struct {
	ccfg         *api.ClusterConfig
	reconfigType string
}

func (t *EtcdEtcdReconfigTask) Name() string {
	return "EtcdEtcdReconfigTask"
}

func getEtcdIDByName(etcds []*etcdMember, name string) string {
	for _, etcd := range etcds {
		if etcd.name == name {
			return etcd.id
		}
	}
	return ""
}

func (t *EtcdEtcdReconfigTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	etcds := getEtcdMembers(t.ccfg.GetCertDir(), r)
	if etcds == nil {
		return fmt.Errorf("get etcds failed")
	}

	id := getEtcdIDByName(etcds, hostConfig.Name)
	if id == "" {
		return fmt.Errorf("get etcd id to remove failed")
	}

	if t.reconfigType == "delete" {
		if err := removeEtcd(r, t.ccfg.GetCertDir(), id); err != nil {
			return err
		}
	}
	if t.reconfigType == "add" {
		// TODO
	}

	return nil
}

func etcdReconfig(conf *api.ClusterConfig, hostconfig *api.HostConfig, reconfigType string) error {
	if len(conf.EtcdCluster.Nodes) == 0 {
		return fmt.Errorf("invalid null etcd node")
	}

	// only one etcd member found, no need to notify etcd cluster to delete
	if len(conf.EtcdCluster.Nodes) == 1 && reconfigType == "delete" {
		return nil
	}

	taskEtcdReconfig := task.NewTaskInstance(
		&EtcdEtcdReconfigTask{
			ccfg:         conf,
			reconfigType: reconfigType,
		},
	)

	nodes := []string{hostconfig.Address}
	if err := nodemanager.RunTaskOnNodes(taskEtcdReconfig, nodes); err != nil {
		return fmt.Errorf("run task on nodes failed: %v", err)
	}

	return nil
}

func ExecRemoveMemberTask(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if !conf.EtcdCluster.External {
		return etcdReconfig(conf, hostconfig, "remove")
	} else {
		logrus.Info("external etcd, ignore remove etcds")
		return nil
	}
}

func ExecAddMemberTask(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if !conf.EtcdCluster.External {
		return etcdReconfig(conf, hostconfig, "add")
	} else {
		logrus.Info("external etcd, ignore add etcds")
		return nil
	}
}

type removeEtcdsTask struct {
	ccfg *api.ClusterConfig
}

func (t *removeEtcdsTask) Name() string {
	return "removeEtcdsTask"
}

type etcdMember struct {
	id     string
	name   string
	leader bool
}

// output:
// 868b499159f00586, started, workder0, https://192.168.0.1:2380, https://192.168.0.1:2379, false
// 6787454327e00766, started, workder1, https://192.168.0.2:2380, https://192.168.0.2:2379, true
func parseEtcdMemberList(output string) []*etcdMember {
	var members []*etcdMember
	var leader bool

	for _, line := range strings.Split(output, "\n") {
		items := strings.Split(line, ",")
		if len(items) < 3 {
			continue
		}
		if strings.TrimSpace(items[len(items)-1]) == "true" {
			leader = true
		}
		members = append(members, &etcdMember{
			id:     items[0],
			name:   items[2],
			leader: leader,
		})
	}

	return members
}

func getEtcdMembers(certDir string, r runner.Runner) []*etcdMember {
	cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl %v member list", getEtcdCertsOpts(certDir))
	output, err := r.RunCommand(utils.AddSudo(cmd))
	if err != nil {
		logrus.Errorf("get etcd members failed: %v\noutput: %v", err, output)
		return nil
	}
	return parseEtcdMemberList(output)
}

func getEtcdCertsOpts(certsPath string) string {
	return fmt.Sprintf("--cert=%v/etcd/server.crt --key=%v/etcd/server.key --cacert=%v/etcd/ca.crt",
		certsPath, certsPath, certsPath)
}

func removeEtcd(r runner.Runner, certDir string, id string) error {
	cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl %v member remove %v",
		getEtcdCertsOpts(certDir), id)
	if output, err := r.RunCommand(utils.AddSudo(cmd)); err != nil {
		logrus.Errorf("remove etcd %v failed: %v\noutput: %v", id, err, output)
		return err
	}
	return nil
}

func (t *removeEtcdsTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	var foundLeader bool

	etcds := getEtcdMembers(t.ccfg.GetCertDir(), r)
	for i, member := range etcds {
		// do not remove etcd leader
		if member.leader {
			foundLeader = true
			continue
		}

		// can not remove itself if only one etcd member exist
		if !foundLeader && i == len(etcds)-1 {
			continue
		}

		if err := removeEtcd(r, t.ccfg.GetCertDir(), member.id); err != nil {
			logrus.Errorf("remove etcd %v failed", member.id)
		}
	}

	return nil
}

func execRemoveEtcdsTask(conf *api.ClusterConfig, node string) error {
	taskRemoveEtcds := task.NewTaskInstance(
		&removeEtcdsTask{
			ccfg: conf,
		},
	)

	task.SetIgnoreErrorFlag(taskRemoveEtcds)
	if err := nodemanager.RunTaskOnNodes(taskRemoveEtcds, []string{node}); err != nil {
		logrus.Errorf("run task for remove etcds failed: %v", err)
		return err
	}

	if err := nodemanager.WaitNodesFinish([]string{node}, time.Minute*5); err != nil {
		logrus.Warnf("wait remove etcds task finish failed: %v", err)
		return err
	}

	return nil
}

func getFirstEtcd(nodes []*api.HostConfig) string {
	for _, node := range nodes {
		if utils.IsType(node.Type, api.ETCD) {
			return node.Address
		}
	}
	return ""
}

func ExecRemoveEtcdsTask(conf *api.ClusterConfig) error {
	if !conf.EtcdCluster.External {
		node := getFirstEtcd(conf.Nodes)
		if node != "" {
			return execRemoveEtcdsTask(conf, node)
		} else {
			logrus.Errorf("cann't found etcd node, ignore remove etcds")
		}
	} else {
		logrus.Info("external etcd, ignore remove etcds")
	}

	return nil
}
