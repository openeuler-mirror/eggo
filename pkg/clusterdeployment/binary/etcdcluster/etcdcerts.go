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
	"path/filepath"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/certs"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
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
	}, savePath, hostname+"-server")
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
	}, savePath, hostname+"-peer")
}

func genEtcdHealthcheckClientCerts(savePath string, hostname string, cg certs.CertGenerator,
	ccfg *api.ClusterConfig) error {
	return cg.CreateCertAndKey(filepath.Join(savePath, "ca.crt"), filepath.Join(savePath, "ca.key"), &certs.CertConfig{
		CommonName: hostname + "-healthcheck-client",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, savePath, hostname+"-healthcheck-client")
}

func genApiserverEtcdClientCerts(savePath string, hostnameList []string, ipList []string, cg certs.CertGenerator,
	ccfg *api.ClusterConfig) error {
	return cg.CreateCertAndKey(filepath.Join(savePath, "etcd", "ca.crt"), filepath.Join(savePath, "etcd", "ca.key"),
		&certs.CertConfig{
			CommonName:    "kube-apiserver-etcd-client",
			Organizations: []string{"system:masters"},
			AltNames: certs.AltNames{
				IPs:      append(ipList, "127.0.0.1"),
				DNSNames: append(hostnameList, "localhost"),
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}, savePath, "kube-apiserver-etcd-client")
}

// see: https://kubernetes.io/docs/setup/best-practices/certificates/
func generateCerts(r runner.Runner, ccfg *api.ClusterConfig) error {
	savePath := api.GetCertificateStorePath(ccfg.Name)
	etcdCertsPath := filepath.Join(savePath, "etcd")
	cg := certs.NewOpensshBinCertGenerator(r)

	// generate etcd root ca
	caConfig := &certs.CertConfig{
		CommonName: "etcd-ca",
	}
	if err := cg.CreateCA(caConfig, etcdCertsPath, "ca"); err != nil {
		return err
	}

	for _, node := range ccfg.EtcdCluster.Nodes {
		// generate etcd-server certificates
		if err := genEtcdServerCerts(etcdCertsPath, node.Name, node.Address, cg, ccfg); err != nil {
			return err
		}

		// generate etcd-peer certificates
		if err := genEtcdPeerCerts(etcdCertsPath, node.Name, node.Address, cg, ccfg); err != nil {
			return err
		}

		// generate etcd-healthcheck-client certificates
		if err := genEtcdHealthcheckClientCerts(etcdCertsPath, node.Name, cg, ccfg); err != nil {
			return err
		}
	}

	var hostnameList []string
	var ipList []string
	for _, node := range ccfg.Nodes {
		hostnameList = append(hostnameList, node.Name)
		ipList = append(ipList, node.Address)
	}

	// generate apiserver-etcd-client certificates
	if err := genApiserverEtcdClientCerts(savePath, hostnameList, ipList, cg, ccfg); err != nil {
		return err
	}

	return nil
}
