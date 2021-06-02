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

func TestCreateSystemdServiceTemplate(t *testing.T) {
	apiConf := &SystemdServiceConfig{
		Description:   "Kubernetes API Server",
		Documentation: "https://kubernetes.io/docs/reference/generated/kube-apiserver/",
		Afters:        []string{"network.target", "etcd.service"},
		Command:       "/usr/bin/kube-apiserver",
		Arguments: []string{
			"--advertise-address=192.168.1.1",
			"--allow-privileged=true",
			"--authorization-mode=Node,RBAC",
			"--enable-admission-plugins=NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota",
			"--secure-port=6443",
			"--enable-bootstrap-token-auth=true",
			"--etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt",
			"--etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt",
			"--etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key",
			"--etcd-servers=192.168.1.1:2379",
			"--client-ca-file=/etc/kubernetes/pki/ca.crt",
			"--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt",
			"--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key",
			"--kubelet-https=true",
			"--proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt",
			"--proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key",
			"--tls-cert-file=/etc/kubernetes/pki/apiserver.crt",
			"--tls-private-key-file=/etc/kubernetes/pki/apiserver.key",
			"--service-cluster-ip-range=10.32.0.0/16",
			"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
			"--service-account-key-file=/etc/kubernetes/pki/sa.pub",
			"--service-account-signing-key-file=/etc/kubernetes/pki/sa.key",
			"--service-node-port-range=30000-32767",
			"--requestheader-allowed-names=front-proxy-client",
			"--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt",
			"--requestheader-extra-headers-prefix=X-Remote-Extra-",
			"--requestheader-group-headers=X-Remote-Group",
			"--requestheader-username-headers=X-Remote-User",
			"--encryption-provider-config=/etc/kubernetes/encryption-config.yaml",
		},
	}
	apiExpect := `[Unit]
Description=Kubernetes API Server
Documentation=https://kubernetes.io/docs/reference/generated/kube-apiserver/
After=network.target
After=etcd.service

[Service]
ExecStart=/usr/bin/kube-apiserver \
		--advertise-address=192.168.1.1 \
		--allow-privileged=true \
		--authorization-mode=Node,RBAC \
		--enable-admission-plugins=NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota \
		--secure-port=6443 \
		--enable-bootstrap-token-auth=true \
		--etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt \
		--etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt \
		--etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key \
		--etcd-servers=192.168.1.1:2379 \
		--client-ca-file=/etc/kubernetes/pki/ca.crt \
		--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt \
		--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key \
		--kubelet-https=true \
		--proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt \
		--proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key \
		--tls-cert-file=/etc/kubernetes/pki/apiserver.crt \
		--tls-private-key-file=/etc/kubernetes/pki/apiserver.key \
		--service-cluster-ip-range=10.32.0.0/16 \
		--service-account-issuer=https://kubernetes.default.svc.cluster.local \
		--service-account-key-file=/etc/kubernetes/pki/sa.pub \
		--service-account-signing-key-file=/etc/kubernetes/pki/sa.key \
		--service-node-port-range=30000-32767 \
		--requestheader-allowed-names=front-proxy-client \
		--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt \
		--requestheader-extra-headers-prefix=X-Remote-Extra- \
		--requestheader-group-headers=X-Remote-Group \
		--requestheader-username-headers=X-Remote-User \
		--encryption-provider-config=/etc/kubernetes/encryption-config.yaml

Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`

	testConf := &SystemdServiceConfig{
		Description:   "Kubernetes API Server",
		Documentation: "https://kubernetes.io/docs/reference/generated/kube-apiserver/",
		Afters:        []string{"network.target", "etcd.service"},
		Command:       "/usr/bin/kube-apiserver",
	}
	testExpect := `[Unit]
Description=Kubernetes API Server
Documentation=https://kubernetes.io/docs/reference/generated/kube-apiserver/
After=network.target
After=etcd.service

[Service]
ExecStart=/usr/bin/kube-apiserver

Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`

	cases := []struct {
		Name   string
		Conf   *SystemdServiceConfig
		Expect string
	}{
		{
			Name:   "apiserver",
			Conf:   apiConf,
			Expect: apiExpect,
		},
		{
			Name:   "test",
			Conf:   testConf,
			Expect: testExpect,
		},
	}

	for _, c := range cases {
		str, err := CreateSystemdServiceTemplate(c.Name, c.Conf)
		if err != nil {
			t.Fatalf("create %s failed: %v", c.Name, err)
		}
		if str == c.Expect {
			t.Logf("create %s success", c.Name)
			return
		}

		t.Fatalf("create %s failed, not expect service, get: \n%s", c.Name, str)
	}

}
