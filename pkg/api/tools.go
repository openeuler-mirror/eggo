package api

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/constants"
	"k8s.io/apimachinery/pkg/util/json"
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

func IsCleanupSchedule(schedule ScheduleType) bool {
	return schedule == SchedulePreCleanup || schedule == SchedulePostCleanup
}

func (hc HostConfig) DeepCopy() (*HostConfig, error) {
	b, err := json.Marshal(hc)
	if err != nil {
		return nil, err
	}
	var result HostConfig
	err = json.Unmarshal(b, &result)
	return &result, err
}

func (cs *ClusterStatus) Show() string {
	var sb strings.Builder
	var fb strings.Builder

	sb.WriteString("-------------------------------\n")
	sb.WriteString("message: ")
	sb.WriteString(cs.Message)
	sb.WriteString("\nsummary: \n")
	if cs.Working {
		sb.WriteString(cs.ControlPlane)
		sb.WriteString("\t\tsuccess")
		sb.WriteString("\n")
	}
	for ip, ok := range cs.StatusOfNodes {
		if ok {
			sb.WriteString(ip)
			sb.WriteString("\t\tsuccess")
			sb.WriteString("\n")
		} else {
			fb.WriteString(ip)
			fb.WriteString("\t\tfailure")
			fb.WriteString("\n")
		}
	}
	sb.WriteString(fb.String())
	sb.WriteString("-------------------------------\n")

	return sb.String()
}

type ClusterConfigOption func(conf *ClusterConfig) *ClusterConfig

func WithEtcdExtrArgs(eargs map[string]string) ClusterConfigOption {
	return func(conf *ClusterConfig) *ClusterConfig {
		conf.EtcdCluster.ExtraArgs = eargs
		return conf
	}
}

func WithAPIServerExtrArgs(eargs map[string]string) ClusterConfigOption {
	return func(conf *ClusterConfig) *ClusterConfig {
		conf.ControlPlane.ApiConf.ExtraArgs = eargs
		return conf
	}
}

func WithControllerManagerExtrArgs(eargs map[string]string) ClusterConfigOption {
	return func(conf *ClusterConfig) *ClusterConfig {
		conf.ControlPlane.ManagerConf.ExtraArgs = eargs
		return conf
	}
}

func WithSchedulerExtrArgs(eargs map[string]string) ClusterConfigOption {
	return func(conf *ClusterConfig) *ClusterConfig {
		conf.ControlPlane.SchedulerConf.ExtraArgs = eargs
		return conf
	}
}

func WithKubeletExtrArgs(eargs map[string]string) ClusterConfigOption {
	return func(conf *ClusterConfig) *ClusterConfig {
		conf.WorkerConfig.KubeletConf.ExtraArgs = eargs
		return conf
	}
}

func WithKubeProxyExtrArgs(eargs map[string]string) ClusterConfigOption {
	return func(conf *ClusterConfig) *ClusterConfig {
		conf.WorkerConfig.ProxyConf.ExtraArgs = eargs
		return conf
	}
}

func ParseScheduleType(schedule string) (ScheduleType, error) {
	switch schedule {
	case string(SchedulePreJoin):
		return SchedulePreJoin, nil
	case string(SchedulePostJoin):
		return SchedulePostJoin, nil
	case string(SchedulePreCleanup):
		return SchedulePreCleanup, nil
	case string(SchedulePostCleanup):
		return SchedulePostCleanup, nil
	default:
		return SchedulePreJoin, fmt.Errorf("invalid schedule type: %s", schedule)
	}
}
