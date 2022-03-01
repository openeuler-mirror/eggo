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
 * Create: 2021-06-29
 * Description: eggo cleanup node binary implement
 ******************************************************************************/

package cleanupcluster

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/runtime"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

var (
	MasterService = []string{"kube-apiserver", "kube-controller-manager", "kube-scheduler"}
	WorkerService = []string{"kubelet", "kube-proxy"}
)

type cleanupNodeTask struct {
	ccfg    *api.ClusterConfig
	delType uint16
}

func (t *cleanupNodeTask) Name() string {
	return "cleanupNodeTask"
}

func umountKubeletSubDirs(r runner.Runner, kubeletDir string) error {
	output, err := r.RunCommand(utils.AddSudo("cat /proc/mounts"))
	if err != nil {
		logrus.Errorf("cat /proc/mounts failed: %v\noutput: %s", err, output)
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
		if output, err := r.RunCommand(utils.AddSudo("umount " + subDir)); err != nil {
			logrus.Errorf("umount %v failed: %v\noutput: %v",
				subDir, err, output)
		}
	}

	return nil
}

func getWorkerPathes(r runner.Runner, ccfg *api.ClusterConfig) []string {
	pathes := []string{
		filepath.Join(ccfg.GetConfigDir(), "kubelet"),
		filepath.Join(ccfg.GetConfigDir(), "kubelet.conf"),
		filepath.Join(ccfg.GetConfigDir(), "kubelet-bootstrap.kubeconfig"),
		filepath.Join(ccfg.GetConfigDir(), "kubelet_config.yaml"),
		filepath.Join(ccfg.GetConfigDir(), "kubelet.kubeconfig"),
		filepath.Join(ccfg.GetConfigDir(), "kube-proxy.conf"),
		filepath.Join(ccfg.GetConfigDir(), "kube-proxy-config.yaml"),
		filepath.Join(ccfg.GetCertDir(), "kube-proxy.crt"),
		filepath.Join(ccfg.GetCertDir(), "kube-proxy.key"),
		"/var/lib/cni", "/etc/cni", "/opt/cni",
		"/usr/lib/systemd/system/kubelet.service",
		"/usr/lib/systemd/system/kube-proxy.service",
	}
	runtime := runtime.GetRuntime(ccfg.WorkerConfig.ContainerEngineConf.Runtime)
	if runtime != nil {
		pathes = append(pathes, runtime.GetRemovedPath()...)
	} else {
		logrus.Errorf("invalid container engine %s", ccfg.WorkerConfig.ContainerEngineConf.Runtime)
	}

	if err := umountKubeletSubDirs(r, "/var/lib/kubelet"); err == nil {
		pathes = append(pathes, "/var/lib/kubelet")
	}

	return pathes
}

func getMasterPathes(ccfg *api.ClusterConfig) []string {
	return []string{
		filepath.Join(ccfg.GetConfigDir(), "admin.conf"),
		filepath.Join(ccfg.GetConfigDir(), "apiserver"),
		filepath.Join(ccfg.GetConfigDir(), "controller-manager"),
		filepath.Join(ccfg.GetConfigDir(), "controller-manager.conf"),
		filepath.Join(ccfg.GetConfigDir(), "encryption-config.yaml"),
		filepath.Join(ccfg.GetConfigDir(), "manifests"),
		filepath.Join(ccfg.GetCertDir(), "admin.crt"),
		filepath.Join(ccfg.GetCertDir(), "admin.key"),
		filepath.Join(ccfg.GetCertDir(), "apiserver.crt"),
		filepath.Join(ccfg.GetCertDir(), "apiserver-etcd-client.crt"),
		filepath.Join(ccfg.GetCertDir(), "apiserver-etcd-client.key"),
		filepath.Join(ccfg.GetCertDir(), "apiserver.key"),
		filepath.Join(ccfg.GetCertDir(), "apiserver-kubelet-client.crt"),
		filepath.Join(ccfg.GetCertDir(), "apiserver-kubelet-client.key"),
		filepath.Join(ccfg.GetCertDir(), "ca.key"),
		filepath.Join(ccfg.GetCertDir(), "ca.srl"),
		filepath.Join(ccfg.GetCertDir(), "controller-manager.crt"),
		filepath.Join(ccfg.GetCertDir(), "controller-manager.key"),
		filepath.Join(ccfg.GetCertDir(), "front-proxy-ca.crt"),
		filepath.Join(ccfg.GetCertDir(), "front-proxy-ca.key"),
		filepath.Join(ccfg.GetCertDir(), "front-proxy-ca.srl"),
		filepath.Join(ccfg.GetCertDir(), "front-proxy-client.crt"),
		filepath.Join(ccfg.GetCertDir(), "front-proxy-client.key"),
		filepath.Join(ccfg.GetCertDir(), "sa.key"),
		filepath.Join(ccfg.GetCertDir(), "sa.pub"),
		filepath.Join(ccfg.GetCertDir(), "scheduler.crt"),
		filepath.Join(ccfg.GetCertDir(), "scheduler.key"),
		filepath.Join(ccfg.GetConfigDir(), "proxy"),
		filepath.Join(ccfg.GetConfigDir(), "scheduler"),
		filepath.Join(ccfg.GetConfigDir(), "scheduler.conf"),
		"/usr/lib/systemd/system/kube-apiserver.service",
		"/usr/lib/systemd/system/kube-scheduler.service",
		"/usr/lib/systemd/system/kube-controller-manager.service",
	}
}

