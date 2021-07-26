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
	ccfg           *api.ClusterConfig
	reconfigType   string
	reconfigHost   *api.HostConfig
	initialCluster string
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

func getInitalCluster(output string) (string, error) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "ETCD_INITIAL_CLUSTER=") {
			result := strings.TrimSuffix(strings.TrimPrefix(line, "ETCD_INITIAL_CLUSTER="), "\r")
			if result == "" {
				return "", fmt.Errorf("found null initial cluster from output failed: %v", output)
			}
			return result, nil
		}
	}

	return "", fmt.Errorf("error found initial cluster from output: %v", output)
}

func (t *EtcdEtcdReconfigTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	if t.reconfigType == "remove" {
		etcds := getEtcdMembers(t.ccfg.GetCertDir(), r)
		if etcds == nil {
			return fmt.Errorf("get etcds failed")
		}

		id := getEtcdIDByName(etcds, t.reconfigHost.Name)
		if id == "" {
			// no need to delete if the etcd not exist
			return nil
		}

		if err := removeEtcd(r, t.ccfg.GetCertDir(), id); err != nil {
			return err
		}
	}
	if t.reconfigType == "add" {
		output, err := addEtcd(r, t.ccfg.GetCertDir(), t.reconfigHost.Name, t.reconfigHost.Address)
		if err != nil {
			return err
		}

		var initialCluster string
		initialCluster, err = getInitalCluster(output)
		if err != nil {
			return err
		}

		t.initialCluster = initialCluster
	}

	return nil
}

func etcdReconfig(conf *api.ClusterConfig, hostconfig *api.HostConfig, reconfigType string) (string, error) {
	if len(conf.EtcdCluster.Nodes) == 0 {
		return "", fmt.Errorf("invalid null etcd node")
	}

	// only one etcd member found, no need to notify etcd cluster to delete
	if len(conf.EtcdCluster.Nodes) == 1 && reconfigType == "remove" {
		return "", nil
	}

	t := &EtcdEtcdReconfigTask{
		ccfg:         conf,
		reconfigType: reconfigType,
		reconfigHost: hostconfig,
	}
	taskEtcdReconfig := task.NewTaskInstance(t)

	nodes := []string{conf.EtcdCluster.Nodes[0].Address}
	if err := nodemanager.RunTaskOnNodes(taskEtcdReconfig, nodes); err != nil {
		return "", fmt.Errorf("run task on nodes failed: %v", err)
	}

	if err := nodemanager.WaitNodesFinish(nodes, time.Minute); err != nil {
		return "", fmt.Errorf("wait for etcd reconfig task finish failed: %v", err)
	}

	return t.initialCluster, nil
}

func ExecRemoveMemberTask(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	if !conf.EtcdCluster.External {
		_, ret := etcdReconfig(conf, hostconfig, "remove")
		return ret
	} else {
		logrus.Info("external etcd, ignore remove etcds")
		return nil
	}
}

func ExecAddMemberTask(conf *api.ClusterConfig, hostconfig *api.HostConfig) (string, error) {
	if !conf.EtcdCluster.External {
		return etcdReconfig(conf, hostconfig, "add")
	} else {
		logrus.Info("external etcd, ignore add etcds")
		return "", nil
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
			id:     strings.TrimSpace(items[0]),
			name:   strings.TrimSpace(items[2]),
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
	logrus.Debugf("remove etcd command: %v", cmd)
	if output, err := r.RunCommand(utils.AddSudo(cmd)); err != nil {
		logrus.Errorf("remove etcd %v failed: %v\noutput: %v", id, err, output)
		return err
	}
	return nil
}

func addEtcd(r runner.Runner, certDir string, name string, ip string) (string, error) {
	cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl %v member add %v --peer-urls=https://%v:2380",
		getEtcdCertsOpts(certDir), name, ip)
	logrus.Debugf("add etcd command: %v", cmd)

	var err error
	var output string
	retry := 10
	for retry != 0 {
		if output, err = r.RunCommand(utils.AddSudo(cmd)); err == nil {
			return output, nil
		}
		retry--
		time.Sleep(3 * time.Second)
	}
	logrus.Errorf("add etcd %v failed: %v\noutput: %v", name, err, output)
	return "", err
}

func (t *removeEtcdsTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	etcds := getEtcdMembers(t.ccfg.GetCertDir(), r)
	for _, member := range etcds {
		// do not delete self
		if member.name == hostConfig.Name {
			continue
		}
		if err := removeEtcd(r, t.ccfg.GetCertDir(), member.id); err != nil {
			logrus.Errorf("remove etcd %v failed", member.id)
		}
	}

	return nil
}

func execRemoveEtcdsTask(conf *api.ClusterConfig, node string) error {
	taskRemoveEtcds := task.NewTaskIgnoreErrInstance(
		&removeEtcdsTask{
			ccfg: conf,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskRemoveEtcds, []string{node}); err != nil {
		logrus.Errorf("run task for remove etcds failed: %v", err)
		return err
	}

	if err := nodemanager.WaitNodesFinish([]string{node}, time.Minute*2); err != nil {
		logrus.Warnf("wait remove etcds task finish failed: %v", err)
		return err
	}

	return nil
}

type getEtcdLeaderTask struct {
	ccfg   *api.ClusterConfig
	leader string
}

func (t *getEtcdLeaderTask) Name() string {
	return "getEtcdLeaderTask"
}

func getFirstEtcd(nodes []*api.HostConfig) string {
	for _, node := range nodes {
		if utils.IsType(node.Type, api.ETCD) {
			return node.Address
		}
	}
	return ""
}

func getNodeIpByName(nodes []*api.HostConfig, name string) string {
	for _, node := range nodes {
		if node.Name == name {
			return node.Address
		}
	}
	return ""
}

func (t *getEtcdLeaderTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	etcds := getEtcdMembers(t.ccfg.GetCertDir(), r)
	for _, member := range etcds {
		if member.leader {
			t.leader = getNodeIpByName(t.ccfg.Nodes, member.name)
			return nil
		}
	}

	return nil
}

func getEtcdLeader(conf *api.ClusterConfig, node string) string {
	t := &getEtcdLeaderTask{ccfg: conf}
	taskGetEtcdLeader := task.NewTaskInstance(t)

	if err := nodemanager.RunTaskOnNodes(taskGetEtcdLeader, []string{node}); err != nil {
		logrus.Errorf("run task for get etcd leader failed: %v", err)
		return ""
	}

	if err := nodemanager.WaitNodesFinish([]string{node}, time.Minute*2); err != nil {
		logrus.Warnf("wait get etcd leader task finish failed: %v", err)
		return ""
	}

	return t.leader
}

func ExecRemoveEtcdsTask(conf *api.ClusterConfig) error {
	firstEtcdNode := getFirstEtcd(conf.Nodes)
	execNode := getEtcdLeader(conf, firstEtcdNode)
	if execNode == "" {
		execNode = firstEtcdNode
	}
	return execRemoveEtcdsTask(conf, execNode)
}
