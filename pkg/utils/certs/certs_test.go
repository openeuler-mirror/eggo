package certs

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
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

	if err := cg.CleanAll(savePath); err != nil {
		t.Fatalf("clean all failed: %v", err)
	}
}
