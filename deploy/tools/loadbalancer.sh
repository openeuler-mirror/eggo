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
##- @Description: eggo apiserver loadbalancer deploy tool
##- @Author: zhangxiaoyu
##- @Create: 2021-04-20
#######################################################################

source ./helper.sh

function set_nginx_configs() {
	if [ $# -ne 1 ]; then
		echo "Usage:"
		echo "set_nginx_configs port"
		exit 1
	fi

    local port=$1

    cat >/etc/kubernetes/kube-nginx.conf <<EOF
load_module /usr/lib64/nginx/modules/ngx_stream_module.so;

worker_processes 1;

events {
    worker_connections  1024;
}

stream {
    upstream backend {
        hash $remote_addr consistent;
    }

    server {
        listen 0.0.0.0:$port;
        proxy_connect_timeout 1s;
        proxy_pass backend;
    }
}
EOF

    # insert server
    local insert_line=11
    for i in "${!MASTER_IPS[@]}"; do
        sed -i "$insert_line a\\        server ${MASTER_IPS[$i]}:6443        max_fails=3 fail_timeout=30s;" $result_dir/kube-nginx.conf
        insert_line=$(($insert_line+1))
    done

	cat >/usr/lib/systemd/system/nginx.service <<EOF
[Unit]
Description=kube-apiserver nginx proxy
After=network.target
After=network-online.target
Wants=network-online.target

[Service]
Type=forking
ExecStartPre=/usr/sbin/nginx -c $result_dir/kube-nginx.conf -t
ExecStart=/usr/sbin/nginx -c $result_dir/kube-nginx.conf
ExecReload=/usr/sbin/nginx -c $result_dir/kube-nginx.conf -s reload
PrivateTmp=true
Restart=always
RestartSec=5
StartLimitInterval=0
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

	echo "----enable nginx----"
	systemctl enable nginx
}

function do_pre() {
	mkdir -p /etc/kubernetes
}

do_pre

lb_port=$API_SERVER_EXPOSE_PORT

firewall-cmd --zone=public --add-port=${lb_port}/tcp
echo "-------set_nginx_configs $lb_port-------"
set_nginx_configs "$lb_port"

# start services
systemctl start nginx
