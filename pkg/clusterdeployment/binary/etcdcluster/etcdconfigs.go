/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2021. All rights reserved.
 * eggo licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: wangfengtu
 * Create: 2021-05-25
 * Description: eggo etcdcluster generate configs implement
 ******************************************************************************/

package etcdcluster

import (
	"fmt"
	"path/filepath"
)

type etcdEnvConfig struct {
	Arch          string
	Ip            string
	Token         string
	Hostname      string
	State         string
	PeerAddresses string
	DataDir       string
	CertsDir      string
	ExtraArgs     map[string]string
}

func createEtcdEnv(conf *etcdEnvConfig) string {
	args := map[string]string{
		"ETCD_ADVERTISE_CLIENT_URLS":       "https://" + conf.Ip + ":2379",
		"ETCD_DATA_DIR":                    conf.DataDir,
		"ETCD_INITIAL_ADVERTISE_PEER_URLS": "https://" + conf.Ip + ":2380",
		"ETCD_INITIAL_CLUSTER":             conf.Token,
		"ETCD_LISTEN_CLIENT_URLS":          "https://127.0.0.1:2379,https://" + conf.Ip + ":2379",
		"ETCD_LISTEN_METRICS_URLS":         "https://" + conf.Ip + ":2381",
		"ETCD_LISTEN_PEER_URLS":            "https://" + conf.Ip + ":2380",
		"ETCD_NAME":                        conf.Hostname,
		"ETCD_SNAPSHOT_COUNT":              "10000",
		"ETCD_INITIAL_CLUSTER_STATE":       conf.State,
		"ETCD_INITIAL_CLUSTER_TOKEN":       conf.PeerAddresses,
		"ETCD_CLIENT_CERT_AUTH":            "true",
		"ETCD_TRUSTED_CA_FILE":             filepath.Join(conf.CertsDir, "etcd", "ca.crt"),
		"ETCD_CERT_FILE":                   filepath.Join(conf.CertsDir, "etcd", "server.crt"),
		"ETCD_KEY_FILE":                    filepath.Join(conf.CertsDir, "etcd", "server.key"),
		"ETCD_PEER_CLIENT_CERT_AUTH":       "true",
		"ETCD_PEER_TRUSTED_CA_FILE":        filepath.Join(conf.CertsDir, "etcd", "ca.crt"),
		"ETCD_PEER_CERT_FILE":              filepath.Join(conf.CertsDir, "etcd", "peer.crt"),
		"ETCD_PEER_KEY_FILE":               filepath.Join(conf.CertsDir, "etcd", "peer.key"),
		"ETCDCTL_ENDPOINTS":                "https://127.0.0.1:2379",
		"ETCDCTL_CA_FILE":                  filepath.Join(conf.CertsDir, "etcd", "ca.crt"),
		"ETCDCTL_KEY_FILE":                 filepath.Join(conf.CertsDir, "etcd", "healthcheck-client.crt"),
		"ETCDCTL_CERT_FILE":                filepath.Join(conf.CertsDir, "etcd", "healthcheck-client.key"),
	}

	if conf.Arch != "amd64" {
		args["ETCD_UNSUPPORTED_ARCH"] = conf.Arch
	}

	if conf.ExtraArgs != nil {
		for k, v := range conf.ExtraArgs {
			args[k] = v
		}
	}

	var envStr string
	for k, v := range args {
		envStr += fmt.Sprintf("%v=%v\n", k, v)
	}

	return envStr
}

func createEtcdService() string {
	return `[Unit]
Description=Etcd Server
After=network.target
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
WorkingDirectory=/var/lib/etcd/
EnvironmentFile=-/etc/etcd/etcd.conf
# set GOMAXPROCS to number of processors
ExecStart=/bin/bash -c "GOMAXPROCS=$(nproc) /usr/bin/etcd"
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`
}
