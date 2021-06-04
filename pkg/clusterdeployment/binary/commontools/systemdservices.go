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
 * Author: haozi007
 * Create: 2021-06-07
 * Description: util for generate system service
 ******************************************************************************/
package commontools

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/template"
	"github.com/sirupsen/logrus"
)

const (
	SystemdServiceConfigPath = "/usr/lib/systemd/system"
)

func SetupAPIServerService(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	defaultArgs := map[string]string{
		"--advertise-address":                  hcf.Address,
		"--allow-privileged":                   "true",
		"--authorization-mode":                 "Node,RBAC",
		"--enable-admission-plugins":           "NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota",
		"--secure-port":                        "6443",
		"--enable-bootstrap-token-auth":        "true",
		"--etcd-cafile":                        "/etc/kubernetes/pki/etcd/ca.crt",
		"--etcd-certfile":                      "/etc/kubernetes/pki/apiserver-etcd-client.crt",
		"--etcd-keyfile":                       "/etc/kubernetes/pki/apiserver-etcd-client.key",
		"--etcd-servers":                       api.GetEtcdServers(&ccfg.EtcdCluster),
		"--client-ca-file":                     "/etc/kubernetes/pki/ca.crt",
		"--kubelet-client-certificate":         "/etc/kubernetes/pki/apiserver-kubelet-client.crt",
		"--kubelet-client-key":                 "/etc/kubernetes/pki/apiserver-kubelet-client.key",
		"--kubelet-https":                      "true",
		"--proxy-client-cert-file":             "/etc/kubernetes/pki/front-proxy-client.crt",
		"--proxy-client-key-file":              "/etc/kubernetes/pki/front-proxy-client.key",
		"--tls-cert-file":                      "/etc/kubernetes/pki/apiserver.crt",
		"--tls-private-key-file":               "/etc/kubernetes/pki/apiserver.key",
		"--service-cluster-ip-range":           ccfg.ServiceCluster.CIDR,
		"--service-account-issuer":             "https://kubernetes.default.svc.cluster.local",
		"--service-account-key-file":           "/etc/kubernetes/pki/sa.pub",
		"--service-account-signing-key-file":   "/etc/kubernetes/pki/sa.key",
		"--service-node-port-range":            "30000-32767",
		"--requestheader-allowed-names":        "front-proxy-client",
		"--requestheader-client-ca-file":       "/etc/kubernetes/pki/front-proxy-ca.crt",
		"--requestheader-extra-headers-prefix": "X-Remote-Extra-",
		"--requestheader-group-headers":        "X-Remote-Group",
		"--requestheader-username-headers":     "X-Remote-User",
		"--encryption-provider-config":         "/etc/kubernetes/encryption-config.yaml",
	}
	if ccfg.ControlPlane.ApiConf != nil {
		for k, v := range ccfg.ControlPlane.ApiConf.ExtraArgs {
			defaultArgs[k] = v
		}
	}

	var args []string
	for k, v := range defaultArgs {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	conf := &template.SystemdServiceConfig{
		Description:   "Kubernetes API Server",
		Documentation: "https://kubernetes.io/docs/reference/generated/kube-apiserver/",
		Afters:        []string{"network.target", "etcd.service"},
		Command:       "/usr/bin/kube-apiserver",
		Arguments:     args,
	}
	serviceConf, err := template.CreateSystemdServiceTemplate("api-server-systemd", conf)
	if err != nil {
		logrus.Errorf("create api-server systemd service config failed: %v", err)
		return err
	}
	var sb strings.Builder
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", csrBase64, filepath.Join(SystemdServiceConfigPath, "kube-apiserver.service")))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	_, err = r.RunCommand("sudo systemctl enable kube-apiserver")
	if err != nil {
		return err
	}
	return nil
}

func SetupControllerManagerService(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	defaultArgs := map[string]string{
		"--bind-address":                     "0.0.0.0",
		"--cluster-cidr":                     ccfg.Network.PodCIDR,
		"--cluster-name":                     "kubernetes",
		"--cluster-signing-cert-file":        "/etc/kubernetes/pki/ca.crt",
		"--cluster-signing-key-file":         "/etc/kubernetes/pki/ca.key",
		"--kubeconfig":                       "/etc/kubernetes/controller-manager.conf",
		"--leader-elect":                     "true",
		"--root-ca-file":                     "/etc/kubernetes/pki/ca.crt",
		"--service-account-private-key-file": "/etc/kubernetes/pki/sa.key",
		"--service-cluster-ip-range":         ccfg.ServiceCluster.CIDR,
		"--use-service-account-credentials":  "true",
		"--authentication-kubeconfig":        "/etc/kubernetes/controller-manager.conf",
		"--authorization-kubeconfig":         "/etc/kubernetes/controller-manager.conf",
		"--requestheader-client-ca-file":     "/etc/kubernetes/pki/front-proxy-ca.crt",
		"--controllers":                      "*,bootstrapsigner,tokencleaner",
		"--v":                                "2",
	}
	if ccfg.ControlPlane.ManagerConf != nil {
		for k, v := range ccfg.ControlPlane.ManagerConf.ExtraArgs {
			defaultArgs[k] = v
		}
	}

	var args []string
	for k, v := range defaultArgs {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	conf := &template.SystemdServiceConfig{
		Description:   "Kubernetes Controller Manager",
		Documentation: "https://kubernetes.io/docs/reference/generated/kube-controller-manager/",
		Command:       "/usr/bin/kube-controller-manager",
		Arguments:     args,
	}
	serviceConf, err := template.CreateSystemdServiceTemplate("controller-manager-systemd", conf)
	if err != nil {
		logrus.Errorf("create controller-manager systemd service config failed: %v", err)
		return err
	}
	var sb strings.Builder
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", csrBase64, filepath.Join(SystemdServiceConfigPath, "kube-controller-manager.service")))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	_, err = r.RunCommand("sudo systemctl enable kube-controller-manager")
	if err != nil {
		return err
	}
	return nil
}

func SetupSchedulerService(r runner.Runner, ccfg *api.ClusterConfig) error {
	defaultArgs := map[string]string{
		"--kubeconfig":                "/etc/kubernetes/scheduler.conf",
		"--authentication-kubeconfig": "/etc/kubernetes/scheduler.conf",
		"--authorization-kubeconfig":  "/etc/kubernetes/scheduler.conf",
		"--leader-elect":              "true",
		"--v":                         "2",
	}
	if ccfg.ControlPlane.SchedulerConf != nil {
		for k, v := range ccfg.ControlPlane.SchedulerConf.ExtraArgs {
			defaultArgs[k] = v
		}
	}

	var args []string
	for k, v := range defaultArgs {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	conf := &template.SystemdServiceConfig{
		Description:   "Kubernetes Scheduler Plugin",
		Documentation: "https://kubernetes.io/docs/reference/generated/kube-scheduler/",
		Command:       "/usr/bin/kube-scheduler",
		Arguments:     args,
	}
	serviceConf, err := template.CreateSystemdServiceTemplate("kube-scheduler-systemd", conf)
	if err != nil {
		logrus.Errorf("create kube-scheduler systemd service config failed: %v", err)
		return err
	}
	var sb strings.Builder
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"sudo echo %s | base64 -d > %s\"", csrBase64, filepath.Join(SystemdServiceConfigPath, "kube-scheduler.service")))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	_, err = r.RunCommand("sudo systemctl enable kube-scheduler")
	if err != nil {
		return err
	}
	return nil
}

func SetupMasterServices(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	// set up api-server service
	if err := SetupAPIServerService(r, ccfg, hcf); err != nil {
		logrus.Errorf("setup api server service failed: %v", err)
		return err
	}

	if err := SetupControllerManagerService(r, ccfg, hcf); err != nil {
		logrus.Errorf("setup k8s controller manager service failed: %v", err)
		return err
	}

	if err := SetupSchedulerService(r, ccfg); err != nil {
		logrus.Errorf("setup k8s scheduler service failed: %v", err)
		return err
	}

	_, err := r.RunCommand("sudo systemctl start kube-apiserver kube-controller-manager kube-scheduler")
	if err != nil {
		logrus.Errorf("start k8s master services failed: %v", err)
	}
	logrus.Info("setup k8s master services success")
	return nil
}

func SetupKubeletService(r runner.Runner, kcf *api.Kubelet, hcf *api.HostConfig) error {
	defaultArgs := map[string]string{
		"--config":               "/etc/kubernetes/kubelet_config.yaml",
		"--network-plugin":       "cni",
		"--kubeconfig":           "/etc/kubernetes/kubelet.kubeconfig",
		"--bootstrap-kubeconfig": "/etc/kubernetes/kubelet-bootstrap.kubeconfig",
		"--register-node":        "true",
		"--hostname-override":    hcf.Name,
		"--v":                    "2",
	}

	if kcf != nil {
		configArgs := map[string]string{
			"--network-plugin":            kcf.NetworkPlugin,
			"--cni-bin-dir":               kcf.CniBinDir,
			"--pod-infra-container-image": kcf.PauseImage,
		}

		if !utils.IsDocker(kcf.Runtime) {
			configArgs["--container-runtime"] = "remote"
			configArgs["--container-runtime-endpoint"] = kcf.RuntimeEndpoint
		}

		for k, v := range kcf.ExtraArgs {
			defaultArgs[k] = v
		}
		for k, v := range configArgs {
			if v != "" {
				defaultArgs[k] = v
			}
		}
	}

	var args []string
	for k, v := range defaultArgs {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	conf := &template.SystemdServiceConfig{
		Description:   "The Kubernetes Node Agent",
		Documentation: "https://kubernetes.io/docs/reference/generated/kubelet/",
		Afters:        []string{"network-online.target"},
		Command:       "/usr/bin/kubelet",
		Arguments:     args,
		ExecStartPre:  []string{"swapoff -a"},
	}
	serviceConf, err := template.CreateSystemdServiceTemplate("kubelet-systemd", conf)
	if err != nil {
		logrus.Errorf("create kubelet systemd service config failed: %v", err)
		return err
	}
	var sb strings.Builder
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", csrBase64, filepath.Join(SystemdServiceConfigPath, "kubelet.service")))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	_, err = r.RunCommand("sudo systemctl enable kubelet")
	if err != nil {
		return err
	}
	return nil
}

func SetupProxyService(r runner.Runner, kpcf *api.KubeProxy, hcf *api.HostConfig) error {
	defaultArgs := map[string]string{
		"--config":            "/etc/kubernetes/kube-proxy-config.yaml",
		"--hostname-override": hcf.Name,
		"--logtostderr":       "true",
		"--v":                 "2",
	}
	if kpcf != nil {
		for k, v := range kpcf.ExtraArgs {
			defaultArgs[k] = v
		}
	}

	var args []string
	for k, v := range defaultArgs {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	conf := &template.SystemdServiceConfig{
		Description:   "Kubernetes Kube-Proxy Server",
		Documentation: "https://kubernetes.io/docs/reference/generated/kube-proxy/",
		Command:       "/usr/bin/kube-proxy",
		Arguments:     args,
	}
	serviceConf, err := template.CreateSystemdServiceTemplate("proxy-systemd", conf)
	if err != nil {
		logrus.Errorf("create proxy systemd service config failed: %v", err)
		return err
	}
	var sb strings.Builder
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", csrBase64, filepath.Join(SystemdServiceConfigPath, "kube-proxy.service")))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	_, err = r.RunCommand("sudo systemctl enable kube-proxy")
	if err != nil {
		return err
	}
	return nil
}

func SetupWorkerServices(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	// set up k8s worker service
	if err := SetupKubeletService(r, ccfg.ControlPlane.KubeletConf, hcf); err != nil {
		logrus.Errorf("setup k8s kubelet service failed: %v", err)
		return err
	}

	if err := SetupProxyService(r, ccfg.ControlPlane.ProxyConf, hcf); err != nil {
		logrus.Errorf("setup k8s proxy service failed: %v", err)
		return err
	}

	_, err := r.RunCommand("sudo -E /bin/sh -c \"systemctl start kubelet kube-proxy\"")
	if err != nil {
		logrus.Errorf("start k8s worker services failed: %v", err)
	}
	logrus.Info("setup k8s worker services success")
	return nil
}
