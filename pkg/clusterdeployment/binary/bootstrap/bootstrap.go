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
 * Author: zhangxiaoyu
 * Create: 2021-05-22
 * Description: eggo bootstrap binary implement
 ******************************************************************************/

package bootstrap

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/commontools"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/controlplane"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/infrastructure"
	"gitee.com/openeuler/eggo/pkg/utils"
	"gitee.com/openeuler/eggo/pkg/utils/certs"
	"gitee.com/openeuler/eggo/pkg/utils/endpoint"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

const (
	RootCAName                  = "ca"
	KubeProxyKubeConfigName     = "kube-proxy"
	KubeConfigFileNameKubeProxy = "kube-proxy.conf"
)

var (
	KubeWorkerSoftwares = []string{"kubelet", "kube-proxy", "kubectl"}
	tokenTask           *GetTokenTask
)

type GetTokenTask struct {
	tokenStr string
}

func (gt *GetTokenTask) Name() string {
	return "GetTokenTask"
}

func (gt *GetTokenTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	token, err := commontools.GetBootstrapToken(r, gt.tokenStr)
	if err != nil {
		return err
	}
	gt.tokenStr = token
	return nil
}

func getTokenString() string {
	if tokenTask == nil {
		return ""
	}
	return tokenTask.tokenStr
}

type BootstrapTask struct {
	ccfg *api.ClusterConfig
}

func (it *BootstrapTask) Name() string {
	return "BootstrapTask"
}

func (it *BootstrapTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("do join new worker...\n")

	// check worker dependences
	if err := check(r, it.ccfg); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := prepareISulad(r, it.ccfg); err != nil {
		logrus.Errorf("prepare Env failed: %v", err)
		return err
	}

	if err := prepareConfig(r, it.ccfg, hcg); err != nil {
		logrus.Errorf("prepare config failed: %v", err)
		return err
	}

	if err := commontools.SetupWorkerServices(r, it.ccfg, hcg); err != nil {
		logrus.Errorf("run service failed: %v", err)
		return err
	}

	logrus.Info("join worker success\n")
	return nil
}

func check(r runner.Runner, ccfg *api.ClusterConfig) error {
	if ccfg.ControlPlane.KubeletConf == nil {
		return fmt.Errorf("empty kubeletconf")
	}

	var softwares []string
	if utils.IsISulad(ccfg.ControlPlane.KubeletConf.Runtime) {
		softwares = append(KubeWorkerSoftwares, "isula", "isulad")
	} else if utils.IsDocker(ccfg.ControlPlane.KubeletConf.Runtime) {
		softwares = append(KubeWorkerSoftwares, "docker", "dockerd")
	} else {
		return fmt.Errorf("invalid container engine %s", ccfg.ControlPlane.KubeletConf.Runtime)
	}

	if err := infrastructure.CheckDependences(r, softwares); err != nil {
		return err
	}

	_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"ls %s\"", filepath.Join(ccfg.Certificate.SavePath, "ca.crt")))
	if err != nil {
		logrus.Errorf("chech ca cert failed: %v\n", err)
		return err
	}

	return nil
}

func prepareISulad(r runner.Runner, ccfg *api.ClusterConfig) error {
	if !utils.IsISulad(ccfg.ControlPlane.KubeletConf.Runtime) {
		return nil
	}

	pauseImage, cniBinDir := "k8s.gcr.io/pause:3.2", "/usr/libexec/cni"
	if ccfg.ControlPlane.KubeletConf.PauseImage != "" {
		pauseImage = ccfg.ControlPlane.KubeletConf.PauseImage
	}

	if ccfg.ControlPlane.KubeletConf.CniBinDir != "" {
		cniBinDir = ccfg.ControlPlane.KubeletConf.CniBinDir
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString("sed -i '/registry-mirrors/a\\    \t\"docker.io\"' /etc/isulad/daemon.json && ")
	sb.WriteString("sed -i '/insecure-registries/a\\    \t\"quay.io\"' /etc/isulad/daemon.json && ")
	sb.WriteString("sed -i '/insecure-registries/a\\    \t\"k8s.gcr.io\",' /etc/isulad/daemon.json && ")
	sb.WriteString(fmt.Sprintf("sed -i 's#pod-sandbox-image\\\": \\\"#pod-sandbox-image\\\": \\\"%s#g' /etc/isulad/daemon.json && ", pauseImage))
	sb.WriteString("sed -i 's#network-plugin\\\": \\\"#network-plugin\\\": \\\"cni#g' /etc/isulad/daemon.json && ")
	sb.WriteString(fmt.Sprintf("sed -i 's#cni-bin-dir\\\": \\\"#cni-bin-dir\\\": \\\"%s#g' /etc/isulad/daemon.json && ", cniBinDir))
	sb.WriteString("systemctl restart isulad\"")

	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}

	return nil
}

