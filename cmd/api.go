package cmd

type ConfigExtraArgs struct {
	Name      string            `yaml:"name"`
	ExtraArgs map[string]string `yaml:"extra-args"`
}

type PackageSrcConfig struct {
	Type    string            `yaml:"type"`    // tar.gz...
	DstPath string            `yaml:"dstpath"` // untar path on dst node
	SrcPath map[string]string `yaml:"srcpath"` // key: arm/amd/risc-v
}

type PackageConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"` // repo bin file dir image yaml shell
	Dst      string `yaml:"dst,omitempty"`
	Schedule string `yaml:"schedule,omitempty"`
	TimeOut  string `yaml:"timeout,omitempty"`
}

type InstallConfig struct {
	PackageSrc       *PackageSrcConfig           `yaml:"package-source"`
	KubernetesMaster []*PackageConfig            `yaml:"kubernetes-master"`
	KubernetesWorker []*PackageConfig            `yaml:"kubernetes-worker"`
	Network          []*PackageConfig            `yaml:"network"`
	ETCD             []*PackageConfig            `yaml:"etcd"`
	LoadBalance      []*PackageConfig            `yaml:"loadbalance"`
	Container        []*PackageConfig            `yaml:"container"`
	Image            []*PackageConfig            `yaml:"image"`
	Addition         map[string][]*PackageConfig `yaml:"addition"` // key: master, worker, etcd, loadbalance
}

type HostConfig struct {
	Name string `yaml:"name"`
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
	Arch string `yaml:"arch"` // amd64, aarch64, default amd64
}

type LoadBalance struct {
	Name     string `yaml:"name"`
	Ip       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	Arch     string `yaml:"arch"` // amd64, aarch64, default amd64
	BindPort int    `yaml:"bind-port"`
}

type DnsConfig struct {
	CorednsType  string `yaml:"corednstype"`
	ImageVersion string `yaml:"imageversion"`
	Replicas     int    `yaml:"replicas"`
}

type ServiceClusterConfig struct {
	CIDR    string    `json:"cidr"`
	DNSAddr string    `json:"dnsaddress"`
	Gateway string    `json:"gateway"`
	DNS     DnsConfig `json:"dns"`
}

type NetworkConfig struct {
	PodCIDR    string            `yaml:"podcidr"`
	Plugin     string            `yaml:"plugin"`
	PluginArgs map[string]string `yaml:"pluginargs"`
}

type Sans struct {
	DNSNames []string `yaml:"dnsnames"`
	IPs      []string `yaml:"ips"`
}

type OpenPorts struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"` // tcp/udp
}

type DeployConfig struct {
	ClusterID            string                  `yaml:"cluster-id"`
	Username             string                  `yaml:"username"`
	Password             string                  `yaml:"password"`
	PrivateKeyPath       string                  `yaml:"private-key-path"`
	Masters              []*HostConfig           `yaml:"masters"`
	Workers              []*HostConfig           `yaml:"workers"`
	Etcds                []*HostConfig           `yaml:"etcds"`
	LoadBalance          LoadBalance             `yaml:"loadbalance"`
	ExternalCA           bool                    `yaml:"external-ca"`
	ExternalCAPath       string                  `yaml:"external-ca-path"`
	Service              ServiceClusterConfig    `yaml:"service"`
	NetWork              NetworkConfig           `yaml:"network"`
	ApiServerEndpoint    string                  `yaml:"apiserver-endpoint"`
	ApiServerCertSans    Sans                    `yaml:"apiserver-cert-sans"`
	ApiServerTimeout     string                  `yaml:"apiserver-timeout"`
	EtcdExternal         bool                    `yaml:"etcd-external"`
	EtcdToken            string                  `yaml:"etcd-token"`
	DnsVip               string                  `yaml:"dns-vip"`
	DnsDomain            string                  `yaml:"dns-domain"`
	PauseImage           string                  `yaml:"pause-image"`
	NetworkPlugin        string                  `yaml:"network-plugin"`
	EnableKubeletServing bool                    `yaml:"enable-kubelet-serving"`
	CniBinDir            string                  `yaml:"cni-bin-dir"`
	Runtime              string                  `yaml:"runtime"`
	RuntimeEndpoint      string                  `yaml:"runtime-endpoint"`
	RegistryMirrors      []string                `yaml:"registry-mirrors"`
	InsecureRegistries   []string                `yaml:"insecure-registries"`
	ConfigExtraArgs      []*ConfigExtraArgs      `yaml:"config-extra-args"`
	OpenPorts            map[string][]*OpenPorts `yaml:"open-ports"` // key: master, worker, etcd, loadbalance
	InstallConfig        InstallConfig           `yaml:"install"`
}
