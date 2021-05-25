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
 * Create: 2021-05-11
 * Description: eggo controlplane binary implement
 ******************************************************************************/

package controlplane

import (
	"crypto/x509"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"time"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/commontools"
	"gitee.com/openeuler/eggo/pkg/utils/certs"
	"gitee.com/openeuler/eggo/pkg/utils/endpoint"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

const (
	RootCAName                      = "ca"
	FrontProxyCAName                = "front-proxy-ca"
	APIServerCertName               = "apiserver"
	APIServerKubeletName            = "apiserver-kubelet-client"
	FrontProxyClientName            = "front-proxy-client"
	AdminKubeConfigName             = "admin"
	KubeletKubeConfigName           = "kubelet"
	ControllerManagerKubeConfigName = "controller-manager"
	SchedulerKubeConfigName         = "scheduler"
	KubeProxyKubeConfigName         = "kube-proxy"

	KubeConfigFileNameAdmin      = "admin.conf"
	KubeConfigFileNameKubelet    = "kubelet.conf"
	KubeConfigFileNameController = "controller-manager.conf"
	KubeConfigFileNameScheduler  = "scheduler.conf"
	KubeConfigFileNameKubeProxy  = "kube-proxy.conf"
)

var (
	KubeSoftwares = []string{"kubectl", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}
)

type ControlPlaneTask struct {
	ccfg *clusterdeployment.ClusterConfig
}

var (
	ctask *task.TaskInstance
)

func (ct *ControlPlaneTask) Name() string {
	return "ControlplaneTask"
}

func (ct *ControlPlaneTask) Run(r runner.Runner, hcf *clusterdeployment.HostConfig) error {
	if hcf == nil {
		return fmt.Errorf("empty cluster config")
	}

	// do precheck phase
	err := check(r)
	if err != nil {
		return err
	}

	// generate certificates and kubeconfigs
	if err = generateCertsAndKubeConfigs(r, ct.ccfg); err != nil {
		return err
	}

	// run services of k8s
	if err = runKubernetesServices(r, ct.ccfg, hcf); err != nil {
		return err
	}

	return nil
}

func check(r runner.Runner) error {
	// check dependences softwares
	for _, s := range KubeSoftwares {
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"which %s\"", s))
		if err != nil {
			logrus.Errorf("chech kubernetes software: %s, failed: %v\n", s, err)
			return err
		}
		logrus.Debugf("check kubernetes software: %s success\n", s)
	}
	return nil
}

