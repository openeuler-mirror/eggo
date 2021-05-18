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
 * Description: testcases for template utils
 ******************************************************************************/
package template

import (
	"testing"
)

func TestCreateCsrTemplate(t *testing.T) {
	apiserver_expect := `[ req ]
default_bits = 4096
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
O = kubernetes
CN = kube-apiserver

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = kubernetes
DNS.2 = kubernetes.default
DNS.3 = kubernetes.default.svc
DNS.4 = kubernetes.default.svc.cluster
DNS.5 = kubernetes.default.svc.cluster.local
IP.1 = 0.0.0.0
IP.2 = 10.32.0.1
IP.3 = 127.0.0.1

[ v3_ext ]
authorityKeyIdentifier = keyid,issuer:always
basicConstraints = CA:FALSE
keyUsage = keyEncipherment,dataEncipherment
extendedKeyUsage = serverAuth,clientAuth
subjectAltName = @alt_names
`
	apiserver_conf := &CsrConfig{
		Organization:     "kubernetes",
		CommonName:       "kube-apiserver",
		IPs:              []string{"0.0.0.0", "10.32.0.1", "127.0.0.1"},
		DNSNames:         []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster", "kubernetes.default.svc.cluster.local"},
		ExtendedKeyUsage: "serverAuth,clientAuth",
	}

	kubelet_expect := `[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
O = system:masters
CN = kube-apiserver-kubelet-client

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
`
	kubelet_conf := &CsrConfig{
		Organization:     "system:masters",
		CommonName:       "kube-apiserver-kubelet-client",
		ExtendedKeyUsage: "clientAuth",
	}

	front_proxy_client_expect := `[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
CN = front-proxy-client

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
`
	front_proxy_client_conf := &CsrConfig{
		CommonName:       "front-proxy-client",
		ExtendedKeyUsage: "clientAuth",
	}

	cases := []struct {
		Name   string
		Conf   *CsrConfig
		Expect string
	}{
		{
			Name:   "apiserver",
			Conf:   apiserver_conf,
			Expect: apiserver_expect,
		},
		{
			Name:   "kubelet",
			Conf:   kubelet_conf,
			Expect: kubelet_expect,
		},
		{
			Name:   "front-proxy-client",
			Conf:   front_proxy_client_conf,
			Expect: front_proxy_client_expect,
		},
	}

	for _, c := range cases {
		str, err := CreateCsrTemplate(c.Name, c.Conf)
		if err != nil {
			t.Fatalf("create %s csr config failed: %v", c.Name, err)
		}
		if str == c.Expect {
			t.Logf("create %s csr config success", c.Name)
			return
		}

		t.Fatalf("create %s csr config failed, get: \n%s", c.Name, str)
	}
}
