package certs

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gitee.com/openeuler/eggo/pkg/constants"
)

func TestNewLocalCertGenerator(t *testing.T) {
	savePath := "/tmp/haozi"
	cg := NewLocalCertGenerator()
	err := cg.CreateServiceAccount(savePath)
	if err != nil {
		t.Fatalf("create service account failed: %v", err)
	}
	f, err := os.Open(filepath.Join(savePath, "sa.pub"))
	if err != nil {
		t.Fatalf("read sa.pub file failed: %v", err)
	}
	f.Close()
	f, err = os.Open(filepath.Join(savePath, "sa.key"))
	if err != nil {
		t.Fatalf("read sa.key file failed: %v", err)
	}
	f.Close()

	apiserverConfig := &CertConfig{
		CommonName:    "kube-apiserver",
		Organizations: []string{"kubernetes"},
		AltNames: AltNames{
			IPs:      []string{"127.0.0.1"},
			DNSNames: []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster"},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	err = cg.CreateCA(apiserverConfig, savePath, "ca")
	if err != nil {
		t.Fatalf("create apiserver ca failed: %v", err)
	}
	f, err = os.Open(filepath.Join(savePath, "ca.crt"))
	if err != nil {
		t.Fatalf("read ca.crt file failed: %v", err)
	}
	f.Close()
	f, err = os.Open(filepath.Join(savePath, "ca.key"))
	if err != nil {
		t.Fatalf("read ca.key file failed: %v", err)
	}
	f.Close()

	adminConfig := &CertConfig{
		CommonName:    "kubernetes-admin",
		Organizations: []string{"system:masters"},
		Usages:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	caCertPath := fmt.Sprintf("%s/ca.crt", savePath)
	caKeyPath := fmt.Sprintf("%s/ca.key", savePath)
	err = cg.CreateCertAndKey(caCertPath, caKeyPath, adminConfig, savePath, "admin")
	if err != nil {
		t.Fatalf("create cert and key for admin failed: %v", err)
	}
	err = cg.CreateKubeConfig(savePath, constants.KubeConfigFileNameAdmin, caCertPath, "default-admin",
		filepath.Join(savePath, "admin.crt"), filepath.Join(savePath, "admin.key"), "https://127.0.0.1:6443")
	if err != nil {
		t.Fatalf("create kubeconfig for admin failed: %v", err)
	}
	if err := cg.CleanAll(savePath); err != nil {
		t.Fatalf("clean all failed: %v", err)
	}
}
