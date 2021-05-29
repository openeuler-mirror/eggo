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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/etcdcluster"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
)

var osfuncs funcs

type funcs interface {
	unmount(target string, flags int) error
	removeAll(path string) error
}

type originFuncs struct {
}

func (f *originFuncs) unmount(target string, flags int) error {
	return syscall.Unmount(target, flags)
}

func (f *originFuncs) removeAll(path string) error {
	return os.RemoveAll(path)
}

type cleanupClusterTask struct {
	ccfg       *clusterdeployment.ClusterConfig
	r          runner.Runner
	hostConfig *clusterdeployment.HostConfig
}

func (t *cleanupClusterTask) Name() string {
	return "cleanupClusterTask"
}

func isType(curType uint16, expectedType uint16) bool {
	return curType&expectedType != 0
}

func umountKubeletSubDirs(kubeletDir string) error {
	mounts, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}

	// avoid umount kubelet directory
	if !strings.HasSuffix(kubeletDir, "/") {
		kubeletDir += "/"
	}

	lines := strings.Split(string(mounts), "\n")
	for _, line := range lines {
		items := strings.Split(line, " ")
		if len(items) < 2 || !strings.HasPrefix(items[1], kubeletDir) {
			continue
		}
		subDir := items[1]
		if err := osfuncs.unmount(subDir, 0); err != nil {
			logrus.Errorf("umount %v failed: %v", subDir, err)
		}
	}

	return nil
}

func addSudo(cmd string) string {
	return "sudo -E /bin/sh -c \"" + cmd + "\""
}

func removePathes(pathes []string) {
	for _, path := range pathes {
		if err := osfuncs.removeAll(path); err != nil {
			logrus.Errorf("remove path %v failed: %v", path, err)
		}
	}
}

func getKubeHomePath(savePath string) string {
	if savePath != "" {
		return filepath.Clean(savePath)
	} else {
		return clusterdeployment.DefaultCertPath
	}
}

func cleanupWorker(t *cleanupClusterTask) {
	// umount kubelet subdirs
	cleanupPathes := []string{
		getKubeHomePath(t.ccfg.Certificate.SavePath),
		"/etc/kubernetes", "/run/kubernetes",
		"/var/lib/cni", "/etc/cni", "/opt/cni",
		"/usr/lib/systemd/system/kubelet.service",
		"/usr/lib/systemd/system/kube-proxy.service",
	}
	if err := umountKubeletSubDirs("/var/lib/kubelet"); err == nil {
		cleanupPathes = append(cleanupPathes, "/var/lib/kubelet")
	}

	// remove directories
	removePathes(cleanupPathes)
}

