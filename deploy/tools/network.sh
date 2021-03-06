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
##- @Description: eggo cni network deploy tool
##- @Author: haozi007
##- @Create: 2021-04-10
#######################################################################

source ./configs

function do_pre() {
  mkdir -p /etc/cni/net.d
  mkdir -p /opt/cni
  ln -s /usr/libexec/cni/ /opt/cni/bin
}

function add_default_cni_configs() {
  cat >/etc/cni/net.d/99-bridge.conf <<EOF
{
  "cniVersion": "0.3.1",
  "name": "bridge",
  "type": "bridge",
  "bridge": "cni0",
  "isGateway": true,
  "ipMasq": true,
  "ipam": {
    "type": "host-local",
    "subnet": "$CLUSTER_IP_RANGE",
    "gateway": "$CLUSTER_GATEWAY",
    "routes": [
      {"dst": "0.0.0.0/0"}
    ]
  },
  "dns": {
    "nameservers": [
      "$CLUSTER_GATEWAY"
    ]
  }
}
EOF
  cat >/etc/cni/net.d/99-loopback.conf <<EOF
{
    "cniVersion": "0.3.1",
    "name": "lo",
    "type": "loopback"
}
EOF
}

do_pre

if [[ x"$1" == x"default" ]]; then
  add_default_cni_configs
fi