func generateApiServerCertificate(savePath string, cg certs.CertGenerator, ccfg *clusterdeployment.ClusterConfig) error {
	ips := []string{"0.0.0.0", "127.0.0.1"}
	dnsnames := []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster", "kubernetes.default.svc.cluster.local"}

	if ccfg.ServiceCluster.Gateway != "" {
		ips = append(ips, ccfg.ServiceCluster.Gateway)
	}
	if ccfg.ControlPlane.ApiConf != nil {
		ips = append(ips, ccfg.ControlPlane.ApiConf.CertSans.IPs...)
		dnsnames = append(dnsnames, ccfg.ControlPlane.ApiConf.CertSans.DNSNames...)
	}

	apiserverConfig := &certs.CertConfig{
		CommonName:   "kube-apiserver",
		Organization: "kubernetes",
		AltNames: certs.AltNames{
			IPs:      ips,
			DNSNames: dnsnames,
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, apiserverConfig, savePath, APIServerCertName)
}

func generateApiServerKubeletCertificate(savePath string, cg certs.CertGenerator) error {
	apiserverConfig := &certs.CertConfig{
		CommonName:   "kube-apiserver-kubelet-client",
		Organization: "system:masters",
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, apiserverConfig, savePath, APIServerKubeletName)
}

func generateFrontProxyClientCertificate(savePath string, cg certs.CertGenerator) error {
	apiserverConfig := &certs.CertConfig{
		CommonName: "front-proxy-client",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, FrontProxyCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, FrontProxyCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, apiserverConfig, savePath, FrontProxyClientName)
}

func generateAdminCertificate(savePath string, cg certs.CertGenerator) error {
	adminConfig := &certs.CertConfig{
		CommonName:   "kubernetes-admin",
		Organization: "system:masters",
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, adminConfig, savePath, AdminKubeConfigName)
}

func generateKubeletCertificate(savePath string, nodeName string, cg certs.CertGenerator) error {
	adminConfig := &certs.CertConfig{
		CommonName:   fmt.Sprintf("system:node:%s", nodeName),
		Organization: "system:nodes",
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, adminConfig, savePath, nodeName+"-"+KubeletKubeConfigName)
}

func generateControllerManagerCertificate(savePath string, cg certs.CertGenerator) error {
	controllerConfig := &certs.CertConfig{
		CommonName: "system:kube-controller-manager",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, controllerConfig, savePath, ControllerManagerKubeConfigName)
}

func generateSchedulerCertificate(savePath string, cg certs.CertGenerator) error {
	controllerConfig := &certs.CertConfig{
		CommonName: "system:kube-scheduler",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, controllerConfig, savePath, SchedulerKubeConfigName)
}

func generateKubeProxyCertificate(savePath string, cg certs.CertGenerator) error {
	// TODO: maybe just to use service account
	controllerConfig := &certs.CertConfig{
		CommonName: "system:kube-proxy",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, controllerConfig, savePath, KubeProxyKubeConfigName)
}

func generateCerts(savePath string, cg certs.CertGenerator, ccfg *clusterdeployment.ClusterConfig) (err error) {
	// create root ca
	caConfig := &certs.CertConfig{
		CommonName: "kubernetes",
	}
	if err = cg.CreateCA(caConfig, savePath, RootCAName); err != nil {
		return
	}

	// create front proxy ca
	frontCaConfig := &certs.CertConfig{
		CommonName: "front-proxy-ca",
	}
	if err = cg.CreateCA(frontCaConfig, savePath, FrontProxyCAName); err != nil {
		return
	}

	// create certificate and keys

	if err = generateApiServerCertificate(savePath, cg, ccfg); err != nil {
		return
	}

	if err = generateApiServerKubeletCertificate(savePath, cg); err != nil {
		return
	}

	return generateFrontProxyClientCertificate(savePath, cg)
}

func getEndpoint(ccfg *clusterdeployment.ClusterConfig) (string, error) {
	host, sport, err := net.SplitHostPort(ccfg.ControlPlane.Endpoint)
	if err != nil {
		host = ccfg.LocalEndpoint.AdvertiseAddress
		sport = strconv.Itoa(int(ccfg.LocalEndpoint.BindPort))
	}
	port, err := endpoint.ParsePort(sport)
	if err != nil {
		return "", err
	}
	return endpoint.GetEndpoint(host, port)
}

func generateKubeConfigs(rootPath, certPath string, cg certs.CertGenerator, ccfg *clusterdeployment.ClusterConfig) (err error) {
	// create temp certificates and keys for kubeconfigs
	if err = generateAdminCertificate(certPath, cg); err != nil {
		return
	}
	apiEndpoint, err := getEndpoint(ccfg)
	if err != nil {
		return
	}
	localEndpoint, err := endpoint.GetEndpoint(ccfg.LocalEndpoint.AdvertiseAddress, int(ccfg.LocalEndpoint.BindPort))
	if err != nil {
		return
	}
	err = cg.CreateKubeConfig(rootPath, KubeConfigFileNameAdmin, filepath.Join(certPath, "ca.crt"), "default-admin",
		filepath.Join(certPath, "admin.key"), filepath.Join(certPath, "admin.crt"), apiEndpoint)
	if err != nil {
		return
	}

	for _, node := range ccfg.Nodes {
		if err = generateKubeletCertificate(certPath, node.Name, cg); err != nil {
			return
		}
		err = cg.CreateKubeConfig(rootPath, node.Name+"-"+KubeConfigFileNameKubelet, filepath.Join(certPath, "ca.crt"), "default-auth",
			filepath.Join(certPath, node.Name+"-kubelet.key"), filepath.Join(certPath, node.Name+"-kubelet.crt"), apiEndpoint)
		if err != nil {
			return
		}
	}

	if err = generateControllerManagerCertificate(certPath, cg); err != nil {
		return
	}
	err = cg.CreateKubeConfig(rootPath, KubeConfigFileNameController, filepath.Join(certPath, "ca.crt"), "default-controller-manager",
		filepath.Join(certPath, "controller-manager.key"), filepath.Join(certPath, "controller-manager.crt"), localEndpoint)
	if err != nil {
		return
	}

	if err = generateSchedulerCertificate(certPath, cg); err != nil {
		return
	}
	err = cg.CreateKubeConfig(rootPath, KubeConfigFileNameScheduler, filepath.Join(certPath, "ca.crt"), "default-scheduler",
		filepath.Join(certPath, "scheduler.key"), filepath.Join(certPath, "scheduler.crt"), localEndpoint)
	if err != nil {
		return
	}

	if err = generateKubeProxyCertificate(certPath, cg); err != nil {
		return err
	}
	return cg.CreateKubeConfig(rootPath, KubeConfigFileNameKubeProxy, filepath.Join(certPath, "ca.crt"), "default-kube-proxy",
		filepath.Join(certPath, "kube-proxy.key"), filepath.Join(certPath, "kube-proxy.crt"), apiEndpoint)
}

func generateCertsAndKubeConfigs(r runner.Runner, ccfg *clusterdeployment.ClusterConfig) (err error) {
	rootPath := certs.DefaultKubeHomePath
	if ccfg.Certificate.SavePath != "" {
		if filepath.IsAbs(ccfg.Certificate.SavePath) {
			return fmt.Errorf("certifacates store path: '%s' must be absolute", ccfg.Certificate.SavePath)
		}
		rootPath = filepath.Clean(ccfg.Certificate.SavePath)
	}
	certPath := filepath.Join(rootPath, "pki")

	cg := certs.NewOpensshBinCertGenerator(r)
	defer func() {
		if err != nil {
			cg.CleanAll(rootPath)
		}
	}()

	if err = cg.CreateServiceAccount(certPath); err != nil {
		return
	}

	// clean generated certifactes
	if err = generateCerts(certPath, cg, ccfg); err != nil {
		return
	}

	return generateKubeConfigs(rootPath, certPath, cg, ccfg)
}

func runKubernetesServices(r runner.Runner, ccfg *clusterdeployment.ClusterConfig, hcf *clusterdeployment.HostConfig) error {
	// set up api-server service
	if err := commontools.SetupControlplaneServices(r, ccfg, hcf); err != nil {
		return err
	}

	return nil
}

func Init(conf *clusterdeployment.ClusterConfig) error {
	ctask = task.NewTaskInstance(
		&ControlPlaneTask{
			ccfg: conf,
		},
	)

	for _, node := range conf.Nodes {
		if node.Type&clusterdeployment.Master != 0 {
			if err := nodemanager.RunTaskOnNodes(ctask, []string{node.Address}); err != nil {
				logrus.Errorf("init first control plane master failed: %v", err)
				return err
			}
			err := nodemanager.WaitTaskOnNodesFinished(ctask, []string{node.Address}, time.Minute*5)
			if err != nil {
				logrus.Errorf("wait to init first control plane master finish failed: %v", err)
				return err
			}
			// TODO: join other master use bootstrap
			break
		}
	}

	// TODO: upload certificates and kubeconfigs into cluster

	return nil
}