func cleanupMaster(t *cleanupClusterTask) {
	// remove directories
	removePathes([]string{
		getKubeHomePath(t.ccfg.Certificate.SavePath),
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

func cleanupEtcd(t *cleanupClusterTask) {
	// remove directories
	removePathes([]string{
		getKubeHomePath(t.ccfg.Certificate.SavePath),
		getEtcdDataDir(t.ccfg.EtcdCluster.DataDir),
		"/etc/etcd",
		"/usr/lib/systemd/system/etcd.service",
		"/var/lib/etcd",
	})
}

func cleanupCoreDNS(t *cleanupClusterTask) {
	// remove directories
	removePathes([]string{
		getKubeHomePath(t.ccfg.Certificate.SavePath),
		"/usr/lib/systemd/system/coredns.service",
	})
}

func cleanupLoadBalance(t *cleanupClusterTask) {
	// remove directories
	removePathes([]string{
		getKubeHomePath(t.ccfg.Certificate.SavePath),
		"/etc/nginx", "/usr/lib/systemd/system/nginx.service",
	})
}

func postCleanup(t *cleanupClusterTask) {
	// save firewall config
	if output, err := t.r.RunCommand(addSudo("firewall-cmd --zone=public --runtime-to-permanent")); err != nil {
		logrus.Errorf("save firewall config on node %v failed: %v\noutput: %v",
			t.hostConfig.Address, err, output)
	}
}

func isPkgInstalled(t *cleanupClusterTask, pkg string) bool {
	for k := range t.hostConfig.Packages {
		if strings.HasPrefix(k, pkg) {
			return true
		}
	}
	return false
}

func (t *cleanupClusterTask) Run(r runner.Runner, hostConfig *clusterdeployment.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	t.r, t.hostConfig = r, hostConfig

	// TODO: call infrastructure function to cleanup

	if isType(hostConfig.Type, clusterdeployment.Worker) {
		cleanupWorker(t)
	}

	if isType(hostConfig.Type, clusterdeployment.Master) {
		cleanupMaster(t)
	}

	if isType(hostConfig.Type, clusterdeployment.ETCD) {
		if !t.ccfg.EtcdCluster.External {
			cleanupEtcd(t)
		} else {
			logrus.Info("external etcd, ignore remove etcds")
		}
	}

	if isPkgInstalled(t, "nginx") {
		cleanupLoadBalance(t)
	}
	if isPkgInstalled(t, "coredns") {
		cleanupCoreDNS(t)
	}

	postCleanup(t)

	return nil
}

func getAllIps(nodes []*clusterdeployment.HostConfig) []string {
	var ips []string

	for _, node := range nodes {
		ips = append(ips, node.Address)
	}

	return ips
}

type removeWorkersTask struct {
	ccfg       *clusterdeployment.ClusterConfig
	r          runner.Runner
	hostConfig *clusterdeployment.HostConfig
}

func (t *removeWorkersTask) Name() string {
	return "removeWorkersTask"
}

func getFirstMaster(nodes []*clusterdeployment.HostConfig) string {
	for _, node := range nodes {
		if isType(node.Type, clusterdeployment.Master) {
			return node.Address
		}
	}
	return ""
}

func runRemoveWorker(t *removeWorkersTask, worker string) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("kubectl drain %v --delete-local-data --force --ignore-daemonsets", worker))
	sb.WriteString(fmt.Sprintf("&& kubectl delete node %v", worker))
	if output, err := t.r.RunCommand(addSudo(sb.String())); err != nil {
		logrus.Errorf("remove workder %v failed: %v\noutput: %v", worker, err, output)
	}
}

func removeWorkers(t *removeWorkersTask) {
	for _, node := range t.ccfg.Nodes {
		if isType(node.Type, clusterdeployment.Worker) {
			runRemoveWorker(t, node.Name)
		}
	}
}

func (t *removeWorkersTask) Run(r runner.Runner, hostConfig *clusterdeployment.HostConfig) error {
	t.hostConfig, t.r = hostConfig, r

	// TODO: remove system resource

	// remove workers
	removeWorkers(t)

	return nil
}

type removeEtcdsTask struct {
	ccfg       *clusterdeployment.ClusterConfig
	r          runner.Runner
	hostConfig *clusterdeployment.HostConfig
}

func (t *removeEtcdsTask) Name() string {
	return "removeEtcdsTask"
}

func getFirstEtcd(nodes []*clusterdeployment.HostConfig) string {
	for _, node := range nodes {
		if isType(node.Type, clusterdeployment.ETCD) {
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

func getEtcdMembers(t *removeEtcdsTask) []*etcdMember {
	certsOpts := getEtcdCertsOpts(t.ccfg.Certificate.SavePath)
	cmd := fmt.Sprintf("etcdctl %v member list", certsOpts)
	output, err := t.r.RunCommand(addSudo(cmd))
	if err != nil {
		logrus.Errorf("get etcd members failed: %v\noutput: %v", err, output)
		return nil
	}
	return parseEtcdMemberList(output)
}

func getEtcdCertsOpts(savePath string) string {
	certsPath := clusterdeployment.DefaultCertPath
	if savePath != "" {
		certsPath = savePath
	}
	return fmt.Sprintf("--cert-file=%v/etcd/server.crt --key-file=%v/etcd/server.key --ca-file=%v/etcd/ca.key",
		certsPath, certsPath, certsPath)
}

func (t *removeEtcdsTask) Run(r runner.Runner, hostConfig *clusterdeployment.HostConfig) error {
	t.hostConfig, t.r = hostConfig, r

	for _, member := range getEtcdMembers(t) {
		// do not remove etcd member on this machine currently
		if member.name == t.hostConfig.Name {
			continue
		}
		certsOpts := getEtcdCertsOpts(t.ccfg.Certificate.SavePath)
		cmd := fmt.Sprintf("etcdctl %v member remove %v", certsOpts, member.id)
		if output, err := t.r.RunCommand(addSudo(cmd)); err != nil {
			logrus.Errorf("remove workder %v failed: %v\noutput: %v", member.name, err, output)
		}
	}

	return nil
}

func execRemoveWorkersTask(conf *clusterdeployment.ClusterConfig, node string) {
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

func execRemoveEtcdsTask(conf *clusterdeployment.ClusterConfig, node string) {
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

func Init(conf *clusterdeployment.ClusterConfig) error {
	osfuncs = &originFuncs{}

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