func getWorkerServices(runtimeName string) ([]string, error) {
	services := []string{}
	services = append(services, WorkerService...)

	r := runtime.GetRuntime(runtimeName)
	if r == nil {
		return nil, fmt.Errorf("invalid container engine %s", runtimeName)
	}
	return append(services, r.GetRuntimeService()), nil
}

func isAllNodeDeleted(nodeType uint16, delType uint16) bool {
	// &^ means clean bits
	remain := nodeType &^ delType
	return !utils.IsType(remain, api.Master) && !utils.IsType(remain, api.Worker)
}

func (t *cleanupNodeTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if hostConfig == nil {
		return fmt.Errorf("empty host config")
	}

	if utils.IsType(t.delType, api.Worker) {
		services, err := getWorkerServices(t.ccfg.WorkerConfig.ContainerEngineConf.Runtime)
		if err != nil {
			logrus.Errorf("get worker services failed")
		}

		if err := stopServices(r, services); err != nil {
			logrus.Warnf("stop service failed: %v", err)
		}
		removePathes(r, getWorkerPathes(r, t.ccfg))
	}

	if utils.IsType(t.delType, api.Master) {
		if err := stopServices(r, MasterService); err != nil {
			logrus.Warnf("stop master service failed: %v", err)
		}
		removePathes(r, getMasterPathes(t.ccfg))
	}

	// if master and worker are all delted, delete the shared files
	if isAllNodeDeleted(hostConfig.Type, t.delType) {
		removePathes(r, []string{filepath.Join(t.ccfg.GetCertDir(), "ca.crt")})
	}

	PostCleanup(r)

	return nil
}

type removeWorkerTask struct {
	ccfg       *api.ClusterConfig
	workerName string
}

func (t *removeWorkerTask) Name() string {
	return "removeWorkerTask"
}

func getFirstMaster(nodes []*api.HostConfig) string {
	for _, node := range nodes {
		if utils.IsType(node.Type, api.Master) {
			return node.Address
		}
	}
	return ""
}

func runRemoveWorker(configDir string, r runner.Runner, worker string) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("KUBECONFIG=%s/%s kubectl delete node %v --force --grace-period=0",
		configDir, constants.KubeConfigFileNameAdmin, worker))
	if output, err := r.RunCommand(utils.AddSudo(sb.String())); err != nil {
		logrus.Errorf("remove workder %v failed: %v\noutput: %v", worker, err, output)
		return err
	}

	return nil
}

func (t *removeWorkerTask) Run(r runner.Runner, hostConfig *api.HostConfig) error {
	if err := runRemoveWorker(t.ccfg.GetConfigDir(), r, t.workerName); err != nil {
		return err
	}

	return nil
}

func execRemoveWorkerTask(conf *api.ClusterConfig, hostconfig *api.HostConfig) error {
	taskRemoveWorker := task.NewTaskIgnoreErrInstance(
		&removeWorkerTask{
			ccfg:       conf,
			workerName: hostconfig.Name,
		},
	)

	master := getFirstMaster(conf.Nodes)
	if master == "" {
		return fmt.Errorf("failed to get first master")
	}

	if err := nodemanager.RunTaskOnNodes(taskRemoveWorker, []string{master}); err != nil {
		return err
	}

	return nil
}

func CleanupNode(conf *api.ClusterConfig, hostconfig *api.HostConfig, delType uint16) error {
	if conf == nil || hostconfig == nil {
		return fmt.Errorf("invalid null config")
	}

	if utils.IsType(delType, api.Worker) {
		if err := execRemoveWorkerTask(conf, hostconfig); err != nil {
			logrus.Warnf("ignore: remove workers failed: %v", err)
		}
	}

	taskCleanupNode := task.NewTaskIgnoreErrInstance(
		&cleanupNodeTask{
			ccfg:    conf,
			delType: delType,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskCleanupNode, []string{hostconfig.Address}); err != nil {
		return fmt.Errorf("run task for cleanup cluster failed: %v", err)
	}

	return nil
}
