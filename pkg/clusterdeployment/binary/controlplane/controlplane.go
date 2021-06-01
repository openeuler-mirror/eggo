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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/commontools"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/infrastructure"
	"gitee.com/openeuler/eggo/pkg/constants"
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

	AdminRoleConfig = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
	annotations:
	rbac.authorization.kubernetes.io/autoupdate: "true"
	labels:
	kubernetes.io/bootstrapping: rbac-defaults
	name: system:kube-apiserver-to-kubelet
rules:
	- apiGroups:
		- ""
	resources:
		- nodes/proxy
		- nodes/stats
		- nodes/log
		- nodes/spec
		- nodes/metrics
	verbs:
		- "*"
`
	AdminRoleBindConfig = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
	name: system:kube-apiserver
	namespace: ""
roleRef:
	apiGroup: rbac.authorization.k8s.io
	kind: ClusterRole
	name: system:kube-apiserver-to-kubelet
subjects:
	- apiGroup: rbac.authorization.k8s.io
	kind: User
	name: kubernetes
`
)

var (
	KubeSoftwares = []string{"kubectl", "kube-apiserver", "kube-controller-manager", "kube-scheduler"}
)

type ControlPlaneTask struct {
	ccfg *api.ClusterConfig
}

func NewControlPlaneTask(ccf *api.ClusterConfig) *ControlPlaneTask {
	return &ControlPlaneTask{
		ccfg: ccf,
	}
}

func (ct *ControlPlaneTask) Name() string {
	return "ControlplaneTask"
}

func (ct *ControlPlaneTask) Run(r runner.Runner, hcf *api.HostConfig) error {
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
	if err := infrastructure.CheckDependences(r, KubeSoftwares); err != nil {
		return err
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

func generateApiServerCertificate(savePath string, cg certs.CertGenerator, ccfg *api.ClusterConfig) error {
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

func generateCerts(savePath string, cg certs.CertGenerator, ccfg *api.ClusterConfig) (err error) {
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

func generateKubeConfigs(rootPath, certPath string, cg certs.CertGenerator, ccfg *api.ClusterConfig) (err error) {
	// create temp certificates and keys for kubeconfigs
	if err = generateAdminCertificate(certPath, cg); err != nil {
		return
	}
	apiEndpoint, err := endpoint.GetAPIServerEndpoint(ccfg.ControlPlane.Endpoint, ccfg.LocalEndpoint)
	if err != nil {
		return
	}
	localEndpoint, err := endpoint.GetEndpoint(ccfg.LocalEndpoint.AdvertiseAddress, int(ccfg.LocalEndpoint.BindPort))
	if err != nil {
		return
	}

	err = cg.CreateKubeConfig(rootPath, constants.KubeConfigFileNameAdmin, filepath.Join(certPath, "ca.crt"), "default-admin",
		filepath.Join(certPath, "admin.key"), filepath.Join(certPath, "admin.crt"), apiEndpoint)
	if err != nil {
		return
	}

	if err = generateControllerManagerCertificate(certPath, cg); err != nil {
		return
	}
	err = cg.CreateKubeConfig(rootPath, constants.KubeConfigFileNameController, filepath.Join(certPath, "ca.crt"), "default-controller-manager",
		filepath.Join(certPath, "controller-manager.key"), filepath.Join(certPath, "controller-manager.crt"), localEndpoint)
	if err != nil {
		return
	}

	if err = generateSchedulerCertificate(certPath, cg); err != nil {
		return
	}

	return cg.CreateKubeConfig(rootPath, constants.KubeConfigFileNameScheduler, filepath.Join(certPath, "ca.crt"), "default-scheduler",
		filepath.Join(certPath, "scheduler.key"), filepath.Join(certPath, "scheduler.crt"), localEndpoint)
}

func generateEncryption(r runner.Runner, savePath string) error {
	const encry = `kind: EncryptionConfig
apiVersion: v1
resources:
	- resources:
		- secrets
	providers:
		- aescbc:
			keys:
			- name: key1
				secret: ${ENCRYPTION_KEY}
		- identity: {}
`
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString("local ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64)")
	encryBase64 := base64.StdEncoding.EncodeToString([]byte(encry))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/%s", encryBase64, savePath, constants.EncryptionConfigName))
	sb.WriteString("\"")

	_, err := r.RunCommand(sb.String())
	return err
}

func generateCertsAndKubeConfigs(r runner.Runner, ccfg *api.ClusterConfig) (err error) {
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

	if err = generateKubeConfigs(rootPath, certPath, cg, ccfg); err != nil {
		return err
	}

	return generateEncryption(r, rootPath)
}

func runKubernetesServices(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	// set up api-server service
	if err := commontools.SetupMasterServices(r, ccfg, hcf); err != nil {
		return err
	}

	return nil
}

func JoinMaterNode(conf *api.ClusterConfig, masterNode *api.HostConfig) error {
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

func Init(conf *api.ClusterConfig) error {
	var firstMaster *api.HostConfig
	for _, node := range conf.Nodes {
		if node.Type&api.Master != 0 {
			firstMaster = node
			break
		}
	}

	// generate ca certificates in eggo
	err := prepareCAs(api.GetCertificateStorePath(conf.Name))
	if err != nil {
		logrus.Errorf("[certs] create ca certificates failed: %v", err)
		return err
	}

	if err := JoinMaterNode(conf, firstMaster); err != nil {
		return err
	}

	post := task.NewTaskInstance(
		&PostControlPlaneTask{
			cluster: conf,
		},
	)
	err = nodemanager.RunTaskOnNodes(post, []string{firstMaster.Address})
	if err != nil {
		return err
	}

	if err := nodemanager.WaitTaskOnNodesFinished(post, []string{firstMaster.Address}, time.Minute*5); err != nil {
		logrus.Errorf("wait to post task for master finish failed: %v", err)
		return err
	}

	return nil
}

type PostControlPlaneTask struct {
	cluster *api.ClusterConfig
}

func (ct *PostControlPlaneTask) Name() string {
	return "PostControlPlaneTask"
}

func (ct *PostControlPlaneTask) doAdminRole(r runner.Runner) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("KUBECONFIG=%s/admin.conf", ct.cluster.GetConfigDir()))
	roleBase64 := base64.StdEncoding.EncodeToString([]byte(AdminRoleConfig))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/admin_cluster_role.yaml", roleBase64, ct.cluster.GetManifestDir()))
	sb.WriteString(fmt.Sprintf(" && kubectl apply -f %s/admin_cluster_role.yaml", ct.cluster.GetManifestDir()))
	sb.WriteString("\"")
	_, err := r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("apply admin role failed: %v", err)
		return err
	}

	sb.Reset()
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("KUBECONFIG=%s/admin.conf", ct.cluster.GetConfigDir()))
	rolebindBase64 := base64.StdEncoding.EncodeToString([]byte(AdminRoleBindConfig))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/admin_cluster_rolebind.yaml", rolebindBase64, ct.cluster.GetManifestDir()))
	sb.WriteString(fmt.Sprintf(" && kubectl apply --kub -f %s/admin_cluster_rolebind.yaml", ct.cluster.GetManifestDir()))
	sb.WriteString("\"")
	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("apply admin rolebind failed: %v", err)
		return err
	}
	return nil
}

func (ct *PostControlPlaneTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	// we should setup some resources for new cluster

	// 1. create admin rolebinding
	if err := ct.doAdminRole(r); err != nil {
		return err
	}

	return nil
}
