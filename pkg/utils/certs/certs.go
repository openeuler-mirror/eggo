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
 * Create: 2021-05-19
 * Description: eggo certificate utils
 ******************************************************************************/
package certs

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/template"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	certutil "k8s.io/client-go/util/cert"
	keyutil "k8s.io/client-go/util/keyutil"
)

const (
	ServiceAccountKeyBaseName    = "sa"
	ServiceAccountPrivateKeyName = "sa.key"
	ServiceAccountPublicKeyName  = "sa.pub"
)

type AltNames struct {
	DNSNames []string
	IPs      []string
}

type CertConfig struct {
	CommonName         string
	Organizations      []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

type CertGenerator interface {
	RunCommand(cmd string) (string, error)
	CreateServiceAccount(savePath string) error
	CreateCA(config *CertConfig, savePath string, name string) error
	CreateCertAndKey(caCertPath, caKeyPath string, config *CertConfig, savePath string, name string) error
	CreateKubeConfig(savePath, filename string, caCertPath, credName, certPath, keyPath string, enpoint string) error
	CleanAll(savePath string) error
}

type LocalCertGenerator struct {
	keyAlgorithm x509.PublicKeyAlgorithm
	lr           runner.Runner
}

func NewLocalCertGenerator() CertGenerator {
	// TODO: only support RSA now
	return &LocalCertGenerator{
		keyAlgorithm: x509.RSA,
		lr:           &runner.LocalRunner{},
	}
}

func (l *LocalCertGenerator) RunCommand(cmd string) (string, error) {
	return l.lr.RunCommand(cmd)
}

func (l *LocalCertGenerator) CreateServiceAccount(savePath string) error {
	_, err := keyutil.PrivateKeyFromFile(filepath.Join(savePath, ServiceAccountPrivateKeyName))
	if err == nil {
		logrus.Info("service account certs exist\n")
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "service account exist, but not valid")
	}

	signer, err := GetKeySigner(l.keyAlgorithm)
	if err != nil {
		logrus.Errorf("new private key for service account failed: %v", err)
		return err
	}
	err = WriteKey(signer, filepath.Join(savePath, "sa.key"))
	if err != nil {
		logrus.Errorf("write private key for service account failed: %v", err)
		return err
	}
	err = WritePublicKey(signer.Public(), filepath.Join(savePath, "sa.pub"))
	if err != nil {
		logrus.Errorf("write public key for service account failed: %v", err)
	}
	return err
}

func (l *LocalCertGenerator) CreateCA(config *CertConfig, savePath string, name string) error {
	if _, err := certutil.CertsFromFile(filepath.Join(savePath, GetCertName(name))); err == nil {
		if _, err := keyutil.PrivateKeyFromFile(filepath.Join(savePath, GetKeyName(name))); err == nil {
			logrus.Infof("[certs] using exist %s ca", GetCertName(name))
			return nil
		}
		logrus.Infof("[certs] using exist %s keyless ca", name)
		return nil
	}
	ips, err := ParseIPsFromString(config.AltNames.IPs)
	if err != nil {
		logrus.Errorf("parse altnames failed: %v", err)
		return err
	}

	signer, err := GetKeySigner(config.PublicKeyAlgorithm)
	if err != nil {
		logrus.Errorf("invalid public key algorithm: %v", err)
		return err
	}

	cc := certutil.Config{
		CommonName:   config.CommonName,
		Organization: config.Organizations,
		Usages:       config.Usages,
		AltNames: certutil.AltNames{
			DNSNames: config.AltNames.DNSNames,
			IPs:      ips,
		},
	}
	cert, err := certutil.NewSelfSignedCACert(cc, signer)
	if err != nil {
		logrus.Errorf("create self signed ca cert failed: %v", err)
		return err
	}

	if err := WriteKey(signer, filepath.Join(savePath, GetKeyName(name))); err != nil {
		logrus.Errorf("write key: %s failed: %v", GetKeyName(name), err)
		return err
	}

	if err := WriteCert(cert, filepath.Join(savePath, GetCertName(name))); err != nil {
		logrus.Errorf("write cert: %s failed: %v", GetCertName(name), err)
		return err
	}

	return nil
}

func (l *LocalCertGenerator) CreateCertAndKey(caCertPath, caKeyPath string, config *CertConfig, savePath string, name string) error {
	logrus.Info("TODO: do not need CreateCertAndKey for LocalCertGenerator now.")
	return nil
}

func (l *LocalCertGenerator) CreateKubeConfig(savePath, filename string, caCertPath, credName, certPath, keyPath string, enpoint string) error {
	logrus.Info("TODO: do not need CreateKubeConfig for LocalCertGenerator now.")
	return nil
}

func (l *LocalCertGenerator) CleanAll(savePath string) error {
	return os.RemoveAll(savePath)
}

type OpensshBinCertGenerator struct {
	r runner.Runner
}

func NewOpensshBinCertGenerator(r runner.Runner) CertGenerator {
	return &OpensshBinCertGenerator{
		r: r,
	}
}

