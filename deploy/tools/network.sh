#!/bin/bash

function do_pre() {
    mkdir -p /etc/cni/net.d
    mkdir -p /opt/cni
}

function add_default_cni_configs() {
    cat >/etc/cni/net.d/10-bridge.conf <<EOF
{
  "cniVersion": "0.3.1",
  "name": "bridge",
  "type": "bridge",
  "bridge": "cnio0",
  "isGateway": true,
  "ipMasq": true,
  "ipam": {
    "type": "host-local",
    "subnet": "10.244.0.0/16",
    "gateway": "10.244.0.1"
  },
  "dns": {
    "nameservers": [
      "10.244.0.1"
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