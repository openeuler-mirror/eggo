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
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/addons"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/etcdcluster"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/infrastructure"
	"gitee.com/openeuler/eggo/pkg/utils"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
)

type cleanupClusterTask struct {
	ccfg *api.ClusterConfig
}

func (t *cleanupClusterTask) Name() string {
	return "cleanupClusterTask"
}

func umountKubeletSubDirs(r runner.Runner, hostConfig *api.HostConfig, kubeletDir string) error {
	output, err := r.RunCommand(addSudo("cat /proc/mounts"))
	if err != nil {
		logrus.Errorf("cat /proc/mounts on node %v failed: %v\noutput: %v",
			hostConfig.Address, err, output)
		return err
	}

	// avoid umount kubelet directory
	if !strings.HasSuffix(kubeletDir, "/") {
		kubeletDir += "/"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		items := strings.Split(line, " ")
		if len(items) < 2 || !strings.HasPrefix(items[1], kubeletDir) {
			continue
		}
		subDir := items[1]
		if output, err := r.RunCommand(addSudo("umount " + subDir)); err != nil {
			logrus.Errorf("umount %v on node %v failed: %v\noutput: %v",
				subDir, hostConfig.Address, err, output)
		}
	}

	return nil
}

func addSudo(cmd string) string {
	return "sudo -E /bin/sh -c \"" + cmd + "\""
}

func removePathes(r runner.Runner, hostConfig *api.HostConfig, pathes []string) {
	for _, path := range pathes {
		// TODO: dot not delete user configed directory, delete directories and files we addded only
		if output, err := r.RunCommand(addSudo("rm -rf " + path)); err != nil {
			logrus.Errorf("remove path %v on node %v failed: %v\noutput: %v",
				path, hostConfig.Address, err, output)
		}
	}
}

func cleanupWorker(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig) {
	// umount kubelet subdirs
	cleanupPathes := []string{
		ccfg.GetConfigDir(),
		ccfg.GetCertDir(),
		"/etc/kubernetes", "/run/kubernetes",
		"/var/lib/cni", "/etc/cni", "/opt/cni",
		"/usr/lib/systemd/system/kubelet.service",
		"/usr/lib/systemd/system/kube-proxy.service",
	}
	if err := umountKubeletSubDirs(r, hostConfig, "/var/lib/kubelet"); err == nil {
		cleanupPathes = append(cleanupPathes, "/var/lib/kubelet")
	}

	// remove directories
	removePathes(r, hostConfig, cleanupPathes)
}

func cleanupMaster(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig) {
	// remove directories
	removePathes(r, hostConfig, []string{
		ccfg.GetConfigDir(),
		ccfg.GetCertDir(),
		"/etc/kubernetes", "/run/kubernetes",
		"/usr/lib/systemd/system/kube-apiserver.service",
		"/usr/lib/systemd/system/kube-scheduler.service",
		"/usr/lib/systemd/system/kube-controller-manager.service",
	})
}

func getEtcdDataDir(dataDir string) string {
	if dataDir != "" {
		return dataDir
	}
	return etcdcluster.DefaultEtcdDataDir
}

func cleanupEtcd(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig) {
	// remove directories
	removePathes(r, hostConfig, []string{
		ccfg.GetConfigDir(),
		ccfg.GetCertDir(),
		getEtcdDataDir(ccfg.EtcdCluster.DataDir),
		"/etc/etcd",
		"/usr/lib/systemd/system/etcd.service",
		"/var/lib/etcd",
	})
}

func cleanupCoreDNS(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig) {
	// remove directories
	removePathes(r, hostConfig, []string{
		ccfg.GetConfigDir(),
		ccfg.GetCertDir(),
		"/usr/lib/systemd/system/coredns.service",
	})
}

func cleanupLoadBalance(ccfg *api.ClusterConfig, r runner.Runner, hostConfig *api.HostConfig) {
	// remove directories
	removePathes(r, hostConfig, []string{
		ccfg.GetConfigDir(),
		ccfg.GetCertDir(),
		"/etc/nginx", "/usr/lib/systemd/system/nginx.service",
	})
}