func prepareConfig(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	if err := genConfig(r, ccfg, hcf, getTokenString()); err != nil {
		logrus.Errorf("generate config failed: %v", err)
		return err
	}

	return nil
}

func genConfig(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig, token string) error {
	apiEndpoint, err := getEndpoint(ccfg)
	if err != nil {
		logrus.Errorf("get api server endpoint failed: %v", err)
		return err
	}

	if err := genKubeletBootstrapAndConfig(r, ccfg, token, apiEndpoint); err != nil {
		logrus.Errorf("generate kubelet bootstrap and config failed: %v", err)
		return err
	}

	if err := genProxyCertAndConfig(r, ccfg, apiEndpoint); err != nil {
		logrus.Errorf("generate proxy cert and kubeconfig failed: %v", err)
		return err
	}

	return nil
}

func getEndpoint(ccfg *api.ClusterConfig) (string, error) {
	host, sport, err := net.SplitHostPort(ccfg.ControlPlane.Endpoint)
	if err != nil {
		// TODO: get ready master by master status list
		for _, n := range ccfg.Nodes {
			if n.Type&api.Master != 0 {
				host = n.Address
				sport = strconv.Itoa(int(ccfg.LocalEndpoint.BindPort))
				break
			}
		}
	}
	if host == "" || sport == "" {
		return "", fmt.Errorf("invalid host or sport")
	}
	port, err := endpoint.ParsePort(sport)
	if err != nil {
		return "", err
	}
	return endpoint.GetEndpoint(host, port)
}

func genKubeletBootstrapAndConfig(r runner.Runner, ccfg *api.ClusterConfig, token, apiEndpoint string) error {
	if err := genKubeletBootstrap(r, ccfg, token, apiEndpoint); err != nil {
		logrus.Errorf("generate kubelet bootstrap failed: %v", err)
		return err
	}

	if err := genKubeletConfig(r, ccfg); err != nil {
		logrus.Errorf("generate kubelet config failed: %v", err)
		return err
	}

	return nil
}

func genKubeletBootstrap(r runner.Runner, ccfg *api.ClusterConfig, token, apiEndpoint string) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"cd /etc/kubernetes/ && ")
	sb.WriteString("kubectl config set-cluster kubernetes" +
		" --certificate-authority=/etc/kubernetes/pki/ca.crt" +
		" --embed-certs=true" +
		" --server=" + apiEndpoint +
		" --kubeconfig=kubelet-bootstrap.kubeconfig")
	sb.WriteString(" && ")
	sb.WriteString("kubectl config set-credentials kubelet-bootstrap" +
		" --token=" + token +
		" --kubeconfig=kubelet-bootstrap.kubeconfig")
	sb.WriteString(" && ")
	sb.WriteString("kubectl config set-context default" +
		" --cluster=kubernetes" +
		" --user=kubelet-bootstrap" +
		" --kubeconfig=kubelet-bootstrap.kubeconfig")
	sb.WriteString(" && ")
	sb.WriteString("kubectl config use-context default" +
		" --kubelet-bootstrap.kubeconfig")
	sb.WriteString("\"")

	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}

	return nil
}

func genKubeletConfig(r runner.Runner, ccfg *api.ClusterConfig) error {
	kubeletConfig := `apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
clusterDNS:
- ` + ccfg.ControlPlane.KubeletConf.DnsVip + `
clusterDomain: ` + ccfg.ControlPlane.KubeletConf.DnsDomain + `
runtimeRequestTimeout: "15m"
	`

	var sb strings.Builder
	cfgBase64 := base64.StdEncoding.EncodeToString([]byte(kubeletConfig))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", cfgBase64, "/etc/kubernetes/kubelet_config.yaml"))
	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}
	return nil
}

