package api

import (
	"fmt"
	"path/filepath"
	"strings"
)

var (
	EggoHomePath = "/etc/eggo/"
)

func GetCertificateStorePath(cluster string) string {
	return filepath.Join(EggoHomePath, cluster, "pki")
}

func GetEtcdServers(ecc *EtcdClusterConfig) string {
	//etcd_servers="https://${MASTER_IPS[$i]}:2379"
	//etcd_servers="$etcd_servers,https://${MASTER_IPS[$i]}:2379"
	if ecc == nil || len(ecc.Nodes) == 0 {
		return "https://127.0.0.1:2379"
	}
	var sb strings.Builder

	for _, n := range ecc.Nodes {
		sb.WriteString(fmt.Sprintf("https://%s:2379,", n.Address))
	}
	ret := sb.String()
	return ret[0 : len(ret)-1]
}
