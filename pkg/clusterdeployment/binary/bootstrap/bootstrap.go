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
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/commontools"
	"isula.org/eggo/pkg/clusterdeployment/binary/controlplane"
	"isula.org/eggo/pkg/clusterdeployment/runtime"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/certs"
	"isula.org/eggo/pkg/utils/endpoint"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"
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
	cluster  *api.ClusterConfig
}

func (gt *GetTokenTask) Name() string {
	return "GetTokenTask"
}

func (gt *GetTokenTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	token, err := commontools.GetBootstrapToken(r, gt.tokenStr,
		filepath.Join(gt.cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin), gt.cluster.GetManifestDir())
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

type NewWorkerTask struct {
	ccfg *api.ClusterConfig
}

func (it *NewWorkerTask) Name() string {
	return "NewWorkerTask"
}

func (it *NewWorkerTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("do join new worker...\n")

	// check worker dependences
	err := check(r, it.ccfg)
	if err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := prepareConfig(r, it.ccfg, hcg); err != nil {
		logrus.Errorf("prepare config failed: %v", err)
		return err
	}

	if _, err := r.RunCommand("sudo -E /bin/sh -c \"mkdir -p /var/lib/kubelet\""); err != nil {
		logrus.Errorf("mkdir /var/lib/kubelet failed: %v", err)
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
	if ccfg.WorkerConfig.KubeletConf == nil {
		return fmt.Errorf("empty kubeletconf")
	}
	if ccfg.WorkerConfig.ContainerEngineConf == nil {
		return fmt.Errorf("empty container engine conf")
	}
	if ccfg.APIEndpoint.AdvertiseAddress == "" {
		return fmt.Errorf("invalid endpoint")
	}

	checkTmpl := `
#!/bin/bash
if [ ! -f {{ .CAFile }} ]; then
	echo "{{ .CAFile }} not found" 1>&2
	exit 1
fi

if [ ! -f /etc/resolv.conf ]; then
	echo "/etc/resolv.conf not found, will cause kubelet start failed" 1>&2
	exit 1
fi

exit 0
`
	datastore := make(map[string]interface{})
	datastore["CAFile"] = filepath.Join(ccfg.Certificate.SavePath, "ca.crt")

	cmdStr, err := template.TemplateRender(checkTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "bootstrap-check")
	if err != nil {
		logrus.Errorf("bootstrap-check failed: %v\n", err)
		return err
	}

	return nil
}

func prepareConfig(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	apiEndpoint, err := endpoint.GetAPIServerEndpoint(ccfg)
	if err != nil {
		logrus.Errorf("get api server endpoint failed: %v", err)
		return err
	}

	token := getTokenString()
	if token == "" {
		return fmt.Errorf("get token failed")
	}

	if err := genKubeletBootstrapAndConfig(r, ccfg, token, apiEndpoint); err != nil {
		logrus.Errorf("generate kubelet bootstrap and config failed: %v", err)
		return err
	}

	if err := genProxyCertAndConfig(r, ccfg, hcf, apiEndpoint); err != nil {
		logrus.Errorf("generate proxy cert and kubeconfig failed: %v", err)
		return err
	}
	logrus.Debug("prepare bootstrap config success")

	return nil
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
	sb.WriteString("kubectl config set-cluster " + ccfg.Name +
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
		" --cluster=" + ccfg.Name +
		" --user=kubelet-bootstrap" +
		" --kubeconfig=kubelet-bootstrap.kubeconfig")
	sb.WriteString(" && ")
	sb.WriteString("kubectl config use-context default" +
		" --kubeconfig=kubelet-bootstrap.kubeconfig")
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
- {{ .DnsVip }}
clusterDomain: {{ .DnsDomain }}
rotateCertificates: true
runtimeRequestTimeout: "15m"
{{- if .EnableServer }}
serverTLSBootstrap: true
{{- end}}
`

	datastore := make(map[string]interface{})
	datastore["DnsVip"] = ccfg.WorkerConfig.KubeletConf.DNSVip
	datastore["DnsDomain"] = ccfg.WorkerConfig.KubeletConf.DNSDomain
	datastore["EnableServer"] = ccfg.WorkerConfig.KubeletConf.EnableServer

	config, err := template.TemplateRender(kubeletConfig, datastore)
	if err != nil {
		return err
	}
	cfgBase64 := base64.StdEncoding.EncodeToString([]byte(config))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", cfgBase64, "/etc/kubernetes/kubelet_config.yaml"))
	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}
	return nil
}

func genProxyCertAndConfig(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig, apiEndpoint string) error {
	if err := genProxyCert(r, ccfg, hcf); err != nil {
		logrus.Errorf("generate kube-proxy certs failed: %v", err)
		return err
	}

	if err := genProxyConfig(r, ccfg, apiEndpoint); err != nil {
		logrus.Errorf("generate proxy config failed: %v", err)
		return err
	}

	return nil
}

func genProxyCert(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	// TODO:
	//		generate kube proxy CSR and key on worker
	// 		transfer CSR to CA(eggo)
	//		CA generate cert by CSR and ca.key
	//		transfer cert to worker

	certPath := api.GetCertificateStorePath(ccfg.Name)
	certPrefix := KubeProxyKubeConfigName + "-" + hcf.Name
	certGen := certs.NewLocalCertGenerator()

	proxyConfig := &certs.CertConfig{
		CommonName: "system:kube-proxy",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", certPath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", certPath, RootCAName)
	if err := certGen.CreateCertAndKey(caCertPath, caKeyPath, proxyConfig, certPath, certPrefix); err != nil {
		logrus.Errorf("generate proxy cert and key failed for node %s: %v", hcf.Address, err)
		return err
	}

	if err := r.Copy(filepath.Join(certPath, certPrefix+".key"), filepath.Join(ccfg.Certificate.SavePath, KubeProxyKubeConfigName+".key")); err != nil {
		logrus.Errorf("copy cert: %s to host: %s failed: %v", certPrefix+".key", hcf.Name, err)
		return err
	}

	if err := r.Copy(filepath.Join(certPath, certPrefix+".crt"), filepath.Join(ccfg.Certificate.SavePath, KubeProxyKubeConfigName+".crt")); err != nil {
		logrus.Errorf("copy cert: %s to host: %s failed: %v", certPrefix+".key", hcf.Name, err)
		return err
	}
	logrus.Infof("copy certs to host: %s success", hcf.Name)

	return nil
}

func genProxyConfig(r runner.Runner, ccfg *api.ClusterConfig, apiEndpoint string) error {
	proxyConfig := `kind: KubeProxyConfiguration
apiVersion: kubeproxy.config.k8s.io/v1alpha1
clientConnection:
  kubeconfig: /etc/kubernetes/kube-proxy.conf
clusterCIDR: ` + ccfg.Network.PodCIDR + `
mode: "iptables"
`

	rootPath := ccfg.GetConfigDir()
	certPath := ccfg.GetCertDir()
	configGen := certs.NewOpensshBinCertGenerator(r)
	err := configGen.CreateKubeConfig(rootPath, KubeConfigFileNameKubeProxy, filepath.Join(certPath, "ca.crt"), ccfg.Name, "default-kube-proxy",
		filepath.Join(certPath, "kube-proxy.crt"), filepath.Join(certPath, "kube-proxy.key"), apiEndpoint)
	if err != nil {
		logrus.Errorf("generate proxy kube config failed: %v", err)
		return err
	}

	var sb strings.Builder
	cfgBase64 := base64.StdEncoding.EncodeToString([]byte(proxyConfig))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", cfgBase64, "/etc/kubernetes/kube-proxy-config.yaml"))
	if _, err := r.RunCommand(sb.String()); err != nil {
		return err
	}
	return nil
}

func JoinMaster(config *api.ClusterConfig, master *api.HostConfig) error {
	joinMasterTasks := []task.Task{
		task.NewTaskInstance(
			&commontools.CopyCaCertificatesTask{
				Cluster:  config,
				JoinType: master.Type,
			},
		),
		task.NewTaskInstance(
			controlplane.NewControlPlaneTask(config),
		),
	}

	if err := nodemanager.RunTasksOnNodes(joinMasterTasks, []string{master.Address}); err != nil {
		return err
	}

	return nil
}

func JoinWorker(config *api.ClusterConfig, controlPlane *api.HostConfig, worker *api.HostConfig) error {
	if tokenTask == nil {
		tokenTask = &GetTokenTask{
			cluster: config,
		}

		if err := nodemanager.RunTaskOnNodes(task.NewTaskInstance(tokenTask), []string{controlPlane.Address}); err != nil {
			return err
		}
		if err := nodemanager.WaitNodesFinish([]string{controlPlane.Address}, time.Minute*2); err != nil {
			return err
		}
	}

	joinWorkerTasks := []task.Task{
		task.NewTaskInstance(
			&commontools.CopyCaCertificatesTask{
				Cluster:  config,
				JoinType: worker.Type,
			},
		),
		task.NewTaskInstance(
			runtime.NewDeployRuntimeTask(config),
		),
		task.NewTaskInstance(
			&NewWorkerTask{
				ccfg: config,
			},
		),
	}

	if err := nodemanager.RunTasksOnNodes(joinWorkerTasks, []string{worker.Address}); err != nil {
		return err
	}

	return nil
}
