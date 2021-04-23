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
##- @Description: eggo deploy command entry point
##- @Author: haozi007
##- @Create: 2021-04-10
#######################################################################

source ./helper.sh

function do_install_etcd() {
	local current_ip=$1
	local node_list=''
	if [ x"${current_ip}" == x"" ]; then
		echo "ERROR: invalid current ip"
		exit 1
	fi
	for i in "${!ETCD_CLUSTER_IPS[@]}"; do
		if [ $i -eq 0 ]; then
			node_list="${ETCD_CLUSTER_NAMES[$i]}=https://${ETCD_CLUSTER_IPS[$i]}:2380"
		else
			node_list="$node_list,${ETCD_CLUSTER_NAMES[$i]}=https://${ETCD_CLUSTER_IPS[$i]}:2380"
		fi
	done
	./etcd.sh config $ETCD_CLUSTER_TOKEN $current_ip $node_list
}

function do_first_controller_start_services() {
	# install etcd at current controller
	do_install_etcd $1

	# install kube-apiserver, controller-manager, scheduler and so on.
	./install_controller.sh $1

	add_admin_cluster_role

	# deploy coredns at first controller
	deploy_coredns $1
}

function do_first_controller_install() {
	local current_ip=$1
	local host_name=$2
	if [ x"${current_ip}" == x"" ]; then
		current_ip=${NODE_IPS[0]}
	fi
	if [ x"${host_name}" == x"" ]; then
		host_name=${MASTER_NAMES[0]}
	fi
	hostnamectl set-hostname $host_name

	# install custom modules
	install_controller_modules $MODULE_SAVE_PATH

	# generate certificates and keys
	rm -rf $result_dir
	./openssl_new_cas.sh

	# generate kube configs
	./new_kubeconfig.sh

	# remove prefix of etcd certificates
	rm -f $etcd_cas_dir/peer.crt $etcd_cas_dir/peer.key $etcd_cas_dir/server.crt $etcd_cas_dir/server.key
	cp $etcd_cas_dir/$host_name-peer.crt $etcd_cas_dir/peer.crt
	cp $etcd_cas_dir/$host_name-peer.key $etcd_cas_dir/peer.key
	cp $etcd_cas_dir/$host_name-server.crt $etcd_cas_dir/server.crt
	cp $etcd_cas_dir/$host_name-server.key $etcd_cas_dir/server.key

	do_first_controller_start_services ${current_ip} &
}

function do_join_controller_install() {
	local current_ip=$1
	if [ x"${current_ip}" == x"" ]; then
		echo "ERROR: Cannot join controller without ip"
		exit 1
	fi
	install_controller_modules $MODULE_SAVE_PATH

	# install etcd at current controller
	do_install_etcd $current_ip

	# install kube-apiserver, controller-manager, scheduler and so on.
	./install_controller.sh $current_ip
}

function do_join_node_install() {
	install_node_modules $MODULE_SAVE_PATH

	# install kubelet, kube-proxy, cni configs and so on.
	./node.sh $1
}

function do_join_loadbalancer_install() {
	install_loadbalancer_modules $MODULE_SAVE_PATH

	# install nginx
	./loadbalancer.sh
}

function join_new_controller() {
	ip=$1
	name=$2

	rm -rf $tmp_dir/$name
	mkdir -p $tmp_dir/$name
	cp -r $result_dir $tmp_dir/$name
	rm -f $tmp_dir/$name/kubernetes/pki/etcd/peer.crt $tmp_dir/$name/kubernetes/pki/etcd/peer.key $tmp_dir/$name/kubernetes/pki/etcd/server.crt $tmp_dir/$name/kubernetes/pki/etcd/server.key
	cp $tmp_dir/$name/kubernetes/pki/etcd/$name-peer.crt $tmp_dir/$name/kubernetes/pki/etcd/peer.crt
	cp $tmp_dir/$name/kubernetes/pki/etcd/$name-peer.key $tmp_dir/$name/kubernetes/pki/etcd/peer.key
	cp $tmp_dir/$name/kubernetes/pki/etcd/$name-server.crt $tmp_dir/$name/kubernetes/pki/etcd/server.crt
	cp $tmp_dir/$name/kubernetes/pki/etcd/$name-server.key $tmp_dir/$name/kubernetes/pki/etcd/server.key

	# update hostname to user set name
	if [ x"$name" != x"" ]; then
		remote_run $ip $BOOTSTRAP_NODE_USER "hostnamectl set-hostname $name"
	fi

	remote_run $ip $BOOTSTRAP_NODE_USER "mkdir -p $MODULE_SAVE_PATH"
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER $MODULE_SAVE_PATH $MODULE_SAVE_PATH/..
	# copy certificates to new controller
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER $tmp_dir/$name/kubernetes /etc
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER "$current_dir" ~
	remote_run $ip $BOOTSTRAP_NODE_USER "cd ~/tools/ && ./deploy.sh install-controller $ip"
}

