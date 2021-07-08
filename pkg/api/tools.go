package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/constants"
)

var (
	EggoHomePath = "/etc/eggo/"
)

func (c ClusterConfig) GetConfigDir() string {
	if c.ConfigDir != "" {
		if !filepath.IsAbs(c.ConfigDir) {
			logrus.Debugf("ignore invalid config dir: %s, just use default", c.ConfigDir)
			return constants.DefaultK8SRootDir
		}
		return filepath.Clean(c.ConfigDir)
	}
	return constants.DefaultK8SRootDir
}

func (c ClusterConfig) GetCertDir() string {
	if c.Certificate.SavePath != "" {
		if !filepath.IsAbs(c.Certificate.SavePath) {
			logrus.Debugf("ignore invalid certificate save path: %s, just use default", c.Certificate.SavePath)
			return constants.DefaultK8SCertDir
		}
		return filepath.Clean(c.Certificate.SavePath)
	}
	return constants.DefaultK8SCertDir
}

func (c ClusterConfig) GetManifestDir() string {
	if c.ConfigDir != "" {
		if !filepath.IsAbs(c.ConfigDir) {
			logrus.Debugf("ignore invalid config dir: %s, just use default", c.ConfigDir)
			return constants.DefaultK8SManifestsDir
		}
		return filepath.Join(filepath.Clean(c.ConfigDir), "manifests")
	}
	return constants.DefaultK8SManifestsDir
}

func (p PackageSrcConfig) GetPkgDstPath() string {
	if p.DstPath == "" {
		return constants.DefaultPackagePath
	}

	return p.DstPath
}

func GetClusterHomePath(cluster string) string {
	return filepath.Join(EggoHomePath, cluster)
}

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
