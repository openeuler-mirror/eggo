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
	ControllerManagerKubeConfigName = "controller-manager"
	SchedulerKubeConfigName         = "scheduler"

	KubeConfigFileNameAdmin      = "admin.conf"
	KubeConfigFileNameController = "controller-manager.conf"
	KubeConfigFileNameScheduler  = "scheduler.conf"
)

var (
	KubeSoftwares = []string{"kubectl", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}
)

type ControlPlaneTask struct {
	ccfg *clusterdeployment.ClusterConfig
}

func (ct *ControlPlaneTask) Name() string {
	return "ControlplaneTask"
}

func (ct *ControlPlaneTask) Run(r runner.Runner, hcf *clusterdeployment.HostConfig) error {
	if hcf == nil {
		return fmt.Errorf("empty cluster config")
	}

	// do precheck phase
	err := check(r, ct.ccfg.Certificate.SavePath)
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

func check(r runner.Runner, savePath string) error {
	// check dependences softwares
	for _, s := range KubeSoftwares {
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"which %s\"", s))
		if err != nil {
			logrus.Errorf("chech kubernetes software: %s, failed: %v\n", s, err)
			return err
		}
		logrus.Debugf("check kubernetes software: %s success\n", s)
	}
	for _, ca := range commontools.CommonCaCerts {
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"ls %s\"", filepath.Join(savePath, ca)))
		if err != nil {
			logrus.Errorf("chech ca cert: %s, failed: %v\n", ca, err)
			return err
		}
		logrus.Debugf("chech ca cert: %s success\n", ca)
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
		CommonName:    "kube-apiserver",
		Organizations: []string{"kubernetes"},
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
		CommonName:    "kube-apiserver-kubelet-client",
		Organizations: []string{"system:masters"},
		Usages:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
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
		CommonName:    "kubernetes-admin",
		Organizations: []string{"system:masters"},
		Usages:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, RootCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, RootCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, adminConfig, savePath, AdminKubeConfigName)
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

func generateCerts(savePath string, cg certs.CertGenerator, ccfg *clusterdeployment.ClusterConfig) (err error) {
	// create certificate and keys

	if err = generateApiServerCertificate(savePath, cg, ccfg); err != nil {
		return
	}

	if err = generateApiServerKubeletCertificate(savePath, cg); err != nil {
		return
	}

	return generateFrontProxyClientCertificate(savePath, cg)
}

func prepareCAs(savePath string) error {
	lcg := certs.NewLocalCertGenerator()

	if _, err := lcg.RunCommand(fmt.Sprintf("sudo mkdir -p -m 0700 %s", savePath)); err != nil {
		logrus.Errorf("prepare certificates store path failed: %v", err)
		return err
	}

	if err := lcg.CreateServiceAccount(savePath); err != nil {
		return err
	}
	// create root ca
	caConfig := &certs.CertConfig{
		CommonName: "kubernetes",
	}
	if err := lcg.CreateCA(caConfig, savePath, RootCAName); err != nil {
		return err
	}

	// create front proxy ca
	frontCaConfig := &certs.CertConfig{
		CommonName: "front-proxy-ca",
	}
	if err := lcg.CreateCA(frontCaConfig, savePath, FrontProxyCAName); err != nil {
		return err
	}

	return nil
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

	return cg.CreateKubeConfig(rootPath, KubeConfigFileNameScheduler, filepath.Join(certPath, "ca.crt"), "default-scheduler",
		filepath.Join(certPath, "scheduler.key"), filepath.Join(certPath, "scheduler.crt"), localEndpoint)
}

func generateCertsAndKubeConfigs(r runner.Runner, ccfg *clusterdeployment.ClusterConfig) (err error) {
	rootPath := ccfg.GetConfigDir()
	certPath := ccfg.GetCertDir()

	cg := certs.NewOpensshBinCertGenerator(r)
	defer func() {
		if err != nil {
			cg.CleanAll(rootPath)
		}
	}()

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

func JoinMaterNode(conf *clusterdeployment.ClusterConfig, masterNode *clusterdeployment.HostConfig) error {
	joinMasterTasks := []task.Task{
		task.NewTaskInstance(
			&commontools.CopyCaCertificatesTask{
				Cluster: conf,
			},
		),
		task.NewTaskInstance(
			&ControlPlaneTask{
				ccfg: conf,
			},
		),
	}

	err := nodemanager.RunTasksOnNode(joinMasterTasks, masterNode.Address)
	if err != nil {
		return err
	}

	if err := nodemanager.WaitTasksOnNodeFinished(joinMasterTasks, masterNode.Address, time.Minute*5); err != nil {
		logrus.Errorf("wait to init first control plane master finish failed: %v", err)
		return err
	}

	return nil
}

func Init(conf *clusterdeployment.ClusterConfig) error {
	var firstMaster *clusterdeployment.HostConfig
	for _, node := range conf.Nodes {
		if node.Type&clusterdeployment.Master != 0 {
			firstMaster = node
			break
		}
	}

	// generate ca certificates in eggo
	err := prepareCAs(clusterdeployment.GetCertificateStorePath(conf.Name))
	if err != nil {
		logrus.Errorf("[certs] create ca certificates failed: %v", err)
		return err
	}

	return JoinMaterNode(conf, firstMaster)
}