function join_new_node() {
	ip=$1
	name=$2

	rm -f $result_dir/kubelet.conf
	cp $result_dir/$name-kubelet.conf $result_dir/kubelet.conf

	rm -rf $tmp_dir/$name
	mkdir -p $tmp_dir/$name/kubernetes/pki
	cp $result_dir/$name-kubelet.conf $tmp_dir/$name/kubernetes/kubelet.conf
	cp $result_dir/kube-proxy.conf $tmp_dir/$name/kubernetes/
	cp $result_dir/pki/ca.crt $tmp_dir/$name/kubernetes/pki

	# update hostname to user set name
	if [ x"$name" != x"" ]; then
		remote_run $ip $BOOTSTRAP_NODE_USER "hostnamectl set-hostname $name"
	fi

	remote_run $ip $BOOTSTRAP_NODE_USER "mkdir -p $MODULE_SAVE_PATH"
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER $MODULE_SAVE_PATH $MODULE_SAVE_PATH/..
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER $tmp_dir/$name/kubernetes /etc
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER "$current_dir" ~
	remote_run $ip $BOOTSTRAP_NODE_USER "cd ~/tools/ && ./deploy.sh install-node $name"
}

function join_loadbalancer_install() {
	ip=$1

	remote_run $ip $BOOTSTRAP_NODE_USER "mkdir -p $MODULE_SAVE_PATH"
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER $MODULE_SAVE_PATH $MODULE_SAVE_PATH/..
	copy_file_to_remote $ip $BOOTSTRAP_NODE_USER "$current_dir" ~
	remote_run $ip $BOOTSTRAP_NODE_USER "cd ~/tools/ && ./deploy.sh install-lb"
}

function deploy_cluster() {
	for i in "${!MASTER_IPS[@]}"; do
		if [ $i -eq 0 ]; then
			do_first_controller_install ${MASTER_IPS[$i]} ${MASTER_NAMES[$i]} >$log_dir/first_controller.log 2>&1
		else
			# copy id_rsa.pub into remote authorized_keys, ensure unpasword ssh and scp can working
			remote_run_use_password ${MASTER_IPS[$i]} $BOOTSTRAP_NODE_USER $BOOTSTRAP_NODE_PASSWORD "mkdir -p ~/.ssh"
			copy_file_to_remote_use_password ${MASTER_IPS[$i]} $BOOTSTRAP_NODE_USER $BOOTSTRAP_NODE_PASSWORD ~/.ssh/id_rsa.pub ~/.ssh/authorized_keys

			join_new_controller ${MASTER_IPS[$i]} ${MASTER_NAMES[$i]} >$log_dir/${MASTER_NAMES[$i]}.log 2>&1 &
		fi
	done
	wait

	# apply system resources
	apply_system_resources

	for i in "${!NODE_IPS[@]}"; do

		# copy id_rsa.pub into remote authorized_keys, ensure unpasword ssh and scp can working
		remote_run_use_password ${NODE_IPS[$i]} $BOOTSTRAP_NODE_USER $BOOTSTRAP_NODE_PASSWORD "mkdir -p ~/.ssh"
		copy_file_to_remote_use_password ${NODE_IPS[$i]} $BOOTSTRAP_NODE_USER $BOOTSTRAP_NODE_PASSWORD ~/.ssh/id_rsa.pub ~/.ssh/authorized_keys

		join_new_node ${NODE_IPS[$i]} ${NODE_NAMES[$i]} >$log_dir/${NODE_NAMES[$i]}.log 2>&1 &
	done
	wait

	# TODO: 测试集群功能是否符合预期
	echo "----begin test k8s cluster----"
	sleep 5
	kubectl get --kubeconfig $result_dir/admin.conf nodes

	echo "
Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

	mkdir -p $HOME/.kube
	sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
	sudo chown $(id -u):$(id -g) $HOME/.kube/config

Or add KUBECONFIG ENV

	export KUBECONFIG=/etc/kubernetes/admin.conf
"
}

