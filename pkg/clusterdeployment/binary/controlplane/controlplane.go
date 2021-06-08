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
	"crypto/rand"
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
	"gitee.com/openeuler/eggo/pkg/utils/template"
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
	ClusterRoleBindingTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Name }}
subjects:
- kind: {{ .SubjectKind }}
  name: {{ .SubjectName }}
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: {{ .RoleName }}
  apiGroup: rbac.authorization.k8s.io
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
	if err = generateCertsAndKubeConfigs(r, ct.ccfg, hcf); err != nil {
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

	for _, ca := range commontools.MasterRequiredCerts {
		_, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"ls %s\"", filepath.Join(savePath, ca)))
		if err != nil {
			logrus.Errorf("chech ca cert: %s, failed: %v\n", ca, err)
			return err
		}
		logrus.Debugf("chech ca cert: %s success\n", ca)
	}
	return nil
}

func generateApiServerCertificate(savePath string, cg certs.CertGenerator, ccfg *api.ClusterConfig, hcf *api.HostConfig) error {
	ips := []string{"0.0.0.0", "127.0.0.1"}
	dnsnames := []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster", "kubernetes.default.svc.cluster.local"}

	if ccfg.ServiceCluster.Gateway != "" {
		ips = append(ips, ccfg.ServiceCluster.Gateway)
	}
	if ccfg.ControlPlane.ApiConf != nil {
		ips = append(ips, ccfg.ControlPlane.ApiConf.CertSans.IPs...)
		dnsnames = append(dnsnames, ccfg.ControlPlane.ApiConf.CertSans.DNSNames...)
	}
	ips = append(ips, hcf.Address)

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

func generateCerts(savePath string, cg certs.CertGenerator, ccfg *api.ClusterConfig, hcf *api.HostConfig) (err error) {
	// create certificate and keys

	if err = generateApiServerCertificate(savePath, cg, ccfg, hcf); err != nil {
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
		filepath.Join(certPath, "admin.crt"), filepath.Join(certPath, "admin.key"), apiEndpoint)
	if err != nil {
		return
	}

	if err = generateControllerManagerCertificate(certPath, cg); err != nil {
		return
	}
	err = cg.CreateKubeConfig(rootPath, constants.KubeConfigFileNameController, filepath.Join(certPath, "ca.crt"), "default-controller-manager",
		filepath.Join(certPath, "controller-manager.crt"), filepath.Join(certPath, "controller-manager.key"), localEndpoint)
	if err != nil {
		return
	}

	if err = generateSchedulerCertificate(certPath, cg); err != nil {
		return
	}

	return cg.CreateKubeConfig(rootPath, constants.KubeConfigFileNameScheduler, filepath.Join(certPath, "ca.crt"), "default-scheduler",
		filepath.Join(certPath, "scheduler.crt"), filepath.Join(certPath, "scheduler.key"), localEndpoint)
}

func getRandSecret() (string, error) {
	c := 32
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		logrus.Errorf("create rand secret failed: %v", err)
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	return encoded, nil
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
              secret: HOLDER
      - identity: {}
`
	var sb strings.Builder
	randSecret, err := getRandSecret()
	if err != nil {
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"")
	encryBase64 := base64.StdEncoding.EncodeToString([]byte(encry))
	sb.WriteString(fmt.Sprintf("echo \"%s\" | base64 -d > %s/%s", encryBase64, savePath, constants.EncryptionConfigName))
	sb.WriteString(fmt.Sprintf(" && sed -i \"s#HOLDER#%s#g\" %s/%s", randSecret, savePath, constants.EncryptionConfigName))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	return err
}

func generateCertsAndKubeConfigs(r runner.Runner, ccfg *api.ClusterConfig, hcf *api.HostConfig) (err error) {
	rootPath := ccfg.GetConfigDir()
	certPath := ccfg.GetCertDir()

	cg := certs.NewOpensshBinCertGenerator(r)
	defer func() {
		if err != nil {
			// TODO: dot not delete user configed directory, delete directories and files we addded only
			cg.CleanAll(rootPath)
		}
	}()

	// clean generated certifactes
	if err = generateCerts(certPath, cg, ccfg, hcf); err != nil {
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

	if err = JoinMaterNode(conf, firstMaster); err != nil {
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
	roleBase64 := base64.StdEncoding.EncodeToString([]byte(AdminRoleConfig))
	sb.WriteString(fmt.Sprintf(" echo %s | base64 -d > %s/admin_cluster_role.yaml", roleBase64, ct.cluster.GetManifestDir()))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s/admin.conf kubectl apply -f %s/admin_cluster_role.yaml", ct.cluster.GetConfigDir(), ct.cluster.GetManifestDir()))
	sb.WriteString("\"")
	_, err := r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("apply admin role failed: %v", err)
		return err
	}

	adminRoleBindConfig := &api.ClusterRoleBindingConfig{
		Name:        "system:kube-apiserver",
		SubjectName: "kubernetes",
		SubjectKind: "User",
		RoleName:    "system:kube-apiserver-to-kubelet",
	}

	if err := ct.applyClusterRoleBinding(r, adminRoleBindConfig); err != nil {
		logrus.Errorf("apply admin rolebind failed: %v", err)
		return err
	}

	return nil
}

func (ct *PostControlPlaneTask) createBootstrapCrb() []*api.ClusterRoleBindingConfig {
	csr := &api.ClusterRoleBindingConfig{
		Name:        "create-csrs-for-bootstrapping",
		SubjectName: "system:bootstrappers",
		SubjectKind: "Group",
		RoleName:    "system:node-bootstrapper",
	}
	approve := &api.ClusterRoleBindingConfig{
		Name:        "auto-approve-csrs-for-group",
		SubjectName: "system:bootstrappers",
		SubjectKind: "Group",
		RoleName:    "system:certificates.k8s.io:certificatesigningrequests:nodeclient",
	}
	renew := &api.ClusterRoleBindingConfig{
		Name:        "auto-approve-renewals-for-nodes",
		SubjectName: "system:nodes",
		SubjectKind: "Group",
		RoleName:    "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient",
	}

	return []*api.ClusterRoleBindingConfig{csr, approve, renew}
}

func (ct *PostControlPlaneTask) applyClusterRoleBinding(r runner.Runner, crbc *api.ClusterRoleBindingConfig) error {
	datastore := map[string]interface{}{}
	datastore["Name"] = crbc.Name
	datastore["SubjectName"] = crbc.SubjectName
	datastore["SubjectKind"] = crbc.SubjectKind
	datastore["RoleName"] = crbc.RoleName
	crb, err := template.TemplateRender(ClusterRoleBindingTemplate, datastore)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString("mkdir -p /tmp/.eggo")
	crbYamlBase64 := base64.StdEncoding.EncodeToString([]byte(crb))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > /tmp/.eggo/%s.yaml", crbYamlBase64, crbc.Name))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s/admin.conf kubectl apply -f /tmp/.eggo/%s.yaml", ct.cluster.GetConfigDir(), crbc.Name))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("apply crbs failed: %v", err)
		return err
	}
	return nil
}

func (ct *PostControlPlaneTask) bootstrapClusterRoleBinding(r runner.Runner) error {
	crbcs := ct.createBootstrapCrb()
	for _, crbc := range crbcs {
		if err := ct.applyClusterRoleBinding(r, crbc); err != nil {
			logrus.Errorf("apply ClusterRoleBinding failed: %v", err)
			return err
		}
	}

	return nil
}

func (ct *PostControlPlaneTask) waitClusterReady(r runner.Runner) error {
	check := `
#!/bin/bash
for i in $(seq 60); do
	KUBECONFIG={{ .KubeHomeDir }}/admin.conf kubectl get nodes
	if [ $? -eq 0 ]; then
		exit 0
	fi
	sleep 1
done
exit 1
`
	datastore := map[string]interface{}{}
	datastore["KubeHomeDir"] = ct.cluster.GetConfigDir()
	shell, err := template.TemplateRender(check, datastore)
	if err != nil {
		return err
	}
	output, err := r.RunShell(shell, "waitcluster")
	if err != nil {
		logrus.Errorf("wait cluster failed: %v", err)
		return err
	}
	logrus.Debugf("wait cluster success: %s", output)

	return nil
}

func (ct *PostControlPlaneTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	// we should setup some resources for new cluster
	// 0. wait cluster ready
	if err := ct.waitClusterReady(r); err != nil {
		return err
	}

	// 1. create admin rolebinding
	if err := ct.doAdminRole(r); err != nil {
		return err
	}

	// 2. create bootstrap clusterrolebinding
	if err := ct.bootstrapClusterRoleBinding(r); err != nil {
		return err
	}

	return nil
}