func (g *OpensshBinCertGenerator) RunCommand(cmd string) (string, error) {
	return g.r.RunCommand(cmd)
}

func (o *OpensshBinCertGenerator) CleanAll(savePath string) error {
	if filepath.IsAbs(savePath) {
		return fmt.Errorf("%s is not absolute path", savePath)
	}
	savePath = filepath.Clean(savePath)
	if savePath == "/" {
		return fmt.Errorf("rm -rf %s is risk", savePath)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo rm -rf %s", savePath))

	_, err := o.r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	logrus.Debugf("clean all success")
	return nil
}

func (o *OpensshBinCertGenerator) CreateServiceAccount(savePath string) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s && cd %s", savePath, savePath))
	sb.WriteString(" && openssl genrsa -out sa.key 4096")
	sb.WriteString(" && openssl rsa -in ca.key -pubout -out ca.pub")
	sb.WriteString("\"")

	_, err := o.r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	logrus.Debugf("create service account success")
	return nil
}

func getSubject(config *CertConfig) string {
	var sb strings.Builder
	if config.CommonName != "" {
		sb.WriteString("/CN=")
		sb.WriteString(config.CommonName)
	}
	if len(config.Organizations) > 0 {
		sb.WriteString("/O=")
		// TODO: support multi organizations
		sb.WriteString(config.Organizations[0])
	}
	return sb.String()
}

func (o *OpensshBinCertGenerator) CreateCA(config *CertConfig, savePath string, name string) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s && cd %s", savePath, savePath))
	sb.WriteString(fmt.Sprintf(" && openssl genrsa -out %s.key 4096", name))
	sb.WriteString(fmt.Sprintf(" && openssl req -x509 -new -nodes -key %s.key -subj \"%s\" -days 10000 -out %s.crt", name, getSubject(config), name))
	sb.WriteString("\"")

	_, err := o.r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	logrus.Debugf("create root ca success")
	return nil
}

func createCsrString(name string, config *CertConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("empty cert config")
	}
	var extKeyUsage string
	for _, us := range config.Usages {
		if us == x509.ExtKeyUsageServerAuth {
			extKeyUsage += ",serverAuth"
		} else if us == x509.ExtKeyUsageClientAuth {
			extKeyUsage += ",clientAuth"
		}
	}
	if len(extKeyUsage) > 0 {
		extKeyUsage = extKeyUsage[1:]
	}
	var org string
	if len(config.Organizations) > 0 {
		org = config.Organizations[0]
	}
	csrconfig := &template.CsrConfig{
		Organization:     org,
		CommonName:       config.CommonName,
		IPs:              config.AltNames.IPs,
		DNSNames:         config.AltNames.DNSNames,
		ExtendedKeyUsage: extKeyUsage,
	}
	return template.CreateCsrTemplate(name, csrconfig)
}

func (o *OpensshBinCertGenerator) CreateCertAndKey(caCertPath, caKeyPath string, config *CertConfig, savePath string, name string) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s && cd %s", savePath, savePath))
	csr, err := createCsrString(name, config)
	if err != nil {
		return err
	}
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(csr))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/%s-csr.conf", csrBase64, savePath, name))
	sb.WriteString("\"")
	_, err = o.r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create %s-csr.conf failed: %v", name, err)
		return err
	}

	sb.Reset()
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("cd %s && openssl genrsa -out %s.key 4096", savePath, name))
	sb.WriteString(fmt.Sprintf(" && openssl req -new -key %s.key -out %s.csr -config %s/%s-csr.conf", name, name, savePath, name))
	sb.WriteString(fmt.Sprintf(" && openssl x509 -req -in %s.csr -CA %s -CAkey %s -CAcreateserial -out %s.crt -days 10000 -extensions v3_ext -extfile %s-csr.conf", name, caCertPath, caKeyPath, name, name))
	sb.WriteString(fmt.Sprintf(" && rm %s/%s-csr.conf", savePath, name))
	sb.WriteString("\"")
	_, err = o.r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create certs and keys: '%s' failed: %v", name, err)
		return err
	}
	logrus.Debugf("create certs and keys: '%s' success", name)
	return nil
}

func (o *OpensshBinCertGenerator) CreateKubeConfig(savePath, filename string, caCertPath, credName, certPath, keyPath string, enpoint string) error {
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("cd %s", savePath))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl config set-cluster default-cluster --server=%s --certificate-authority %s --embed-certs", filename, enpoint, caCertPath))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl config set-credentials %s --client-key %s --client-certificate %s --embed-certs", filename, credName, keyPath, certPath))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl config set-context default-system --cluster default-cluster --user %s", filename, credName))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl config use-context default-system", filename))
	sb.WriteString("\"")
	_, err := o.r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create kubeconfig: '%s' failed: %v", filename, err)
		return err
	}
	logrus.Debugf("create kubeconfig: '%s' success", filename)
	return nil
}
