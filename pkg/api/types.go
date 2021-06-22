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
 * Author: haozi007
 * Create: 2021-05-11
 * Description: cluster deploy types
 ******************************************************************************/

package api

import (
	"path/filepath"
	"time"

	"gitee.com/openeuler/eggo/pkg/constants"
	"github.com/sirupsen/logrus"
)

const (
	Master      = 0x1
	Worker      = 0x2
	ETCD        = 0x4
	LoadBalance = 0x8
)

type OpenPorts struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp/udp
}

type Packages struct {
	Name string `json:"name"`
	Type string `json:"type"` // repo, pkg, binary
	Dst  string `json:"dstpath"`
}

type HostConfig struct {
	Arch           string       `json:"arch"`
	Name           string       `json:"name"`
	Address        string       `json:"address"`
	Port           int          `json:"port"`
	ExtraIPs       []string     `json:"extra-ips"`
	OpenPorts      []*OpenPorts `json:"open-ports"`
	UserName       string       `json:"username"`
	Password       string       `json:"password"`
	PrivateKey     string       `json:"private-key"`
	PrivateKeyPath string       `json:"private-key-path"`
	Packages       []*Packages  `json:"packages"`

	// 0x1 is master, 0x2 is worker, 0x4 is etcd
	// 0x3 is master and worker
	// 0x7 is master, worker and etcd
	Type uint16 `json:"type"`

	Labels map[string]string `json:"labels"`
}

type Sans struct {
	DNSNames []string `json:"dns-names"`
	IPs      []string `json:"ips"`
}
type ApiServer struct {
	CertSans  Sans              `json:"cert-sans,omitempty"`
	Timeout   string            `json:"timeout,omitempty"`
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type ControlManager struct {
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type Scheduler struct {
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type WorkerConfig struct {
	KubeletConf         *Kubelet         `json:"kubeletconf,omitempty"`
	ProxyConf           *KubeProxy       `json:"kubeproxyconf,omitempty"`
	ContainerEngineConf *ContainerEngine `json:"containerengineconf,omitempty"`
}

type Kubelet struct {
	DnsVip        string            `json:"dns-vip,omitempty"`
	DnsDomain     string            `json:"dns-domain"`
	PauseImage    string            `json:"pause-image"`
	NetworkPlugin string            `json:"network-plugin"`
	CniBinDir     string            `json:"cni-bin-dir"`
	ExtraArgs     map[string]string `json:"extra-args,omitempty"`
}

type KubeProxy struct {
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type ContainerEngine struct {
	Runtime            string   `json:"runtime"`
	RuntimeEndpoint    string   `json:"runtime-endpoint"`
	RegistryMirrors    []string `json:"registry-mirrors"`
	InsecureRegistries []string `json:"insecure-registries"`
}

type APIEndpoint struct {
	AdvertiseAddress string `json:"advertise-address,omitempty"`
	BindPort         int32  `json:"bind-port,omitempty"`
}
type ControlPlaneConfig struct {
	Endpoint      string          `json:"endpoint,omitempty"`
	ApiConf       *ApiServer      `json:"apiconf,omitempty"`
	ManagerConf   *ControlManager `json:"managerconf,omitempty"`
	SchedulerConf *Scheduler      `json:"schedulerconf,omitempty"`
}

type CertificateConfig struct {
	SavePath       string `json:"savepath"` // default is "/etc/kubernetes/pki"
	ExternalCA     bool   `json:"external-ca"`
	ExternalCAPath string `json:"external-ca-path"`
}

type DnsConfig struct {
	CorednsType  string `json:"coredns-type"`
	ImageVersion string `json:"image-version"`
	Replicas     int    `json:"replicas"`
}

type ServiceClusterConfig struct {
	CIDR    string    `json:"cidr"`
	DNSAddr string    `json:"dns-address"`
	Gateway string    `json:"gateway"`
	DNS     DnsConfig `json:"dns"`
}

type PackageSrcConfig struct {
	Type     string `json:"type"`      // tar.gz...
	DistPath string `json:"dist-path"` // on dist node, untar path
	ArmSrc   string `json:"arm-srcpath"`
	X86Src   string `json:"x86-srcPath"`
}

func (p PackageSrcConfig) GetPkgDistPath() string {
	if p.DistPath == "" {
		return constants.DefaultPkgUntarPath
	}

	return p.DistPath
}

type EtcdClusterConfig struct {
	Token     string            `json:"token"`
	Nodes     []*HostConfig     `json:"nodes"`
	DataDir   string            `json:"data-dir"`
	CertsDir  string            `json:"certs-dir"` // local certs dir in machine running eggo, default /etc/kubernetes/pki
	External  bool              `json:"external"`  // if use external, eggo will ignore etcd deploy and cleanup
	ExtraArgs map[string]string `json:"extra-args"`
	// TODO: add loadbalance configuration
}

type NetworkConfig struct {
	PodCIDR    string            `json:"pod-cidr"`
	Plugin     string            `json:"plugin"`
	PluginArgs map[string]string `json:"plugin-args"`
}

type BootstrapTokenConfig struct {
	Description     string         `json:"description"`
	ID              string         `json:"ID"`
	Secret          string         `json:"secret"`
	TTL             *time.Duration `json:"ttl"`
	Usages          []string       `json:"usages"`
	AuthExtraGroups []string       `json:"auth_extra_groups"`
}

type ClusterRoleBindingConfig struct {
	Name        string `json:"Name"`
	SubjectName string `json:"SubjectName"`
	SubjectKind string `json:"SubjectKind"`
	RoleName    string `json:"RoleName"`
}

type LoadBalancer struct {
	IP   string `json:"ip"`
	Port string `json:"port"`
}

type AddonConfig struct {
	Type     string `json:"type"`
	Filename string `json:"filename"`
}

type ClusterConfig struct {
	Name            string                  `json:"name"`
	DeployDriver    string                  `json:"deploy-driver"` // default is binary
	ConfigDir       string                  `json:"config-dir"`    // default "/etc/kubernetes"
	Certificate     CertificateConfig       `json:"certificate,omitempty"`
	ServiceCluster  ServiceClusterConfig    `json:"servicecluster,omitempty"`
	Network         NetworkConfig           `json:"network,omitempty"`
	LocalEndpoint   APIEndpoint             `json:"local-endpoint,omitempty"`
	ControlPlane    ControlPlaneConfig      `json:"controlplane,omitempty"`
	PackageSrc      PackageSrcConfig        `json:"packagesource,omitempty"`
	EtcdCluster     EtcdClusterConfig       `json:"etcdcluster,omitempty"`
	Nodes           []*HostConfig           `json:"nodes,omitempty"`
	BootStrapTokens []*BootstrapTokenConfig `json:"bootstrap-tokens"`
	LoadBalancer    LoadBalancer            `json:"loadBalancer"`
	WorkerConfig    WorkerConfig            `json:"workerconfig"`
	Addons          []*AddonConfig          `json:"addons"`

	// TODO: add other configurations at here
}

type ClusterStatus struct {
}

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

type ClusterDeploymentAPI interface {
	PrepareInfrastructure() error
	DeployEtcdCluster() error
	DeployLoadBalancer() error
	InitControlPlane() error
	JoinBootstrap() error
	UpgradeCluster() error
	CleanupCluster() error
	ClusterStatus() (*ClusterStatus, error)
	PrepareNetwork() error
	ApplyAddons() error
	Finish()
}
