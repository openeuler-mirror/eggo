#!/bin/bash

source ./helper.sh

function set_apiserver_configs() {
	if [ $# -ne 3 ]; then
		echo "Usage:"
		echo "set_apiserver_configs api-server-ip etcd-server-ips service-cluster-ip-cidr"
		exit 1
	fi

	cat >$result_dir/apiserver <<EOF
KUBE_ADVERTIS_ADDRESS="--advertise-address=$1"
KUBE_ALLOW_PRIVILEGED="--allow-privileged=true"
KUBE_AUTHORIZATION_MODE="--authorization-mode=Node,RBAC"
KUBE_ENABLE_ADMISSION_PLUGINS="--enable-admission-plugins=NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota"
KUBE_SECURE_PORT="--secure-port=6443"
KUBE_ENABLE_BOOTSTRAP_TOKEN_AUTH="--enable-bootstrap-token-auth=true"
KUBE_ETCD_CAFILE="--etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt"
KUBE_ETCD_CERTFILE="--etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt"
KUBE_ETCD_KEYFILE="--etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key"
KUBE_ETCD_SERVERS="--etcd-servers=$2"
KUBE_CLIENT_CA_FILE="--client-ca-file=/etc/kubernetes/pki/ca.crt"
KUBE_KUBELET_CLIENT_CERT="--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt"
KUBE_KUBELET_CLIENT_KEY="--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key"
KUBE_KUBELET_HTTPS="--kubelet-https=true"
KUBE_PROXY_CLIENT_CERT_FILE="--proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt"
KUBE_PROXY_CLIENT_KEY_FILE="--proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key"
KUBE_TLS_CERT_FILE="--tls-cert-file=/etc/kubernetes/pki/apiserver.crt"
KUBE_TLS_PRIVATE_KEY_FILE="--tls-private-key-file=/etc/kubernetes/pki/apiserver.key"
KUBE_SERVICE_CLUSTER_IP_RANGE="--service-cluster-ip-range=$3"
KUBE_SERVICE_ACCOUNT_ISSUER="--service-account-issuer=https://kubernetes.default.svc.cluster.local"
KUBE_SERVICE_ACCOUNT_KEY_FILE="--service-account-key-file=/etc/kubernetes/pki/sa.pub"
KUBE_SERVICE_ACCOUNT_SIGN_KEY_FILE="--service-account-signing-key-file=/etc/kubernetes/pki/sa.key"
KUBE_SERVICE_NODE_PORT_RANGE="--service-node-port-range=30000-32767"
KUBE_REQUEST_HEADER_ALLOWED_NAME="--requestheader-allowed-names=front-proxy-client"
KUBE_REQUEST_HEADER_CLIENT_CA="--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt"
KUBE_REQUEST_HEADER_EXTRA_HEADER_PREF="--requestheader-extra-headers-prefix=X-Remote-Extra-"
KUBE_REQUEST_HEADER_GROUP_HEADER="--requestheader-group-headers=X-Remote-Group"
KUBE_REQUEST_HEADER_USERNAME_HEADER="--requestheader-username-headers=X-Remote-User"
KUB_ENCRYPTION_PROVIDER_CONF="--encryption-provider-config=/etc/kubernetes/encryption-config.yaml"
KUBE_API_ARGS=""
EOF

	cat >/usr/lib/systemd/system/kube-apiserver.service <<EOF
[Unit]
Description=Kubernetes API Server
Documentation=https://kubernetes.io/docs/reference/generated/kube-apiserver/
After=network.target
After=etcd.service

[Service]
EnvironmentFile=-/etc/kubernetes/apiserver
ExecStart=/usr/bin/kube-apiserver \\
	    \$KUBE_ADVERTIS_ADDRESS \\
	    \$KUBE_ALLOW_PRIVILEGED \\
	    \$KUBE_AUTHORIZATION_MODE \\
	    \$KUBE_ENABLE_ADMISSION_PLUGINS \\
 	    \$KUBE_SECURE_PORT \\
	    \$KUBE_ENABLE_BOOTSTRAP_TOKEN_AUTH \\
	    \$KUBE_ETCD_CAFILE \\
	    \$KUBE_ETCD_CERTFILE \\
	    \$KUBE_ETCD_KEYFILE \\
	    \$KUBE_ETCD_SERVERS \\
	    \$KUBE_CLIENT_CA_FILE \\
	    \$KUBE_KUBELET_CLIENT_CERT \\
	    \$KUBE_KUBELET_CLIENT_KEY \\
	    \$KUBE_PROXY_CLIENT_CERT_FILE \\
	    \$KUBE_PROXY_CLIENT_KEY_FILE \\
	    \$KUBE_TLS_CERT_FILE \\
	    \$KUBE_TLS_PRIVATE_KEY_FILE \\
	    \$KUBE_SERVICE_CLUSTER_IP_RANGE \\
	    \$KUBE_SERVICE_ACCOUNT_ISSUER \\
	    \$KUBE_SERVICE_ACCOUNT_KEY_FILE \\
	    \$KUBE_SERVICE_ACCOUNT_SIGN_KEY_FILE \\
	    \$KUBE_SERVICE_NODE_PORT_RANGE \\
	    \$KUBE_LOGTOSTDERR \\
	    \$KUBE_LOG_LEVEL \\
	    \$KUBE_API_PORT \\
	    \$KUBELET_PORT \\
	    \$KUBE_ALLOW_PRIV \\
	    \$KUBE_SERVICE_ADDRESSES \\
	    \$KUBE_ADMISSION_CONTROL \\
	    \$KUBE_REQUEST_HEADER_ALLOWED_NAME \\
	    \$KUBE_REQUEST_HEADER_CLIENT_CA_FILE \\
	    \$KUBE_REQUEST_HEADER_EXTRA_HEADER_PREF \\
	    \$KUBE_REQUEST_HEADER_GROUP_HEADER \\
	    \$KUBE_REQUEST_HEADER_USERNAME_HEADER \\
	    \$KUB_ENCRYPTION_PROVIDER_CONF \\
	    \$KUBE_API_ARGS
Restart=on-failure
Type=notify
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

	echo "----enable apiserver----"
	systemctl enable kube-apiserver
}

function set_controller_manager_configs() {
	if [ $# -ne 2 ]; then
		echo "Usage:"
		echo "set_apiserver_configs cluster-ip-cidr service-cluster-ip-cidr"
		exit 1
	fi

	cat >$result_dir/controller-manager <<EOF
KUBE_BIND_ADDRESS="--bind-address=127.0.0.1"
KUBE_CLUSTER_CIDR="--cluster-cidr=$1"
KUBE_CLUSTER_NAME="--cluster-name=kubernetes"
KUBE_CLUSTER_SIGNING_CERT_FILE="--cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt"
KUBE_CLUSTER_SIGNING_KEY_FILE="--cluster-signing-key-file=/etc/kubernetes/pki/ca.key"
KUBE_KUBECONFIG="--kubeconfig=/etc/kubernetes/controller-manager.conf"
KUBE_LEADER_ELECT="--leader-elect=true"
KUBE_ROOT_CA_FILE="--root-ca-file=/etc/kubernetes/pki/ca.crt"
KUBE_SERVICE_ACCOUNT_PRIVATE_KEY_FILE="--service-account-private-key-file=/etc/kubernetes/pki/sa.key"
KUBE_SERVICE_CLUSTER_IP_RANGE="--service-cluster-ip-range=$2"
KUBE_USE_SERVICE_ACCOUNT_CRED="--use-service-account-credentials=true"
KUBE_CONTROLLER_MANAGER_ARGS="--v=2"
KUBE_AUTHENTICATION_KUBECONFIG="--authentication-kubeconfig=/etc/kubernetes/controller-manager.conf"
KUBE_AUTHORIZATION_KUBECONFIG="--authorization-kubeconfig=/etc/kubernetes/controller-manager.conf"
KUBE_REQUEST_HEADER_CLIENT_CA="--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt"
EOF

	cat >/usr/lib/systemd/system/kube-controller-manager.service <<EOF
[Unit]
Description=Kubernetes Controller Manager
Documentation=https://kubernetes.io/docs/reference/generated/kube-controller-manager/

[Service]
EnvironmentFile=-/etc/kubernetes/controller-manager
ExecStart=/usr/bin/kube-controller-manager \\
	    \$KUBE_BIND_ADDRESS \\
	    \$KUBE_LOGTOSTDERR \\
	    \$KUBE_LOG_LEVEL \\
	    \$KUBE_CLUSTER_CIDR \\
	    \$KUBE_CLUSTER_NAME \\
	    \$KUBE_CLUSTER_SIGNING_CERT_FILE \\
	    \$KUBE_CLUSTER_SIGNING_KEY_FILE \\
	    \$KUBE_KUBECONFIG \\
	    \$KUBE_LEADER_ELECT \\
	    \$KUBE_ROOT_CA_FILE \\
	    \$KUBE_SERVICE_ACCOUNT_PRIVATE_KEY_FILE \\
	    \$KUBE_SERVICE_CLUSTER_IP_RANGE \\
	    \$KUBE_USE_SERVICE_ACCOUNT_CRED \\
	    \$KUBE_AUTHENTICATION_KUBECONFIG \\
	    \$KUBE_AUTHORIZATION_KUBECONFIG \\
	    \$KUBE_REQUEST_HEADER_CLIENT_CA \\
	    \$KUBE_CONTROLLER_MANAGER_ARGS
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

	echo "----enable controller manager----"
	systemctl enable kube-controller-manager
}

function set_scheduler_configs() {

	cat >$result_dir/scheduler <<EOF
KUBE_CONFIG="--kubeconfig=/etc/kubernetes/scheduler.conf"
KUBE_AUTHENTICATION_KUBE_CONF="--authentication-kubeconfig=/etc/kubernetes/scheduler.conf"
KUBE_AUTHORIZATION_KUBE_CONF="--authorization-kubeconfig=/etc/kubernetes/scheduler.conf"
KUBE_BIND_ADDR="--bind-address=127.0.0.1"
KUBE_LEADER_ELECT="--leader-elect=true"
KUBE_SCHEDULER_ARGS=""
EOF

	cat >/usr/lib/systemd/system/kube-scheduler.service <<EOF
[Unit]
Description=Kubernetes Scheduler Plugin
Documentation=https://kubernetes.io/docs/reference/generated/kube-scheduler/

[Service]
EnvironmentFile=-/etc/kubernetes/scheduler
ExecStart=/usr/bin/kube-scheduler \\
	    \$KUBE_LOGTOSTDERR \\
	    \$KUBE_LOG_LEVEL \\
	    \$KUBE_CONFIG \\
	    \$KUBE_AUTHENTICATION_KUBE_CONF \\
	    \$KUBE_AUTHORIZATION_KUBE_CONF \\
	    \$KUBE_BIND_ADDR \\
	    \$KUBE_LEADER_ELECT \\
	    \$KUBE_SCHEDULER_ARGS
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

	echo "----enable scheduler----"
	systemctl enable kube-scheduler
}

etcd_servers=''
for i in "${!MASTER_IPS[@]}"; do
	if [ $i -eq 0 ]; then
		etcd_servers="https://${MASTER_IPS[$i]}:2379"
	else
		etcd_servers="$etcd_servers,https://${MASTER_IPS[$i]}:2379"
	fi
done

echo "-------set_apiserver_configs $API_SERVER_IP $etcd_servers $SERVICE_CLUSTER_IP_RANGE-------"
set_apiserver_configs "$API_SERVER_IP" "$etcd_servers" "$SERVICE_CLUSTER_IP_RANGE"

echo "-------set_controller_manager_configs $CLUSTER_IP_RANGE $SERVICE_CLUSTER_IP_RANGE ------------"
set_controller_manager_configs "$CLUSTER_IP_RANGE" "$SERVICE_CLUSTER_IP_RANGE"

echo "-------set_scheduler_configs----------"
set_scheduler_configs

# start services
systemctl start kube-apiserver kube-controller-manager kube-scheduler
