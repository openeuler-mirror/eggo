#!/bin/sh
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
##- @Description: eggo etcd deploy tool
##- @Author: WangFengTu
##- @Create: 2021-04-10
#######################################################################

# $1 具体的操作
# $2 集群token
# $3 本机IP地址
# $4 集群所有地址列表，key=value[,key=value]格式，key是etcd在集群中的名称(即--name指定的名称)，
#    value是名称对应的地址

function usage() {
	echo "usage: etcd.sh config/deploy TOKEN IP CLUSTER_NODE_LIST"
	echo "usage: etcd.sh test"
	echo "usage: etcd.sh install"
	echo ""
	echo "example: etcd.sh config etcd-cluster-1 192.168.0.11 node1=https://192.168.0.11:2380,node2=https://192.168.0.12:2380"
}

function config_etcd() {
	sed -i "/User=etcd/d" /usr/lib/systemd/system/etcd.service
	systemctl daemon-reload

	echo $1 $2 $3

	echo "ETCD_UNSUPPORTED_ARCH=arm64" >/etc/etcd/etcd.conf
	echo "ETCD_ADVERTISE_CLIENT_URLS=https://$2:2379" >>/etc/etcd/etcd.conf
	echo "ETCD_CERT_FILE=/etc/kubernetes/pki/etcd/server.crt" >>/etc/etcd/etcd.conf
	echo "ETCD_CLIENT_CERT_AUTH=true" >>/etc/etcd/etcd.conf
	echo "ETCD_DATA_DIR=/var/lib/etcd" >>/etc/etcd/etcd.conf
	echo "ETCD_INITIAL_ADVERTISE_PEER_URLS=https://$2:2380" >>/etc/etcd/etcd.conf
	echo "ETCD_INITIAL_CLUSTER=$3" >>/etc/etcd/etcd.conf
	echo "ETCD_KEY_FILE=/etc/kubernetes/pki/etcd/server.key" >>/etc/etcd/etcd.conf
	echo "ETCD_LISTEN_CLIENT_URLS=https://127.0.0.1:2379,https://$2:2379" >>/etc/etcd/etcd.conf
	echo "ETCD_LISTEN_METRICS_URLS=https://$2:2381" >>/etc/etcd/etcd.conf
	echo "ETCD_LISTEN_PEER_URLS=https://$2:2380" >>/etc/etcd/etcd.conf
	echo "ETCD_NAME=$(hostname)" >>/etc/etcd/etcd.conf
	echo "ETCD_PEER_CERT_FILE=/etc/kubernetes/pki/etcd/peer.crt" >>/etc/etcd/etcd.conf
	echo "ETCD_PEER_CLIENT_CERT_AUTH=true" >>/etc/etcd/etcd.conf
	echo "ETCD_PEER_KEY_FILE=/etc/kubernetes/pki/etcd/peer.key" >>/etc/etcd/etcd.conf
	echo "ETCD_TRUSTED_CA_FILE=/etc/kubernetes/pki/etcd/ca.crt" >>/etc/etcd/etcd.conf
	echo "ETCD_SNAPSHOT_COUNT=10000" >>/etc/etcd/etcd.conf
	echo "ETCD_PEER_TRUSTED_CA_FILE=/etc/kubernetes/pki/etcd/ca.crt" >>/etc/etcd/etcd.conf
	echo "ETCD_INITIAL_CLUSTER_STATE=new" >>/etc/etcd/etcd.conf
	echo "ETCD_INITIAL_CLUSTER_TOKEN=$1" >>/etc/etcd/etcd.conf

	cat /etc/etcd/etcd.conf
}

function test_etcd() {
	etcd_list=$(cat /etc/etcd/etcd.conf | grep "ETCD_INITIAL_CLUSTER=")
	etcd_list=${etcd_list#*=}
	arr=(${etcd_list//,/ })

	echo ${arr[@]}

	endpoints=""
	for node in ${arr[@]}; do
		endpoint=${node#*=}

		if [ x"$endpoints" == "x" ]; then
			endpoints="$endpoint"
		else
			endpoints="$endpoints,$endpoint"
		fi
	done

	endpoints=$(echo $endpoints | sed -e "s/2380/2379/g")

	echo $endpoints

	ETCDCTL_API=3 etcdctl -w table endpoint status --endpoints=$endpoints --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key
	if [ $? != 0 ]; then
		echo "test etcd status failed"
		exit 1
	fi

	ETCDCTL_API=3 etcdctl endpoint health --endpoints=$endpoints --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key
	if [ $? != 0 ]; then
		echo "test etcd health failed"
		exit 1
	fi

	# test put value
	arr=(${endpoints//,/ })
	ETCDCTL_API=3 etcdctl put /test_etcd test_etcd_ok --endpoints=${arr[0]} --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key
	if [ $? != 0 ]; then
		echo "test etcd put failed"
		exit 1
	fi

	# test get value
	for node in ${arr[@]}; do
		ETCDCTL_API=3 etcdctl get /test_etcd --endpoints=$node --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key | grep test_etcd_ok
		if [ $? != 0 ]; then
			echo "test etcd get data from endpoint $node failed"
			exit 1
		fi
	done

	# test delete value
	ETCDCTL_API=3 etcdctl del /test_etcd --endpoints=${arr[0]} --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key
	if [ $? != 0 ]; then
		echo "test etcd put failed"
		exit 1
	fi

	# make sure delete success in all endpoints
	for node in ${arr[@]}; do
		ETCDCTL_API=3 etcdctl get /test_etcd --endpoints=$node --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key | grep test_etcd_ok
		if [ $? == 0 ]; then
			echo "test etcd get data from endpoint $node should failed because it's already deleted"
			exit 1
		fi
	done
}

firewall-cmd --zone=public --add-port=2379/tcp
firewall-cmd --zone=public --add-port=2380/tcp
firewall-cmd --zone=public --add-port=2381/tcp
firewall-cmd --runtime-to-permanent

if [ x"$1" == x"config" ] || [ x"$1" == x"deploy" ]; then
	if [ x"$2" == "x" ] || [ x"$3" == "x" ] || [ x"$4" == "x" ]; then
		usage
		exit 1
	fi
fi

if [ x"$1" == x"config" ]; then
	config_etcd $2 $3 $4
elif [ x"$1" == x"test" ]; then
	test_etcd
	echo "test etcd success"
	exit 0
elif [ x"$1" == x"install" ]; then
	rpm -Uvh --force ./etcd*.rpm
elif [ x"$1" == x"deploy" ]; then
	rpm -Uvh --force ./etcd*.rpm
	config_etcd $2 $3 $4
else
	usage
	exit 1
fi

systemctl daemon-reload
systemctl enable etcd
systemctl restart etcd