func postCleanup(r runner.Runner, hostConfig *api.HostConfig) {
	// save firewall config
	if output, err := r.RunCommand(addSudo("firewall-cmd --runtime-to-permanent")); err != nil {
		logrus.Errorf("save firewall config on node %v failed: %v\noutput: %v",
			hostConfig.Address, err, output)
	}

	// daemon-reload
	if output, err := r.RunCommand(addSudo("systemctl daemon-reload")); err != nil {
		logrus.Errorf("daemon-reload on node %v failed: %v\noutput: %v",
			hostConfig.Address, err, output)
	}
}

func isPkgInstalled(hostConfig *api.HostConfig, pkg string) bool {
	for _, p := range hostConfig.Packages {
		if strings.HasPrefix(p.Name, pkg) {
			return true
		}
	}
	return false
}

func (t *cleanupClusterTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	ports := []string{}

	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	// call infrastructure function to cleanup
	if err := infrastructure.RemoveDependences(r, hostConfig, t.ccfg.PackageSrc); err != nil {
		logrus.Errorf("remove dependences failed: %v", err)
	}

	if utils.IsType(hostConfig.Type, api.Worker) {
		cleanupWorker(t.ccfg, r, hostConfig)
		ports = append(ports, infrastructure.WorkPorts...)
	}

	if utils.IsType(hostConfig.Type, api.Master) {
		cleanupMaster(t.ccfg, r, hostConfig)
		ports = append(ports, infrastructure.MasterPorts...)
	}

	if utils.IsType(hostConfig.Type, api.ETCD) {
		if !t.ccfg.EtcdCluster.External {
			cleanupEtcd(t.ccfg, r, hostConfig)
			ports = append(ports, infrastructure.EtcdPosts...)
		} else {
			logrus.Info("external etcd, ignore remove etcds")
		}
	}

	for _, p := range hostConfig.OpenPorts {
		port := strconv.Itoa(p.Port) + "/" + p.Protocol
		ports = append(ports, port)
	}
	shieldPorts(r, ports...)

	if isPkgInstalled(hostConfig, "nginx") {
		cleanupLoadBalance(t.ccfg, r, hostConfig)
	}
	if isPkgInstalled(hostConfig, "coredns") {
		cleanupCoreDNS(t.ccfg, r, hostConfig)
	}

	postCleanup(r, hostConfig)

	return nil
}

func shieldPorts(r runner.Runner, ports ...string) {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for _, p := range ports {
		sb.WriteString(fmt.Sprintf("firewall-cmd --zone=public --remove-port=%s && ", p))
	}

	sb.WriteString("firewall-cmd --runtime-to-permanent \"")
	if _, err := r.RunCommand(sb.String()); err != nil {
		logrus.Errorf("shield port failed: %v", err)
	}
}

func getAllIps(nodes []*api.HostConfig) []string {
	var ips []string

	for _, node := range nodes {
		ips = append(ips, node.Address)
	}

	return ips
}

type removeWorkersTask struct {
	ccfg *api.ClusterConfig
}

func (t *removeWorkersTask) Name() string {
	return "removeWorkersTask"
}

func getFirstMaster(nodes []*api.HostConfig) string {
	for _, node := range nodes {
		if utils.IsType(node.Type, api.Master) {
			return node.Address
		}
	}
	return ""
}

func runRemoveWorker(t *removeWorkersTask, r runner.Runner, worker string) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("kubectl drain %v --delete-emptydir-data --force --ignore-daemonsets", worker))
	sb.WriteString(fmt.Sprintf("&& kubectl delete node %v", worker))
	if output, err := r.RunCommand(addSudo(sb.String())); err != nil {
		logrus.Errorf("remove workder %v failed: %v\noutput: %v", worker, err, output)
	}
}

func removeWorkers(t *removeWorkersTask, r runner.Runner) {
	for _, node := range t.ccfg.Nodes {
		if utils.IsType(node.Type, api.Worker) {
			runRemoveWorker(t, r, node.Name)
		}
	}
}

func (t *removeWorkersTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	// TODO: remove system resource

	// remove workers
	removeWorkers(t, r)

	return nil
}

type removeEtcdsTask struct {
	ccfg *api.ClusterConfig
}

func (t *removeEtcdsTask) Name() string {
	return "removeEtcdsTask"
}

