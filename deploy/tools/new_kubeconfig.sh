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
##- @Description: eggo kubeconfig generator tool
##- @Author: haozi007
##- @Create: 2021-04-10
#######################################################################

source ./helper.sh

which kubectl
if [ $? -ne 0 ]; then
	echo "require kubectl"
	exit 1
fi

api_expose_ip="$API_SERVER_EXPOSE_IP"
api_expose_port="$API_SERVER_EXPOSE_PORT"

function create_admin_conf() {
	pushd $result_dir

	cat >$tmp_dir/admin-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
O = system:masters
OU = "openEuler k8s admin"
CN = kubernetes-admin

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca admin $tmp_dir/admin-csr.conf ./pki/ca.crt ./pki/ca.key

	# create kube config
	echo "---- new_kube_config admin.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-admin admin.key admin.crt ----"
	new_kube_config admin.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-admin admin.key admin.crt

	rm -f admin.key admin.crt admin.csr
	popd
}

function create_controller_manager_conf() {
	pushd $result_dir

	cat >$tmp_dir/controller-manager-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
OU = "openEuler k8s kube controller manager"
CN = system:kube-controller-manager

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca controller-manager $tmp_dir/controller-manager-csr.conf ./pki/ca.crt ./pki/ca.key

	# create kube config
	echo "---- new_kube_config controller-manager.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-controller-manager controller-manager.key controller-manager.crt ----"
	new_kube_config controller-manager.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-controller-manager controller-manager.key controller-manager.crt
	rm -f controller-manager.key controller-manager.crt controller-manager.csr
	popd
}

function create_scheduler_conf() {
	pushd $result_dir

	cat >$tmp_dir/scheduler-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
OU = "openEuler k8s kube scheduler"
CN = system:kube-scheduler

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca scheduler $tmp_dir/scheduler-csr.conf ./pki/ca.crt ./pki/ca.key

	# create kube config
	echo "---- new_kube_config scheduler.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-scheduler scheduler.key scheduler.crt ----"
	new_kube_config scheduler.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-scheduler scheduler.key scheduler.crt
	rm -f scheduler.key scheduler.crt scheduler.csr
	popd
}

function create_kube_proxy_conf() {
	pushd $result_dir

	cat >$tmp_dir/kube-proxy-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
OU = "openEuler k8s kube proxy"
CN = system:kube-proxy

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca kube-proxy $tmp_dir/kube-proxy-csr.conf ./pki/ca.crt ./pki/ca.key

	# create kube-proxy config
	echo "---- new_kube_config kube-proxy.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-scheduler kube-proxy.key kube-proxy.crt ----"
	new_kube_config kube-proxy.conf ./pki/ca.crt $api_expose_ip $api_expose_port default-kube-proxy kube-proxy.key kube-proxy.crt
	rm -f scheduler.key scheduler.crt scheduler.csr
	popd
}

function create_kubelet_conf() {
	if [ $# -ne 1 ]; then
		echo "need set node name which kubelet in"
		exit 1
	fi
	pushd $result_dir

	cat >$tmp_dir/kubelet-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
O = system:nodes
OU = "openEuler k8s node $1"
CN = "system:node:$1"

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca kubelet $tmp_dir/kubelet-csr.conf ./pki/ca.crt ./pki/ca.key

	# create kube config
	echo "---- new_kube_config "$1-kubelet.conf" ./pki/ca.crt $api_expose_ip $api_expose_port default-auth kubelet.key kubelet.crt ----"
	new_kube_config "$1-kubelet.conf" ./pki/ca.crt $api_expose_ip $api_expose_port default-auth kubelet.key kubelet.crt
	rm -rf kubelet.key kubelet.crt kubelet.csr
	popd
}

create_admin_conf

create_controller_manager_conf

create_scheduler_conf

create_kube_proxy_conf

# maybe do not need set hosts
function gen_nodes_kubelet() {
	for i in "${!NODE_NAMES[@]}"; do
		echo "generate: ${NODE_NAMES[$i]} ${NODE_IPS[$i]}"
		create_kubelet_conf "${NODE_NAMES[$i]}"
	done
}
gen_nodes_kubelet