func genProxyCertAndConfig(r runner.Runner, ccfg *api.ClusterConfig, apiEndpoint string) (err error) {
	rootPath := ccfg.GetConfigDir()
	certPath := ccfg.GetCertDir()

	cg := certs.NewOpensshBinCertGenerator(r)
	defer func() {
		if err != nil {
			cg.CleanAll(rootPath)
		}
	}()

	if err = genProxyCert(certPath, cg); err != nil {
		logrus.Errorf("generate proxy cert failed: %v", err)
		return
	}

	err = cg.CreateKubeConfig(rootPath, KubeConfigFileNameKubeProxy, filepath.Join(certPath, "ca.crt"), "default-kube-proxy",
		filepath.Join(certPath, "kube-proxy.key"), filepath.Join(certPath, "kube-proxy.crt"), apiEndpoint)
	if err != nil {
		logrus.Errorf("generate proxy kube config failed: %v", err)
		return
	}

	if err = genProxyConfig(r, ccfg); err != nil {
		logrus.Errorf("generate proxy config failed: %v", err)
		return
	}

	return
}

func genProxyCert(savePath string, cg certs.CertGenerator) error {
	// TODO:
	//		generate kube proxy CSR and key on worker
	// 		transfer CSR to CA(eggo)
	//		CA generate cert by CSR and ca.key
	//		transfer cert to worker

	proxyConfig := &certs.CertConfig{
		CommonName: "system:kube-proxy",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, proxyConfig, savePath, KubeProxyKubeConfigName)
}

func genProxyConfig(r runner.Runner, ccfg *api.ClusterConfig) error {
	proxyConfig := `kind: KubeProxyConfiguration
apiVersion: kubeproxy.config.k8s.io/v1alpha1
clientConnection:
  kubeconfig: /etc/kubernetes/kube-proxy.conf
clusterCIDR: ` + ccfg.Network.PodCIDR + `
mode: "iptables"
		`

	var sb strings.Builder
	cfgBase64 := base64.StdEncoding.EncodeToString([]byte(proxyConfig))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", cfgBase64, "/etc/kubernetes/kube-proxy-config.yaml"))
	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}
	return nil
}

func JoinNode(config *api.ClusterConfig, masters, workers []string) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}
	if len(masters) == 0 && len(workers) == 0 {
		logrus.Warn("empty join node")
		return nil
	}

	joinMasterTasks := []task.Task{
		task.NewTaskInstance(
			&commontools.CopyCaCertificatesTask{
				Cluster: config,
			},
		),
		task.NewTaskInstance(
			controlplane.NewControlPlaneTask(config),
		),
	}

	joinWorkerTasks := []task.Task{
		task.NewTaskInstance(
			&commontools.CopyCaCertificatesTask{
				Cluster: config,
			},
		),
		task.NewTaskInstance(
			&BootstrapTask{
				ccfg: config,
			},
		),
	}

	if err := nodemanager.RunTasksOnNodes(joinMasterTasks, masters); err != nil {
		return err
	}

	if err := nodemanager.RunTasksOnNodes(joinWorkerTasks, workers); err != nil {
		return err
	}

	if err := nodemanager.WaitTasksOnNodesFinished(joinMasterTasks, masters, time.Minute*5); err != nil {
		logrus.Errorf("wait to join masters finish failed: %v", err)
		return err
	}

	if err := nodemanager.WaitTasksOnNodesFinished(joinWorkerTasks, workers, time.Minute*5); err != nil {
		logrus.Errorf("wait to join workers finish failed: %v", err)
		return err
	}

	return nil
}

func Init(config *api.ClusterConfig) error {
	masters, workers := []string{}, []string{}
	for _, node := range config.Nodes {
		if node.Type&api.Master != 0 {
			masters = append(masters, node.Address)
		}
		if node.Type&api.Worker != 0 {
			workers = append(workers, node.Address)
		}
	}
	if len(masters) == 0 {
		return fmt.Errorf("no master found for cluster")
	}

	if tokenTask == nil {
		tokenTask = &GetTokenTask{}
	}
	if err := nodemanager.RunTasksOnNode([]task.Task{task.NewTaskInstance(tokenTask)}, masters[0]); err != nil {
		return err
	}

	return JoinNode(config, masters[1:], workers)
}