func getFirstEtcd(nodes []*api.HostConfig) string {
	for _, node := range nodes {
		if utils.IsType(node.Type, api.ETCD) {
			return node.Address
		}
	}
	return ""
}

type etcdMember struct {
	id   string
	name string
}

// output:
// 868b499159f00586, started, workder0, https://192.168.0.1:2380, https://192.168.0.1:2379, false
// 6787454327e00766, started, workder1, https://192.168.0.2:2380, https://192.168.0.2:2379, true
func parseEtcdMemberList(output string) []*etcdMember {
	var members []*etcdMember

	for _, line := range strings.Split(output, "\n") {
		items := strings.Split(line, ",")
		if len(items) < 3 {
			continue
		}
		members = append(members, &etcdMember{
			id:   items[0],
			name: items[2],
		})
	}

	return members
}

func getEtcdMembers(t *removeEtcdsTask, r runner.Runner) []*etcdMember {
	cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl %v member list", getEtcdCertsOpts(t.ccfg.GetCertDir()))
	output, err := r.RunCommand(addSudo(cmd))
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

func (t *removeEtcdsTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	for _, member := range getEtcdMembers(t, r) {
		// do not remove etcd member on this machine currently
		if member.name == hostConfig.Name {
			continue
		}
		cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl %v member remove %v",
			getEtcdCertsOpts(t.ccfg.GetCertDir()), member.id)
		if output, err := r.RunCommand(addSudo(cmd)); err != nil {
			logrus.Errorf("remove workder %v failed: %v\noutput: %v", member.name, err, output)
		}
	}

	return nil
}

func execRemoveWorkersTask(conf *api.ClusterConfig, node string) {
	taskRemoveWorkers := task.NewTaskInstance(
		&removeWorkersTask{
			ccfg: conf,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskRemoveWorkers, []string{node}); err != nil {
		logrus.Errorf("run task for remove workers failed: %v", err)
		return
	}

	if err := nodemanager.WaitTaskOnNodesFinished(taskRemoveWorkers, []string{node}, time.Second*60*5); err != nil {
		logrus.Errorf("wait for remove workers task finish failed: %v", err)
	}
}

func execRemoveEtcdsTask(conf *api.ClusterConfig, node string) {
	taskRemoveEtcds := task.NewTaskInstance(
		&removeEtcdsTask{
			ccfg: conf,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskRemoveEtcds, []string{node}); err != nil {
		logrus.Errorf("run task for remove etcds failed: %v", err)
		return
	}

	if err := nodemanager.WaitTaskOnNodesFinished(taskRemoveEtcds, []string{node}, time.Second*60); err != nil {
		logrus.Errorf("wait for remove etcds task finish failed: %v", err)
	}
}

func Init(conf *api.ClusterConfig) error {
	// first cleanup addons
	if err := addons.CleanupAddons(conf); err != nil {
		logrus.Errorf("cleanup addons failed: %v", err)
	}

	// remove workers from master
	node := getFirstMaster(conf.Nodes)
	if node != "" {
		execRemoveWorkersTask(conf, node)
	} else {
		logrus.Errorf("cann't found master node, ignore remove workers")
	}

	// TODO: support execute kubectl with eggo
	// remove etcd members
	if !conf.EtcdCluster.External {
		node = getFirstEtcd(conf.Nodes)
		if node != "" {
			execRemoveEtcdsTask(conf, node)
		} else {
			logrus.Errorf("cann't found etcd node, ignore remove etcds")
		}
	} else {
		logrus.Info("external etcd, ignore remove etcds")
	}

	// cleanup cluster
	taskCleanupCluster := task.NewTaskInstance(
		&cleanupClusterTask{
			ccfg: conf,
		},
	)

	nodes := getAllIps(conf.Nodes)
	if err := nodemanager.RunTaskOnNodes(taskCleanupCluster, nodes); err != nil {
		return fmt.Errorf("run task for cleanup cluster failed: %v", err)
	}

	if err := nodemanager.WaitTaskOnNodesFinished(taskCleanupCluster, nodes, time.Second*60*5); err != nil {
		return fmt.Errorf("wait for cleanup cluster task finish failed: %v", err)
	}

	return nil
}
