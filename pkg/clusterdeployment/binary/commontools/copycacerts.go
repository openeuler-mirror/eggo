package commontools

import (
	"fmt"
	"os"
	"path/filepath"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
)

var (
	CommonCaCerts = []string{
		"sa.pub",
		"sa.key",
		"ca.crt",
		"ca.key",
		"front-proxy-ca.crt",
		"front-proxy-ca.key",
		"kube-apiserver-etcd-client.crt",
		"kube-apiserver-etcd-client.key",
		"etcd/ca.key",
		"etcd/ca.crt",
	}
)

type CopyCaCertificatesTask struct {
	Cluster *clusterdeployment.ClusterConfig
}

func (ct *CopyCaCertificatesTask) Name() string {
	return "CopyCaCertificatesTask"
}

func checkCaExists(cluster string) bool {
	for _, cert := range CommonCaCerts {
		_, err := os.Lstat(filepath.Join(clusterdeployment.GetCertificateStorePath(cluster), cert))
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (ct *CopyCaCertificatesTask) Run(r runner.Runner, hcf *clusterdeployment.HostConfig) error {
	if !checkCaExists(ct.Cluster.Name) {
		return fmt.Errorf("[certs] cannot find ca certificates")
	}

	return r.Copy(clusterdeployment.GetCertificateStorePath(ct.Cluster.Name), ct.Cluster.Certificate.SavePath)
}
