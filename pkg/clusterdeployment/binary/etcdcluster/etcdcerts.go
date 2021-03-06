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
 * Author: wangfengtu
 * Create: 2021-05-19
 * Description: eggo etcdcluster generate certificates implement
 ******************************************************************************/

package etcdcluster

import (
	"crypto/x509"
	"fmt"
	"path/filepath"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/certs"
	"isula.org/eggo/pkg/utils/runner"
)

func genEtcdServerCerts(savePath string, hostname string, ip string, cg certs.CertGenerator,
	ccfg *api.ClusterConfig) error {
	return cg.CreateCertAndKey(filepath.Join(savePath, "ca.crt"), filepath.Join(savePath, "ca.key"), &certs.CertConfig{
		CommonName: hostname + "-server",
		AltNames: certs.AltNames{
			IPs:      []string{"127.0.0.1", ip},
			DNSNames: []string{"localhost", hostname},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}, savePath, "server")
}

func genEtcdPeerCerts(savePath string, hostname string, ip string, cg certs.CertGenerator,
	ccfg *api.ClusterConfig) error {
	return cg.CreateCertAndKey(filepath.Join(savePath, "ca.crt"), filepath.Join(savePath, "ca.key"), &certs.CertConfig{
		CommonName: hostname + "-peer",
		AltNames: certs.AltNames{
			IPs:      []string{"127.0.0.1", ip},
			DNSNames: []string{"localhost", hostname},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}, savePath, "peer")
}

func genEtcdHealthcheckClientCerts(savePath string, hostname string, cg certs.CertGenerator,
	ccfg *api.ClusterConfig) error {
	return cg.CreateCertAndKey(filepath.Join(savePath, "ca.crt"), filepath.Join(savePath, "ca.key"), &certs.CertConfig{
		CommonName: hostname + "-healthcheck-client",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, savePath, "healthcheck-client")
}

func genApiserverEtcdClientCerts(savePath string, cg certs.CertGenerator, ccfg *api.ClusterConfig) error {
	return cg.CreateCertAndKey(filepath.Join(savePath, "etcd", "ca.crt"), filepath.Join(savePath, "etcd", "ca.key"),
		&certs.CertConfig{
			CommonName:    "kube-apiserver-etcd-client",
			Organizations: []string{"system:masters"},
			Usages:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}, savePath, "apiserver-etcd-client")
}

// see: https://kubernetes.io/docs/setup/best-practices/certificates/
func generateEtcdCerts(r runner.Runner, ccfg *api.ClusterConfig, hostConfig *api.HostConfig) error {
	etcdCertsPath := filepath.Join(ccfg.GetCertDir(), "etcd")
	cg := certs.NewOpensshBinCertGenerator(r)

	// generate etcd-server certificates
	if err := genEtcdServerCerts(etcdCertsPath, hostConfig.Name, hostConfig.Address, cg, ccfg); err != nil {
		return err
	}

	// generate etcd-peer certificates
	if err := genEtcdPeerCerts(etcdCertsPath, hostConfig.Name, hostConfig.Address, cg, ccfg); err != nil {
		return err
	}

	// generate etcd-healthcheck-client certificates
	if err := genEtcdHealthcheckClientCerts(etcdCertsPath, hostConfig.Name, cg, ccfg); err != nil {
		return err
	}

	return nil
}

// see: https://kubernetes.io/docs/setup/best-practices/certificates/
func generateCaAndApiserverEtcdCerts(ccfg *api.ClusterConfig) error {
	savePath := api.GetCertificateStorePath(ccfg.Name)
	etcdCertsPath := filepath.Join(savePath, "etcd")
	lcg := certs.NewLocalCertGenerator()

	// generate etcd root ca
	caConfig := &certs.CertConfig{
		CommonName: "etcd-ca",
	}

	if ccfg.Certificate.ExternalCA {
		_, err := lcg.RunCommand(fmt.Sprintf("mkdir -p -m 0700 %s && cp -f %s/etcd/%s %s", etcdCertsPath, ccfg.Certificate.ExternalCAPath, certs.GetCertName("ca"), etcdCertsPath))
		if err != nil {
			return err
		}
		_, err = lcg.RunCommand(fmt.Sprintf("cp -f %s/etcd/%s %s", ccfg.Certificate.ExternalCAPath, certs.GetKeyName("ca"), etcdCertsPath))
		if err != nil {
			return err
		}
	}

	if err := lcg.CreateCA(caConfig, etcdCertsPath, "ca"); err != nil {
		return err
	}

	// generate apiserver-etcd-client certificates
	if err := genApiserverEtcdClientCerts(savePath, lcg, ccfg); err != nil {
		return err
	}

	return nil
}
