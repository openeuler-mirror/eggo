#!/bin/bash

source ./configs

current_dir="$(cd "$(dirname "$0")" && pwd)"
result_dir=$CA_ROOT_PATH/kubernetes
cas_dir=${result_dir}/pki
etcd_cas_dir=${cas_dir}/etcd
yaml_dir=${result_dir}/manifests
tmp_dir=/tmp/.k8s/
log_dir=$tmp_dir/logs

mkdir -p $tmp_dir $log_dir $yaml_dir

function format_cas_name() {
	if [ $# -ne 1 ]; then
		echo "invalid name"
		exit 1
	fi
	mv $1-key.pem $1.key
	mv $1.pem $1.crt
}

function openssl_gen_pub_and_key() {
	if [ $# -ne 1 ]; then
		echo "Usage:"
		echo "gen_pub_and_key fname"
		exit 1
	fi
	openssl genrsa -out $1.key 4096
	if [ $? -ne 0 ]; then
		exit 1
	fi
	openssl rsa -in $1.key -pubout -out $1.pub
	if [ $? -ne 0 ]; then
		exit 1
	fi
}

function gen_pub_and_key() {
	if [ $# -ne 2 ]; then
		echo "Usage:"
		echo "gen_pub_and_key csr.json fname"
		exit 1
	fi
	cfssl-newkey $1 | cfssljson -bare $2
	if [ $? -ne 0 ]; then
		exit 1
	fi
	mv $2.csr $2.pub
	mv $2-key.pem $2.key
}

function openssl_gen_ca() {
	if [[ $# -ne 2 ]] && [[ $# -ne 3 ]]; then
		echo "Usage:"
		echo "openssl_gen_ca cert-name common-name [object]"
		exit 1
	fi

	fname=$1
	subj_val="/CN=$2"
	if [ "x$3" != "x" ]; then
		subj_val="$subj_val/O=$3"
	fi

	openssl genrsa -out $fname.key 4096
	if [ $? -ne 0 ]; then
		exit 1
	fi
	openssl req -x509 -new -nodes -key $fname.key -subj "$subj_val" -days 10000 -out $fname.crt
	if [ $? -ne 0 ]; then
		exit 1
	fi
}

function openssl_gen_cert_and_key_with_ca() {
	if [[ $# -ne 4 ]]; then
		echo "Usage:"
		echo "openssl_gen_cert_and_key_with_ca out-name crs-config ca-path ca-key-path"
		exit 1
	fi
	fname=$1
	crs_conf=$2
	ca_path=$3
	ca_key_path=$4

	openssl genrsa -out $fname.key 4096
	if [ $? -ne 0 ]; then
		exit 1
	fi
	openssl req -new -key $fname.key -out $fname.csr -config ${crs_conf}
	if [ $? -ne 0 ]; then
		exit 1
	fi
	openssl x509 -req -in $fname.csr -CA $ca_path -CAkey $ca_key_path -CAcreateserial -out $fname.crt -days 10000 -extensions v3_ext -extfile ${crs_conf}
	if [ $? -ne 0 ]; then
		exit 1
	fi

	echo "---check---"
	openssl x509 -noout -text -in $fname.crt 1>/dev/null
	if [ $? -ne 0 ]; then
		exit 1
	fi
}

function gen_ca() {
	if [ $# -ne 2 ]; then
		echo "Usage:"
		echo "gen_ca csr.json fname"
		exit 1
	fi

	cfssl gencert -initca $1 | cfssljson -bare $2
	format_cas_name $2
}

function gen_cert_and_key_with_ca() {
	if [[ $# -ne 5 ]] && [[ $# -ne 6 ]]; then
		echo "Usage:"
		echo "gen_cert_and_key_with_ca ca-crt ca-key config csr fname [hosts]"
		exit 1
	fi
	if [ $# -eq 5 ]; then
		cfssl gencert -ca=$1 -ca-key=$2 -config=$3 -profile=kubernetes $4 | cfssljson -bare $5
	else
		cfssl gencert -ca=$1 -ca-key=$2 -config=$3 -profile=kubernetes -hostname=$6 $4 | cfssljson -bare $5
	fi
	if [ $? -ne 0 ]; then
		exit 1
	fi

	format_cas_name $5
}

function new_kube_config() {
	if [ $# -ne 6 ]; then
		echo "Usage: "
		echo "new_kube_config filename ca-path host-ip cred-name key-path cert-path"
		exit 1
	fi
	local fname=$1
	local ca_path=$2
	local host_ip=$3
	local cred_name=$4
	local key_path=$5
	local cert_path=$6

	KUBECONFIG=$fname kubectl config set-cluster default-cluster --server=https://$host_ip:6443 --certificate-authority $ca_path --embed-certs
	KUBECONFIG=$fname kubectl config set-credentials $cred_name --client-key $key_path --client-certificate $cert_path --embed-certs
	KUBECONFIG=$fname kubectl config set-context default-system --cluster default-cluster --user $cred_name
	KUBECONFIG=$fname kubectl config use-context default-system
}

function install_node_modules() {
	dnf update -y
	install_node_requires
	if [ $# -eq 1 ]; then
		local requires=(kubernetes-client kubernetes-node kubernetes-kubelet)
		for i in "${!requires[@]}"; do
			rpm -ivh "$1/${requires[$i]}*"
		done
	else
		dnf install -y kubernetes-client kubernetes-node kubernetes-kubelet
	fi
}

function install_controller_modules() {
	if [ $# -eq 1 ]; then
		local requires=(kubernetes-client kubernetes-master etcd coredns)
		for i in "${!requires[@]}"; do
			rpm -ivh "$1/${requires[$i]}*"
		done
	else
		dnf update -y
		dnf install -y kubernetes-client kubernetes-master etcd
	fi
}

function generate_encryption() {
	local ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64)
	cat >$result_dir/encryption-config.yaml <<EOF
kind: EncryptionConfig
apiVersion: v1
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: ${ENCRYPTION_KEY}
      - identity: {}
EOF
}

function deploy_coredns() {
	ip=$1
	mkdir -p $cas_dir/dns
	cat >$cas_dir/dns/Corefile <<EOF
.:53 {
    errors
    health {
      lameduck 5s
    }
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
      pods insecure
      endpoint https://$ip:6443
      tls $cas_dir/ca.crt $cas_dir/admin.key $cas_dir/admin.crt
      kubeconfig $result_dir/admin.conf default-system
      fallthrough in-addr.arpa ip6.arpa
    }
    prometheus :9153
    forward . /etc/resolv.conf {
      max_concurrent 1000
    }
    cache 30
    loop
    reload
    loadbalance
}
EOF
	cat >/usr/lib/systemd/system/coredns.service <<EOF
[Unit]
Description=Kubernetes Core DNS server
Documentation=https://github.com/coredns/coredns
After=network.target

[Service]
ExecStart=bash -c "KUBE_DNS_SERVICE_HOST=$NODE_SERVICE_CLUSTER_DNS coredns -conf $cas_dir/dns/Corefile"

Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
	systemctl enable coredns
	systemctl start coredns
	cat >$yaml_dir/coredns_server.yaml <<EOF
apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
  clusterIP: $NODE_SERVICE_CLUSTER_DNS
  ports:
  - name: dns
    port: 53
    protocol: UDP
  - name: dns-tcp
    port: 53
    protocol: TCP
  - name: metrics
    port: 9153
    protocol: TCP
EOF
	cat >$yaml_dir/coredns_ep.yaml <<EOF
apiVersion: v1
kind: Endpoints
metadata:
  name: kube-dns
  namespace: kube-system
subsets:
  - addresses:
      - ip: $ip
    ports:
      - name: dns-tcp
        port: 53
        protocol: TCP
      - name: dns
        port: 53
        protocol: UDP
      - name: metrics
        port: 9153
        protocol: TCP
EOF
}

function add_admin_cluster_role() {
	cat >$yaml_dir/admin_cluster_role.yaml <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:kube-apiserver-to-kubelet
rules:
  - apiGroups:
      - ""
    resources:
      - nodes/proxy
      - nodes/stats
      - nodes/log
      - nodes/spec
      - nodes/metrics
    verbs:
      - "*"
EOF
	cat >$yaml_dir/admin_cluster_rolebind.yaml <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:kube-apiserver
  namespace: ""
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-apiserver-to-kubelet
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: kubernetes
EOF
}

function pre_init() {
	dnf install -y openssl hostname ipcalc expect openssh
	if [ ! -f ~/.ssh/id_rsa.pub ]; then
		ssh-keygen -t rsa -C "" -f ~/.ssh/id_rsa -P ""
	fi
}

function install_node_requires() {
	dnf install -y docker iSulad conntrack-tools socat containernetworking-plugins
}

function check() {
	local requires=(openssl hostname ipcalc expect ssh-keygen)
	for i in "${!requires[@]}"; do
		which ${requires[$i]}
		if [ $? -ne 0 ]; then
			echo "required ${requires[$i]}, please install first."
			exit 1
		fi
	done
}

pre_init

check

function remote_run() {
	remote_ip=$1
	user=$2
	cmd=$3

	ssh -o StrictHostKeyChecking=no $user@$remote_ip "$cmd"
}

function remote_run_use_password() {
	remote_ip=$1
	user=$2
	passwd=$3
	cmd=$4

	/usr/bin/expect <<-EOF
		set time 30
		spawn ssh -o StrictHostKeyChecking=no $user@$remote_ip \"$cmd\"
		expect {
		"*password:" { send "$passwd\r" }
		}
		expect eof
	EOF
}

function copy_file_to_remote() {
	remote_ip=$1
	user=$2
	src=$3
	dist=$4

	if test -d $src; then
		scp -o StrictHostKeyChecking=no -r $src $user@$remote_ip:$dist
	else
		scp -o StrictHostKeyChecking=no $src $user@$remote_ip:$dist
	fi
}

function copy_file_to_remote_use_password() {
	remote_ip=$1
	user=$2
	passwd=$3
	src=$4
	dist=$5

	if test -d $src; then
		/usr/bin/expect <<-EOF
			set time 30
			spawn scp -o StrictHostKeyChecking=no -r $src $user@$remote_ip:$dist
			expect {
			"*password:" { send "$passwd\r" }
			}
			expect eof
		EOF
	elif test -f $src; then
		/usr/bin/expect <<-EOF
			set time 30
			spawn scp -o StrictHostKeyChecking=no $src $user@$remote_ip:$dist
			expect {
			"*password:" { send "$passwd\r" }
			}
			expect eof
		EOF
	else
		echo "ERROR: src path: $src"
		exit 1
	fi

}

function cleanup_node() {
	echo "---clean k8s services---"
	stop_services=(kubelet kube-proxy docker isulad)
	for ss in ${stop_services[*]}; do
		systemctl stop $ss
	done

	echo "---clean k8s softwares---"
	dnf remove -y kubernetes* docker iSulad conntrack-tools socat containernetworking-plugins

	echo "---clean k8s services systemd configs---"
	remove_files=(/usr/lib/systemd/system/kubelet.service /usr/lib/systemd/system/kube-proxy.service)
	for fname in ${remove_files[*]}; do
		if [ -f $fname ]; then
			rm -f $fname
		fi
	done

	echo "---clean k8s related dirs---"
	remove_dirs=(/etc/kubernetes /etc/cni /opt/cni)
	for fdir in ${remove_dirs[*]}; do
		if [ -d $fdir ]; then
			rm -rf $fdir
		fi
	done
}

function apply_system_resources() {
	echo "-------apply define resouces---------"
	kubectl apply --kubeconfig $result_dir/admin.conf -f $yaml_dir/coredns_server.yaml
	kubectl apply --kubeconfig $result_dir/admin.conf -f $yaml_dir/coredns_ep.yaml
	kubectl apply --kubeconfig $result_dir/admin.conf -f $yaml_dir/admin_cluster_role.yaml
	kubectl apply --kubeconfig $result_dir/admin.conf -f $yaml_dir/admin_cluster_rolebind.yaml
}

function cleanup_system_resources() {
	echo "-------clean define resouces---------"
	kubectl delete --kubeconfig $result_dir/admin.conf -f $yaml_dir/coredns_ep.yaml
	kubectl delete --kubeconfig $result_dir/admin.conf -f $yaml_dir/coredns_server.yaml
	kubectl delete --kubeconfig $result_dir/admin.conf -f $yaml_dir/admin_cluster_rolebind.yaml
	kubectl delete --kubeconfig $result_dir/admin.conf -f $yaml_dir/admin_cluster_role.yaml
}

function cleanup_master() {
	echo "---clean k8s services---"
	stop_services=(kube-controller-manager kube-scheduler kube-apiserver etcd coredns)
	for ss in ${stop_services[*]}; do
		systemctl stop $ss
	done

	echo "---clean k8s softwares---"
	dnf remove -y kubernetes* etcd coredns

	echo "---clean k8s services systemd configs---"
	remove_files=(/usr/lib/systemd/system/kube-apiserver.service /usr/lib/systemd/system/kube-scheduler.service /usr/lib/systemd/system/kube-controller-manager.service /usr/lib/systemd/system/etcd.service)
	for fname in ${remove_files[*]}; do
		if [ -f $fname ]; then
			rm -f $fname
		fi
	done

	echo "---clean k8s related dirs---"
	remove_dirs=(/etc/kubernetes /etc/cni /opt/cni /var/lib/etcd/)
	for fdir in ${remove_dirs[*]}; do
		if [ -d $fdir ]; then
			rm -rf $fdir
		fi
	done
}
