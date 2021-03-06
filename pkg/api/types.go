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
	"time"
)

const (
	Master      = 0x1
	Worker      = 0x2
	ETCD        = 0x4
	LoadBalance = 0x8
)

type ScheduleType string

const (
	SchedulePreJoin     ScheduleType = "prejoin"
	SchedulePostJoin    ScheduleType = "postjoin"
	SchedulePreCleanup  ScheduleType = "precleanup"
	SchedulePostCleanup ScheduleType = "postcleanup"
)

type HookOperator string

const (
	HookOpDeploy  HookOperator = "deploy"
	HookOpCleanup HookOperator = "cleanup"
	HookOpJoin    HookOperator = "join"
	HookOpDelete  HookOperator = "delete"
)

type HookType string

const (
	ClusterPrehookType  HookType = "cluster-prehook"
	ClusterPosthookType HookType = "cluster-posthook"
	PreHookType         HookType = "prehook"
	PostHookType        HookType = "posthook"
)

type HookRunConfig struct {
	ClusterID          string
	ClusterAPIEndpoint string
	ClusterConfigDir   string

	HookType HookType
	Operator HookOperator

	Node      *HostConfig
	Scheduler ScheduleType

	HookDir string
	Hooks   []*PackageConfig
}

type RoleInfra struct {
	OpenPorts []*OpenPorts     `json:"open-ports"`
	Softwares []*PackageConfig `json:"softwares"`
}

type OpenPorts struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp/udp
}

type PackageConfig struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"` // repo bin file dir image yaml shell
	Dst      string       `json:"dst,omitempty"`
	Schedule ScheduleType `json:"schedule,omitempty"`
	TimeOut  string       `json:"timeout,omitempty"`
}

type PackageSrcConfig struct {
	Type    string            `json:"type"`     // tar.gz...
	DstPath string            `json:"dst-path"` // untar path on dst node
	SrcPath map[string]string `json:"srcpath"`  // key: arm/amd/risc-v...
}

type HostConfig struct {
	Arch           string   `json:"arch"`
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Port           int      `json:"port"`
	ExtraIPs       []string `json:"extra-ips"`
	UserName       string   `json:"username"`
	Password       string   `json:"password"`
	PrivateKey     string   `json:"private-key"`
	PrivateKeyPath string   `json:"private-key-path"`

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
type APIServer struct {
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
	DNSVip        string            `json:"dns-vip,omitempty"`
	DNSDomain     string            `json:"dns-domain"`
	PauseImage    string            `json:"pause-image"`
	NetworkPlugin string            `json:"network-plugin"`
	CniBinDir     string            `json:"cni-bin-dir"`
	CniConfDir    string            `json:"cni-conf-dir"`
	EnableServer  bool              `json:"enable-server"`
	ExtraArgs     map[string]string `json:"extra-args,omitempty"`
}

type KubeProxy struct {
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type ContainerEngine struct {
	Runtime            string            `json:"runtime"`
	RuntimeEndpoint    string            `json:"runtime-endpoint"`
	RegistryMirrors    []string          `json:"registry-mirrors"`
	InsecureRegistries []string          `json:"insecure-registries"`
	ExtraArgs          map[string]string `json:"extra-args"`
}

type APIEndpoint struct {
	AdvertiseAddress string `json:"advertise-address,omitempty"`
	BindPort         int32  `json:"bind-port,omitempty"`
}
type ControlPlaneConfig struct {
	APIConf       *APIServer      `json:"apiconf,omitempty"`
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

type ClusterRoleConfig struct {
	Name      string   `json:"Name"`
	APIGroups []string `json:"APIGroups"`
	Resources []string `json:"Resources"`
	Verbs     []string `json:"Verbs"`
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

type ClusterHookConf struct {
	Type       HookType
	Operator   HookOperator
	Target     uint16
	HookSrcDir string
	HookFiles  []string
}

type ClusterConfig struct {
	Name            string                  `json:"name"`
	DeployDriver    string                  `json:"deploy-driver"` // default is binary
	ConfigDir       string                  `json:"config-dir"`    // default "/etc/kubernetes"
	Certificate     CertificateConfig       `json:"certificate,omitempty"`
	ServiceCluster  ServiceClusterConfig    `json:"servicecluster,omitempty"`
	Network         NetworkConfig           `json:"network,omitempty"`
	APIEndpoint     APIEndpoint             `json:"api-endpoint,omitempty"`
	ControlPlane    ControlPlaneConfig      `json:"controlplane,omitempty"`
	PackageSrc      PackageSrcConfig        `json:"packagesource,omitempty"`
	EtcdCluster     EtcdClusterConfig       `json:"etcdcluster,omitempty"`
	Nodes           []*HostConfig           `json:"nodes,omitempty"`
	BootStrapTokens []*BootstrapTokenConfig `json:"bootstrap-tokens"`
	LoadBalancer    LoadBalancer            `json:"loadBalancer"`
	WorkerConfig    WorkerConfig            `json:"workerconfig"`
	RoleInfra       map[uint16]*RoleInfra   `json:"role-infra"`

	// do not encode hooks, just set before use it
	HooksConf []*ClusterHookConf `json:"-"`

	// TODO: add other configurations at here
}

type ClusterStatus struct {
	Message       string          `json:"message"`
	ControlPlane  string          `json:"controlplane"`
	Working       bool            `json:"working"`
	StatusOfNodes map[string]bool `json:"statusOfNodes"`
	SuccessCnt    uint32          `json:"successCnt"`
	FailureCnt    uint32          `json:"failureCnt"`
}

type InfrastructureAPI interface {
	// TODO: should add other dependence cluster configurations
	MachineInfraSetup(machine *HostConfig) error
	MachineInfraDestroy(machine *HostConfig) error
}

type EtcdAPI interface {
	// TODO: should add other dependence cluster configurations
	EtcdClusterSetup() error
	EtcdClusterDestroy() error
	EtcdNodeSetup(machine *HostConfig) error
	EtcdNodeDestroy(machine *HostConfig) error
}

type ClusterManagerAPI interface {
	// TODO: should add other dependence cluster configurations
	PreCreateClusterHooks() error
	PostCreateClusterHooks(nodes []*HostConfig) error
	PreDeleteClusterHooks()
	PostDeleteClusterHooks()

	PreNodeJoinHooks(node *HostConfig) error
	PostNodeJoinHooks(node *HostConfig) error
	PreNodeCleanupHooks(node *HostConfig)
	PostNodeCleanupHooks(node *HostConfig)

	ClusterControlPlaneInit(node *HostConfig) error
	ClusterNodeJoin(node *HostConfig) error
	ClusterNodeCleanup(node *HostConfig, delType uint16) error
	ClusterUpgrade() error
	ClusterStatus() (*ClusterStatus, error)
	AddonsSetup() error
	AddonsDestroy() error

	CleanupLastStep(nodeName string) error
}

type LoadBalancerAPI interface {
	LoadBalancerSetup(lb *HostConfig) error
	LoadBalancerUpdate(lb *HostConfig) error
	LoadBalancerDestroy(lb *HostConfig) error
}

type ClusterDeploymentAPI interface {
	InfrastructureAPI
	EtcdAPI
	ClusterManagerAPI
	LoadBalancerAPI

	Finish()
}
