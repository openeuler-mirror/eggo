#!/bin/bash
#######################################################################
##- @Copyright (C) Huawei Technologies., Ltd. 2021. All rights reserved.
# - eggo licensed under the Mulan PSL v2.
# - You can use this software according to the terms and conditions of the Mulan PSL v2.
# - You may obtain a copy of Mulan PSL v2 at:
# -     http://license.coscl.org.cn/MulanPSL2
# - THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# - IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# - PURPOSE.
# - See the Mulan PSL v2 for more details.
##- @Description: eggo node machine deploy tool
##- @Author: WangFengTu
##- @Create: 2021-04-10
#######################################################################

source ./helper.sh

function set_kubelet() {
	if [ $# -ne 2 ]; then
		echo "Usage:"
		echo "set_kubelet cluster-ip hostname-override"
		exit 1
	fi

	cat >/etc/kubernetes/kubelet_config.yaml <<EOF
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
clusterDNS:
- $1
clusterDomain: cluster.local
runtimeRequestTimeout: "15m"
EOF

	cat >/usr/lib/systemd/system/kubelet.service <<EOF
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=https://kubernetes.io/docs
Wants=network-online.target
After=network-online.target

[Service]
ExecStart=/usr/bin/kubelet \\
	  --config=/etc/kubernetes/kubelet_config.yaml \\
	  --network-plugin=cni \\
	  --pod-infra-container-image=k8s.gcr.io/pause:3.2 \\
	  --kubeconfig=/etc/kubernetes/kubelet.conf \\
	  --register-node=true \\
	  --hostname-override=$2 \\
	  --cni-bin-dir="/usr/libexec/cni/" \\
	  --container-runtime=remote \\
	  --container-runtime-endpoint=unix:///var/run/isulad.sock \\
	  -v=2

Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

	echo "----enable kubelet----"
	systemctl enable kubelet
}

function set_kube_proxy() {
	if [ $# -ne 2 ]; then
		echo "Usage:"
		echo "set_kube_proxy cluster-cidr hostname-override"
		exit 1
	fi

	cat >/etc/kubernetes/kube-proxy-config.yaml <<EOF
kind: KubeProxyConfiguration
apiVersion: kubeproxy.config.k8s.io/v1alpha1
clientConnection:
  kubeconfig: /etc/kubernetes/kube-proxy.conf
clusterCIDR: $1
mode: "iptables"
EOF

	cat >/usr/lib/systemd/system/kube-proxy.service <<EOF
[Unit]
Description=Kubernetes Kube-Proxy Server
Documentation=https://kubernetes.io/docs/reference/generated/kube-proxy/

[Service]
EnvironmentFile=-/etc/kubernetes/config
EnvironmentFile=-/etc/kubernetes/proxy
ExecStart=/usr/bin/kube-proxy \\
	    \$KUBE_LOGTOSTDERR \\
	    \$KUBE_LOG_LEVEL \\
	    --config=/etc/kubernetes/kube-proxy-config.yaml \\
	    --hostname-override=$2 \\
	    \$KUBE_PROXY_ARGS
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

	echo "----enable kube-proxy----"
	systemctl enable kube-proxy
}

function set_isulad_configs() {
	sed -i "/registry-mirrors/a\    \t\"docker.io\"" /etc/isulad/daemon.json
	sed -i "/insecure-registries/a\    \t\"quay.io\"" /etc/isulad/daemon.json
	sed -i "/insecure-registries/a\    \t\"k8s.gcr.io\"," /etc/isulad/daemon.json
	sed -i "s#pod-sandbox-image\": \"#pod-sandbox-image\": \"k8s.gcr.io/pause:3.2#g" /etc/isulad/daemon.json
	sed -i "s#network-plugin\": \"#network-plugin\": \"cni#g" /etc/isulad/daemon.json
	sed -i "s#cni-bin-dir\": \"#cni-bin-dir\": \"/usr/libexec/cni#g" /etc/isulad/daemon.json
	systemctl restart isulad
}

function do_pre() {
	swapoff -a
	mkdir -p /etc/kubernetes
	set_isulad_configs
}

do_pre

hostname_override=$1
if [ x"$hostname_override" == x"" ]; then
	hostname_override=$(hostname)
fi

firewall-cmd --zone=public --add-port=10250/tcp
echo "-------set_kubelet $NODE_SERVICE_CLUSTER_DNS $hostname_override-------"
set_kubelet "$NODE_SERVICE_CLUSTER_DNS" "$hostname_override"

firewall-cmd --zone=public --add-port=10256/tcp
echo "-------set_kube_proxy $NODE_KUBE_CLUSTER_CIDR $hostname_override------------"
set_kube_proxy "$NODE_KUBE_CLUSTER_CIDR" "$hostname_override"

# add cni configs
./network.sh default

# start services
systemctl start kubelet kube-proxy
