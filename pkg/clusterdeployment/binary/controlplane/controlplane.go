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
	"sync"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/certs"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	"github.com/sirupsen/logrus"
)

const (
	RootCAName           = "ca"
	FrontProxyCAName     = "front-proxy-ca"
	APIServerCertName    = "apiserver"
	APIServerKubeletName = "apiserver-kubelet-client"
	FrontProxyClientName = "front-proxy-client"
)

var (
	KubeSoftwares = []string{"kubectl", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}
)

type ControlPlaneTask struct {
	ccfg *clusterdeployment.ClusterConfig
}

var (
	ctask *task.TaskInstance
	lock  sync.Mutex
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

	// generate certificates
	if err = generateCerts(r, ct.ccfg); err != nil {
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
	ips = append(ips, ccfg.ControlPlane.ApiConf.Sans.IPs...)
	dnsnames = append(dnsnames, ccfg.ControlPlane.ApiConf.Sans.DNSNames...)

	// create api server certificate and key
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
	// create api server certificate and key
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
	// create api server certificate and key
	apiserverConfig := &certs.CertConfig{
		CommonName: "front-proxy-client",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/%s.crt", savePath, FrontProxyCAName)
	caKeyPath := fmt.Sprintf("%s/%s.key", savePath, FrontProxyCAName)
	return cg.CreateCertAndKey(caCertPath, caKeyPath, apiserverConfig, savePath, FrontProxyClientName)
}

func generateCerts(r runner.Runner, ccfg *clusterdeployment.ClusterConfig) error {
	savePath := certs.DefaultCertPath
	if ccfg.Certificate.SavePath != "" {
		savePath = ccfg.Certificate.SavePath
	}
	cg := certs.NewOpensshBinCertGenerator(r)
	if err := cg.CreateServiceAccount(savePath); err != nil {
		return err
	}

	// create root ca
	caConfig := &certs.CertConfig{
		CommonName: "kubernetes",
	}
	if err := cg.CreateCA(caConfig, savePath, RootCAName); err != nil {
		return err
	}

	// create front proxy ca
	frontCaConfig := &certs.CertConfig{
		CommonName: "front-proxy-ca",
	}
	if err := cg.CreateCA(frontCaConfig, savePath, FrontProxyCAName); err != nil {
		return err
	}

	// create certificate and keys

	if err := generateApiServerCertificate(savePath, cg, ccfg); err != nil {
		return err
	}

	if err := generateApiServerKubeletCertificate(savePath, cg); err != nil {
		return err
	}

	return generateFrontProxyClientCertificate(savePath, cg)
}

func generateKubeconfigs() error {
	return nil
}

func runKubernetesServices() error {
	return nil
}

func Init(conf *clusterdeployment.ClusterConfig) error {
	lock.Lock()
	defer lock.Unlock()
	if ctask != nil {
		return nil
	}
	ctask = task.NewTaskInstance(
		&ControlPlaneTask{
			ccfg: conf,
		},
	)

	// TODO: run task on every controlplane node
	return nil
}
