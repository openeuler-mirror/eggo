package commontools

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/template"
	"github.com/sirupsen/logrus"
)

const (
	SystemdServiceConfigPath = "/usr/lib/systemd/system"
)

func SetupAPIServerService(r runner.Runner, ccfg *clusterdeployment.ClusterConfig, hcf *clusterdeployment.HostConfig) error {
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
		"--etcd-servers":                       ccfg.ServiceCluster.CIDR,
		"--client-ca-file":                     "/etc/kubernetes/pki/ca.crt",
		"--kubelet-client-certificate":         "/etc/kubernetes/pki/apiserver-kubelet-client.crt",
		"--kubelet-client-key":                 "/etc/kubernetes/pki/apiserver-kubelet-client.key",
		"--kubelet-https":                      "true",
		"--proxy-client-cert-file":             "/etc/kubernetes/pki/front-proxy-client.crt",
		"--proxy-client-key-file":              "/etc/kubernetes/pki/front-proxy-client.key",
		"--tls-cert-file":                      "/etc/kubernetes/pki/apiserver.crt",
		"--tls-private-key-file":               "/etc/kubernetes/pki/apiserver.key",
		"--service-cluster-ip-range":           "10.32.0.0/16",
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

func SetupControllerManagerService(r runner.Runner, ccfg *clusterdeployment.ClusterConfig, hcf *clusterdeployment.HostConfig) error {
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

func SetupSchedulerService(r runner.Runner, ccfg *clusterdeployment.ClusterConfig) error {
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

func SetupControlplaneServices(r runner.Runner, ccfg *clusterdeployment.ClusterConfig, hcf *clusterdeployment.HostConfig) error {
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
		logrus.Errorf("start k8s services failed: %v", err)
	}
	logrus.Info("setup controlplane services success")
	return nil
}
