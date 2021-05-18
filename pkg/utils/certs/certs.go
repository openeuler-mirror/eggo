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
	"strings"

	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/template"
	"github.com/sirupsen/logrus"
)

const (
	DefaultCertPath = "/etc/kubernetes"
)

type AltNames struct {
	DNSNames []string
	IPs      []string
}

type CertConfig struct {
	CommonName         string
	Organization       string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

type CertGenerator interface {
	CreateServiceAccount(savePath string) error
	CreateCA(config *CertConfig, savePath string, name string) error
	CreateCertAndKey(caCertPath, caKeyPath string, config *CertConfig, savePath string, name string) error
}

type OpensshBinCertGenerator struct {
	r runner.Runner
}

func NewOpensshBinCertGenerator(r runner.Runner) CertGenerator {
	return &OpensshBinCertGenerator{
		r: r,
	}
}

func (o *OpensshBinCertGenerator) CreateServiceAccount(savePath string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo mkdir -p %s && sudo cd %s ", savePath, savePath))
	sb.WriteString("&& sudo openssl genrsa -out sa.key 4096 ")
	sb.WriteString("&& sudo openssl rsa -in ca.key -pubout -out ca.pub")

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
	if config.Organization != "" {
		sb.WriteString("/O=")
		sb.WriteString(config.Organization)
	}
	return sb.String()
}

func (o *OpensshBinCertGenerator) CreateCA(config *CertConfig, savePath string, name string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo mkdir -p %s && sudo cd %s ", savePath, savePath))
	sb.WriteString(fmt.Sprintf("&& sudo openssl genrsa -out %s.key 4096 ", name))
	sb.WriteString(fmt.Sprintf("&& sudo openssl req -x509 -new -nodes -key %s.key -subj \"%s\" -days 10000 -out %s.crt", name, getSubject(config), name))

	_, err := o.r.RunCommand(sb.String())
	if err != nil {
		return err
	}
	logrus.Debugf("create service account success")
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
	csrconfig := &template.CsrConfig{
		Organization:     config.Organization,
		CommonName:       config.CommonName,
		IPs:              config.AltNames.IPs,
		DNSNames:         config.AltNames.DNSNames,
		ExtendedKeyUsage: extKeyUsage,
	}
	return template.CreateCsrTemplate(name, csrconfig)
}

func (o *OpensshBinCertGenerator) CreateCertAndKey(caCertPath, caKeyPath string, config *CertConfig, savePath string, name string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo mkdir -p %s && sudo cd %s", savePath, savePath))
	csr, err := createCsrString(name, config)
	if err != nil {
		return err
	}
	csrBase64 := base64.StdEncoding.EncodeToString([]byte(csr))
	sb.WriteString(fmt.Sprintf("sudo echo %s | base64 -d > %s/%s-crs.conf", csrBase64, savePath, name))
	_, err = o.r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create %s-csr.conf failed: %v", name, err)
		return err
	}

	sb.WriteString(fmt.Sprintf("sudo cd %s && sudo openssl genrsa -out %s.key 4096 ", savePath, name))
	sb.WriteString(fmt.Sprintf("&& sudo openssl req -new -key %s.key -out %s.csr -config %s/%s-csr.conf", name, name, savePath, name))
	sb.WriteString(fmt.Sprintf("&& sudo openssl x509 -req -in %s.csr -CA %s -CAkey %s -CAcreateserial -out %s.crt -days 10000 -extensions v3_ext -extfile %s-csr.conf", name, caCertPath, caKeyPath, name, name))
	_, err = o.r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create %s-csr.conf failed: %v", name, err)
		return err
	}
	logrus.Debugf("create service account success")
	return nil
}
