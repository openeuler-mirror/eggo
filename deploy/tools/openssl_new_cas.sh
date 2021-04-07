#!/bin/bash

source ./helper.sh

function init() {
	mkdir -p $etcd_cas_dir
	mkdir -p $tmp_dir
}

function gen_service_account() {
	pushd $cas_dir

	openssl_gen_pub_and_key sa

	popd
}

function gen_root_ca() {
	pushd $cas_dir

	openssl_gen_ca ca kubernetes

	popd
}

function gen_api_server() {
	pushd $cas_dir

	cat >$tmp_dir/apiserver-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
O = kubernetes
OU = "openEuler k8s kube api server"
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
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF


    local idx=4
    local dns_idx=6
    # insert new ip line after IP.3
    local insert_line=27
    for i in "${!MASTER_IPS[@]}"; do
        sed -i "$insert_line a\\IP.$idx = ${MASTER_IPS[$i]}" $tmp_dir/apiserver-csr.conf
        idx=$(($idx+1))
        insert_line=$(($insert_line+1))
    done
    for i in "${!EXTRA_SANS[@]}"; do
        ipcalc -cs ${EXTRA_SANS[$i]}
        if [ $? -eq 0 ]; then
            sed -i "$insert_line a\\IP.$idx = ${EXTRA_SANS[$i]}" $tmp_dir/apiserver-csr.conf
            idx=$(($idx+1))
        else
            sed -i "$insert_line a\\DNS.$dns_idx = ${EXTRA_SANS[$i]}" $tmp_dir/apiserver-csr.conf
            dns_idx=$(($dns_idx+1))
        fi
        insert_line=$(($insert_line+1))
    done

	openssl_gen_cert_and_key_with_ca apiserver $tmp_dir/apiserver-csr.conf ca.crt ca.key
	rm -f apiserver.csr

	popd
}

function gen_kubelet() {
	pushd $cas_dir

	cat >$tmp_dir/kubelet-client-csr.conf <<EOF
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
OU = "openEuler k8s kubelet client"
CN = kube-apiserver-kubelet-client

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca apiserver-kubelet-client $tmp_dir/kubelet-client-csr.conf ca.crt ca.key
	rm -f apiserver-kubelet-client.csr

	popd
}

function gen_front_proxy_ca() {
	pushd $cas_dir

	openssl_gen_ca front-proxy-ca front-proxy-ca

	popd
}

function gen_front_proxy_client() {
	pushd $cas_dir

	cat >$tmp_dir/front-proxy-client-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
OU = "openEuler k8s front proxy client"
CN = front-proxy-client

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca front-proxy-client $tmp_dir/front-proxy-client-csr.conf ca.crt ca.key

	rm -f front-proxy-client.csr
	popd
}

function gen_etcd_ca() {
	pushd $etcd_cas_dir

	openssl_gen_ca ca etcd-ca

	popd
}

function gen_etcd_server() {
	pushd $etcd_cas_dir

	cat >$tmp_dir/etcd-server-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
OU = "openEuler etcd server"
CN = $1

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = $1
DNS.2 = localhost
IP.1 = $2
IP.2 = 127.0.0.1
IP.3 = ::1

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF

	openssl_gen_cert_and_key_with_ca $1-server $tmp_dir/etcd-server-csr.conf ca.crt ca.key
	rm -f $1-server.csr

	popd
}

function gen_etcd_peer() {
	pushd $etcd_cas_dir

	cat >$tmp_dir/etcd-peer-csr.conf <<EOF
[ req ]
default_bits = 4096
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
C = CN
ST = BinJiang
L = HangZhou
OU = "openEuler etcd peer"
CN = $1

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = $1
DNS.2 = localhost
IP.1 = $2
IP.2 = 127.0.0.1
IP.3 = ::1

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=@alt_names
EOF

	openssl_gen_cert_and_key_with_ca $1-peer $tmp_dir/etcd-peer-csr.conf ca.crt ca.key
	rm -f $1-peer.csr

	popd
}

function gen_etcd_healthcheck() {
	pushd $etcd_cas_dir

	cat >$tmp_dir/etcd-health-check-csr.conf <<EOF
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
OU = "openEuler etcd health check client"
CN = kube-etcd-healthcheck-client

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca healthcheck-client $tmp_dir/etcd-health-check-csr.conf ca.crt ca.key
	rm -f healthcheck-client.csr

	popd
}

function gen_etcd_apiserver_client() {
	pushd $etcd_cas_dir

	cat >$tmp_dir/etcd-apiserver-client-csr.conf <<EOF
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
OU = "openEuler etcd health check client"
CN = kube-apiserver-etcd-client

[ v3_ext ]
authorityKeyIdentifier=keyid,issuer:always
basicConstraints=CA:FALSE
keyUsage=keyEncipherment,dataEncipherment
extendedKeyUsage=clientAuth
EOF

	openssl_gen_cert_and_key_with_ca apiserver-etcd-client $tmp_dir/etcd-apiserver-client-csr.conf ca.crt ca.key
	rm -f apiserver-etcd-client.csr

	# move apiser etcd certs to cas dir
	mv apiserver-etcd-client.* $cas_dir
	popd
}

function post_new_cas() {
	echo "----move certs into /etc/kubernetes----"
	rm -f $cas_dir/ca.srl
	rm -f $etcd_cas_dir/ca.srl
	local real_ca_path=$(realpath $CA_ROOT_PATH)
	if [ x"$real_ca_path" == x"/etc" ]; then
		return 0
	fi
	echo "----WARN: remove old /etc/kubernetes dir ----"
	rm -rf /etc/kubernetes
	cp -r $result_dir /etc
}

init

echo "-----generator service account ca-----"
gen_service_account

echo "-----generator root ca-----"
gen_root_ca

echo "-----generator api server cer-----"
gen_api_server

echo "-----generator kubelet cer-----"
gen_kubelet

echo "-----generator kube front proxy ca-----"
gen_front_proxy_ca

echo "-----generator kube front proxy client cer-----"
gen_front_proxy_client

echo "-----generator etcd ca-----"
gen_etcd_ca

for i in "${!MASTER_NAMES[@]}"; do
	echo "-----generator etcd server ${MASTER_NAMES[$i]} ${MASTER_IPS[$i]} -----"
	gen_etcd_server "${MASTER_NAMES[$i]}" "${MASTER_IPS[$i]}"

	echo "-----generator etcd peer ${MASTER_NAMES[$i]} ${MASTER_IPS[$i]} -----"
	gen_etcd_peer "${MASTER_NAMES[$i]}" "${MASTER_IPS[$i]}"
done

echo "-----generator etcd healthcheck-----"
gen_etcd_healthcheck

echo "-----generator etcd apiserver client-----"
gen_etcd_apiserver_client

echo "-----generator encryption provider config-----"
generate_encryption

post_new_cas
