package clusterdeployment

import "path/filepath"

var (
	EggoHomePath = "/etc/eggo/"
)

func GetCertificateStorePath(cluster string) string {
	return filepath.Join(EggoHomePath, cluster, "pki")
}

func GetMasterIPList(c *ClusterConfig) []string {
	var masters []string
	for _, n := range c.Nodes {
		if (n.Type & Master) != 0 {
			masters = append(masters, n.Address)
			continue
		}
	}

	return masters
}
