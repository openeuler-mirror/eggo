package api

import (
	"path/filepath"
)

var (
	EggoHomePath = "/etc/eggo/"
)

func GetCertificateStorePath(cluster string) string {
	return filepath.Join(EggoHomePath, cluster, "pki")
}