function deploy_cluster_with_loadbalancer() {
	# copy id_rsa.pub into remote authorized_keys, ensure unpasword ssh and scp can working
	remote_run_use_password $API_SERVER_EXPOSE_IP $BOOTSTRAP_NODE_USER $BOOTSTRAP_NODE_PASSWORD "mkdir -p ~/.ssh"
	copy_file_to_remote_use_password $API_SERVER_EXPOSE_IP $BOOTSTRAP_NODE_USER $BOOTSTRAP_NODE_PASSWORD ~/.ssh/id_rsa.pub ~/.ssh/authorized_keys

	join_loadbalancer_install $API_SERVER_EXPOSE_IP >$log_dir/loadbalancer.log 2>&1 &

	deploy_cluster
}

function do_clean_first_master() {
	cleanup_master
}

function do_clean_master() {
	cleanup_master
	rm ~/.ssh/authorized_keys
}

function do_clean_node() {
	cleanup_node
	rm ~/.ssh/authorized_keys
}

function do_clean_loadbalancer() {
	cleanup_loadbalancer
	rm ~/.ssh/authorized_keys
}

function clean_cluster() {
	# clean system resources
	cleanup_system_resources

	# first, clean nodes
	for i in "${!NODE_IPS[@]}"; do
		echo "----------------cleanup Node: ${NODE_IPS[$i]}------------------"
		remote_run ${NODE_IPS[$i]} $BOOTSTRAP_NODE_USER "cd ~/tools/ && ./deploy.sh clean-node" &
	done
	wait

	# second, clean other master
	for i in "${!MASTER_IPS[@]}"; do
		if [ $i -ne 0 ]; then
			echo "----------------cleanup Master: ${MASTER_IPS[$i]}------------------"
			remote_run ${MASTER_IPS[$i]} $BOOTSTRAP_NODE_USER "cd ~/tools/ && ./deploy.sh clean-master" &
		fi
	done

	wait
	# finally, clean init master
	echo "----------------cleanup Master: $(hostname)------------------"
	do_clean_first_master
}

function clean_cluster_with_loadbalancer() {
	clean_cluster

	remote_run ${API_SERVER_EXPOSE_IP} $BOOTSTRAP_NODE_USER "cd ~/tools/ && ./deploy.sh clean-lb"
}

function usage() {
	echo "usage: deploy.sh install-cluster"
	echo "usage: deploy.sh install-cluster-with-lb"
	echo "usage: deploy.sh init-controller controller-host-ip [hostname]"
	echo "usage: deploy.sh install-controller controller-host-ip"
	echo "usage: deploy.sh install-node [hostname]"
	echo "usage: deploy.sh install-lb"
	echo "usage: deploy.sh clean-cluster"
	echo "usage: deploy.sh clean-cluster-with-lb"
	echo "usage: deploy.sh clean-master"
	echo "usage: deploy.sh clean-node"
	echo "usage: deploy.sh clean-lb"
	echo ""
	echo "example: deploy.sh init-controller 192.168.1.1"
	echo "example: deploy.sh install-controller 192.168.1.2"
	echo "usage: deploy.sh install-node work1"
}

if [ x"$1" == x"init-controller" ]; then
	do_first_controller_install $2 $3
elif [ x"$1" == x"install-controller" ]; then
	do_join_controller_install $2
elif [ x"$1" == x"install-node" ]; then
	do_join_node_install $2
elif [ x"$1" == x"install-lb" ]; then
	do_join_loadbalancer_install
elif [ x"$1" == x"install-cluster" ]; then
	deploy_cluster
elif [ x"$1" == x"install-cluster-with-lb" ]; then
	deploy_cluster_with_loadbalancer
elif [ x"$1" == x"clean-cluster" ]; then
	clean_cluster
elif [ x"$1" == x"clean-cluster-with-lb" ]; then
	clean_cluster_with_loadbalancer
elif [ x"$1" == x"clean-master" ]; then
	do_clean_master
elif [ x"$1" == x"clean-node" ]; then
	do_clean_node
elif [ x"$1" == x"clean-lb" ]; then
	do_clean_loadbalancer
else
	usage
	exit 1
fi
