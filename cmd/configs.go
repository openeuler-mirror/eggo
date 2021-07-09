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
 * Create: 2021-05-29
 * Description: eggo command configs implement
 ******************************************************************************/

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v1"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/infra"
)

const (
	MasterRole      string = "master"
	WorkerRole      string = "worker"
	ETCDRole        string = "etcd"
	LoadBalanceRole string = "loadbalance"
)

var (
	toTypeInt = map[string]uint16{
		MasterRole:      api.Master,
		WorkerRole:      api.Worker,
		ETCDRole:        api.ETCD,
		LoadBalanceRole: api.LoadBalance,
	}
)

type ConfigExtraArgs struct {
	Name      string            `yaml:"name"`
	ExtraArgs map[string]string `yaml:"extra-args"`
}

type InstallConfig struct {
	PackageSrc       *api.PackageSrcConfig           `yaml:"package-source"`
	KubernetesMaster []*api.PackageConfig            `yaml:"kubernetes-master"`
	KubernetesWorker []*api.PackageConfig            `yaml:"kubernetes-worker"`
	Network          []*api.PackageConfig            `yaml:"network"`
	ETCD             []*api.PackageConfig            `yaml:"etcd"`
	LoadBalance      []*api.PackageConfig            `yaml:"loadbalance"`
	Container        []*api.PackageConfig            `yaml:"container"`
	Image            []*api.PackageConfig            `yaml:"image"`
	Addition         map[string][]*api.PackageConfig `yaml:"addition"` // key: master, worker, etcd, loadbalance
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

type deployConfig struct {
	ClusterID          string                      `yaml:"cluster-id"`
	Username           string                      `yaml:"username"`
	Password           string                      `yaml:"password"`
	PrivateKeyPath     string                      `yaml:"private-key-path"`
	Masters            []*HostConfig               `yaml:"masters"`
	Workers            []*HostConfig               `yaml:"workers"`
	Etcds              []*HostConfig               `yaml:"etcds"`
	LoadBalance        LoadBalance                 `yaml:"loadbalance"`
	ExternalCA         bool                        `yaml:"external-ca"`
	ExternalCAPath     string                      `yaml:"external-ca-path"`
	Service            api.ServiceClusterConfig    `yaml:"service"`
	NetWork            api.NetworkConfig           `yaml:"network"`
	ApiServerEndpoint  string                      `yaml:"apiserver-endpoint"`
	ApiServerCertSans  api.Sans                    `yaml:"apiserver-cert-sans"`
	ApiServerTimeout   string                      `yaml:"apiserver-timeout"`
	EtcdExternal       bool                        `yaml:"etcd-external"`
	EtcdToken          string                      `yaml:"etcd-token"`
	DnsVip             string                      `yaml:"dns-vip"`
	DnsDomain          string                      `yaml:"dns-domain"`
	PauseImage         string                      `yaml:"pause-image"`
	NetworkPlugin      string                      `yaml:"network-plugin"`
	CniBinDir          string                      `yaml:"cni-bin-dir"`
	Runtime            string                      `yaml:"runtime"`
	RuntimeEndpoint    string                      `yaml:"runtime-endpoint"`
	RegistryMirrors    []string                    `yaml:"registry-mirrors"`
	InsecureRegistries []string                    `yaml:"insecure-registries"`
	ConfigExtraArgs    []*ConfigExtraArgs          `yaml:"config-extra-args"`
	Addons             []*api.AddonConfig          `yaml:"addons"`
	OpenPorts          map[string][]*api.OpenPorts `yaml:"open-ports"` // key: master, worker, etcd, loadbalance
	InstallConfig      InstallConfig               `yaml:"install"`
}

func init() {
	if _, err := os.Stat(utils.GetEggoDir()); err == nil {
		return
	}

	if err := os.Mkdir(utils.GetEggoDir(), 0700); err != nil {
		logrus.Errorf("mkdir eggo directory %v failed", utils.GetEggoDir())
	}
}

func defaultDeployConfigPath() string {
	return filepath.Join(utils.GetEggoDir(), "deploy.yaml")
}

func backupedDeployConfigPath() string {
	return filepath.Join(utils.GetEggoDir(), "deploy_backup.yaml")
}

func loadDeployConfig(file string) (*deployConfig, error) {
	yamlStr, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	conf := &deployConfig{}
	if err := yaml.Unmarshal([]byte(yamlStr), conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func getDefaultClusterdeploymentConfig() *api.ClusterConfig {
	return &api.ClusterConfig{
		Name:      "k8s-cluster",
		ConfigDir: constants.DefaultK8SRootDir,
		Certificate: api.CertificateConfig{
			SavePath: constants.DefaultK8SCertDir,
		},
		ServiceCluster: api.ServiceClusterConfig{
			CIDR:    "10.32.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.32.0.1",
		},
		Network: api.NetworkConfig{
			PodCIDR:    "10.244.0.0/16",
			PluginArgs: make(map[string]string),
		},
		ControlPlane: api.ControlPlaneConfig{
			ApiConf: &api.ApiServer{
				Timeout: "120s",
			},
		},
		WorkerConfig: api.WorkerConfig{
			KubeletConf: &api.Kubelet{
				DnsVip:        "10.32.0.10",
				DnsDomain:     "cluster.local",
				PauseImage:    "k8s.gcr.io/pause:3.2",
				NetworkPlugin: "cni",
				CniBinDir:     "/usr/libexec/cni,/opt/cni/bin",
			},
			ContainerEngineConf: &api.ContainerEngine{
				RegistryMirrors:    []string{},
				InsecureRegistries: []string{},
			},
		},
		PackageSrc: api.PackageSrcConfig{},
		EtcdCluster: api.EtcdClusterConfig{
			Token:    "etcd-cluster",
			DataDir:  "/var/lib/etcd/default.etcd",
			CertsDir: constants.DefaultK8SCertDir,
			External: false,
		},
		DeployDriver: "binary",
		RoleInfra:    infra.RegisterInfra(),
	}
}

func getDefaultPrivateKeyPath() string {
	return filepath.Join(utils.GetSysHome(), ".ssh", "id_rsa")
}

func createCommonHostConfig(userHostconfig *HostConfig, defaultName string, username string,
	password string, userPrivateKeyPath string) *api.HostConfig {
	arch, name, port, privateKeyPath := "amd64", defaultName, 22, getDefaultPrivateKeyPath()
	if userHostconfig.Arch != "" {
		arch = userHostconfig.Arch
	}
	if userHostconfig.Name != "" {
		name = userHostconfig.Name
	}
	if userPrivateKeyPath != "" {
		privateKeyPath = userPrivateKeyPath
	}
	// If private key path does not exist, ignore it
	if _, err := os.Stat(privateKeyPath); err != nil {
		privateKeyPath = ""
	}
	if userHostconfig.Port != 0 {
		port = userHostconfig.Port
	}

	hostconfig := &api.HostConfig{
		Arch:           arch,
		Name:           name,
		Address:        userHostconfig.Ip,
		Port:           port,
		UserName:       username,
		Password:       password,
		PrivateKeyPath: privateKeyPath,
	}

	return hostconfig
}

func appendSoftware(software, packageConfig, defaultPackage []*api.PackageConfig) []*api.PackageConfig {
	var packages []*api.PackageConfig
	if len(packageConfig) != 0 {
		packages = packageConfig
	} else {
		packages = defaultPackage
	}

	result := software
	for _, p := range packages {
		splitSoftware := strings.Split(p.Name, ",")
		for _, s := range splitSoftware {
			result = append(result, &api.PackageConfig{
				Name: s,
				Type: p.Type,
				Dst:  p.Dst,
			})
		}
	}

	return result
}

func fillPackageConfig(ccfg *api.ClusterConfig, icfg *InstallConfig) {
	if icfg.PackageSrc != nil {
		setIfStrConfigNotEmpty(&ccfg.PackageSrc.Type, icfg.PackageSrc.Type)
		setIfStrConfigNotEmpty(&ccfg.PackageSrc.ArmSrc, icfg.PackageSrc.ArmSrc)
		setIfStrConfigNotEmpty(&ccfg.PackageSrc.X86Src, icfg.PackageSrc.X86Src)
	}

	software := []struct {
		pc   []*api.PackageConfig
		role uint16
		dpc  []*api.PackageConfig
	}{
		{icfg.LoadBalance, api.LoadBalance, infra.LoadbalancePackages},
		{icfg.Container, api.Worker, infra.ContainerPackages},
		{icfg.Image, api.Worker, []*api.PackageConfig{}},
		{icfg.Network, api.Worker, infra.NetworkPackages},
		{icfg.ETCD, api.ETCD, infra.EtcdPackages},
		{icfg.KubernetesMaster, api.Master, infra.MasterPackages},
		{icfg.KubernetesWorker, api.Worker, infra.WorkerPackages},
	}

	for _, s := range software {
		ccfg.RoleInfra[s.role].Softwares = appendSoftware(ccfg.RoleInfra[s.role].Softwares, s.pc, s.dpc)
	}

	if len(icfg.Addition) == 0 {
		return
	}

	for t, p := range icfg.Addition {
		role, ok := toTypeInt[t]
		if !ok {
			logrus.Warnf("invalid role %s", t)
			continue
		}

		ccfg.RoleInfra[role].Softwares = appendSoftware(ccfg.RoleInfra[role].Softwares, p, []*api.PackageConfig{})
	}
}

func fillOpenPort(ccfg *api.ClusterConfig, openports map[string][]*api.OpenPorts, dnsType string) {
	// key: master, worker, etcd, loadbalance
	for t, p := range openports {
		role, ok := toTypeInt[t]
		if !ok {
			logrus.Warnf("invalid role %s", t)
			continue
		}

		ccfg.RoleInfra[role].OpenPorts = append(ccfg.RoleInfra[role].OpenPorts, p...)
	}

	if dnsType == "binary" || dnsType == "" {
		ccfg.RoleInfra[api.Master].OpenPorts =
			append(ccfg.RoleInfra[api.Master].OpenPorts, infra.CorednsPorts...)
	} else if dnsType == "pod" {
		ccfg.RoleInfra[api.Worker].OpenPorts =
			append(ccfg.RoleInfra[api.Worker].OpenPorts, infra.CorednsPorts...)
	}
}

func defaultHostName(clusterID string, nodeType string, i int) string {
	return fmt.Sprintf("%s-%s-%s", clusterID, nodeType, strconv.Itoa(i))
}

func appendNodeNoDup(hostconfigs []*HostConfig, hostconfig *HostConfig) []*HostConfig {
	for _, h := range hostconfigs {
		if h.Ip == hostconfig.Ip {
			return hostconfigs
		}
	}
	return append(hostconfigs, hostconfig)
}

func deleteNodebyDelName(hostconfigs []*HostConfig, delName string) ([]*HostConfig, error) {
	for i, h := range hostconfigs {
		if h.Ip == delName || h.Name == delName {
			result := append(hostconfigs[:i], hostconfigs[i+1:]...)
			if len(result) == 0 {
				return nil, nil
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("delete node from config failed: %v not found", delName)
}

func deleteConfig(userConfig *deployConfig, delType string, delName string) error {
	var err error
	types := strings.Split(delType, ",")
	for _, nodeType := range types {
		if nodeType == MasterRole {
			userConfig.Masters, err = deleteNodebyDelName(userConfig.Masters, delName)
		}
		if nodeType == WorkerRole {
			userConfig.Workers, err = deleteNodebyDelName(userConfig.Workers, delName)
		}
		if nodeType == ETCDRole {
			userConfig.Etcds, err = deleteNodebyDelName(userConfig.Etcds, delName)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func joinConfig(userConfig *deployConfig, joinType string, joinHost *HostConfig) *api.HostConfig {
	masters := userConfig.Masters
	userConfig.Masters = nil
	workers := userConfig.Workers
	userConfig.Workers = nil
	etcds := userConfig.Etcds
	userConfig.Etcds = nil
	defer func() {
		userConfig.Masters = masters
		userConfig.Workers = workers
		userConfig.Etcds = etcds
	}()

	types := strings.Split(joinType, ",")
	for _, nodeType := range types {
		var hostconfig HostConfig
		var index int

		if nodeType == MasterRole {
			index = len(masters)
		}
		if nodeType == WorkerRole {
			index = len(workers)
		}
		if nodeType == ETCDRole {
			index = len(etcds)
		}

		if joinHost.Name != "" {
			hostconfig.Name = joinHost.Name
		} else {
			hostconfig.Name = defaultHostName(userConfig.ClusterID, nodeType, index)
		}
		hostconfig.Ip = joinHost.Ip
		setIfStrConfigNotEmpty(&hostconfig.Arch, joinHost.Arch)
		if joinHost.Port != 0 {
			hostconfig.Port = joinHost.Port
		}

		if nodeType == MasterRole {
			userConfig.Masters = []*HostConfig{&hostconfig}
			masters = appendNodeNoDup(masters, &hostconfig)
		}
		if nodeType == WorkerRole {
			userConfig.Workers = []*HostConfig{&hostconfig}
			workers = appendNodeNoDup(workers, &hostconfig)
		}
		if nodeType == ETCDRole {
			userConfig.Etcds = []*HostConfig{&hostconfig}
			etcds = appendNodeNoDup(etcds, &hostconfig)
		}
	}

	config := toClusterdeploymentConfig(userConfig)
	for _, h := range config.Nodes {
		if h.Address == joinHost.Ip {
			return h
		}
	}
	return nil
}

func fillHostConfig(ccfg *api.ClusterConfig, conf *deployConfig) {
	var hostconfig *api.HostConfig
	cache := make(map[string]int)
	var nodes []*api.HostConfig

	for i, master := range conf.Masters {
		hostconfig = createCommonHostConfig(master, conf.ClusterID+"-master-"+strconv.Itoa(i),
			conf.Username, conf.Password, conf.PrivateKeyPath)
		hostconfig.Type |= api.Master
		idx, ok := cache[hostconfig.Address]
		if ok {
			nodes[idx] = hostconfig
			continue
		}
		cache[hostconfig.Address] = len(nodes)
		nodes = append(nodes, hostconfig)
	}

	for i, worker := range conf.Workers {
		idx, exist := cache[worker.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(worker, conf.ClusterID+"-worker-"+strconv.Itoa(i),
				conf.Username, conf.Password, conf.PrivateKeyPath)
		} else {
			hostconfig = nodes[idx]
		}
		hostconfig.Type |= api.Worker
		if exist {
			nodes[idx] = hostconfig
			continue
		}
		cache[hostconfig.Address] = len(nodes)
		nodes = append(nodes, hostconfig)
	}

	// if no etcd configed, default to install to master
	var etcds []*HostConfig
	if len(conf.Etcds) == 0 {
		etcds = conf.Masters
	} else {
		etcds = conf.Etcds
	}
	for i, etcd := range etcds {
		idx, exist := cache[etcd.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(etcd, conf.ClusterID+"-etcd-"+strconv.Itoa(i),
				conf.Username, conf.Password, conf.PrivateKeyPath)
		} else {
			hostconfig = nodes[idx]
		}
		hostconfig.Type |= api.ETCD
		if exist {
			nodes[idx] = hostconfig
			continue
		}
		cache[hostconfig.Address] = len(nodes)
		nodes = append(nodes, hostconfig)
	}

	if conf.LoadBalance.Ip != "" {
		idx, exist := cache[conf.LoadBalance.Ip]
		if !exist {
			config := &HostConfig{
				Name: conf.LoadBalance.Name,
				Ip:   conf.LoadBalance.Ip,
				Port: conf.LoadBalance.Port,
				Arch: conf.LoadBalance.Arch,
			}
			hostconfig = createCommonHostConfig(config, conf.ClusterID+"-loadbalance", conf.Username,
				conf.Password, conf.PrivateKeyPath)
		} else {
			hostconfig = nodes[idx]
		}
		hostconfig.Type |= api.LoadBalance

		if exist {
			nodes[idx] = hostconfig
		} else {
			cache[hostconfig.Address] = len(nodes)
			nodes = append(nodes, hostconfig)
		}
	}

	ccfg.Nodes = append(ccfg.Nodes, nodes...)
}

func setIfStrConfigNotEmpty(config *string, userConfig string) {
	if config == nil {
		logrus.Errorf("invalid nil config")
		return
	}
	if userConfig != "" {
		*config = userConfig
	}
}

func setStrStrMap(config map[string]string, userConfig map[string]string) {
	if config == nil {
		logrus.Errorf("invalid nil config")
		return
	}
	for k, v := range userConfig {
		config[k] = v
	}
}

func notInStrArray(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return false
		}
	}
	return true
}

func setStrArray(config *[]string, userConfig []string) {
	for _, v := range userConfig {
		if notInStrArray(*config, v) {
			*config = append(*config, v)
		}
	}
}

func fillLoadBalance(LoadBalancer *api.LoadBalancer, lb LoadBalance) {
	if lb.Ip == "" || lb.BindPort <= 0 {
		return
	}

	setIfStrConfigNotEmpty(&LoadBalancer.IP, lb.Ip)
	setIfStrConfigNotEmpty(&LoadBalancer.Port, strconv.Itoa(lb.BindPort))
}

func fillAPIEndPoint(APIEndpoint *api.APIEndpoint, conf *deployConfig) {
	host, port := "", ""
	if conf.ApiServerEndpoint != "" {
		var err error
		host, port, err = net.SplitHostPort(conf.ApiServerEndpoint)
		if err != nil {
			logrus.Errorf("invalid api endpoint %s: %v", conf.ApiServerEndpoint, err)
		}
	}

	if host == "" || port == "" {
		host, port = conf.LoadBalance.Ip, strconv.Itoa(conf.LoadBalance.Port)
	}
	if (host == "" || port == "") && len(conf.Masters) != 0 {
		host = conf.Masters[0].Ip
		port = "6443"
	}

	if host == "" || port == "" {
		return
	}

	iport, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		logrus.Errorf("invalid port %s: %v", port, err)
		return
	}

	APIEndpoint.AdvertiseAddress = host
	APIEndpoint.BindPort = int32(iport)
}

func toClusterdeploymentConfig(conf *deployConfig) *api.ClusterConfig {
	ccfg := getDefaultClusterdeploymentConfig()

	setIfStrConfigNotEmpty(&ccfg.Name, conf.ClusterID)
	fillHostConfig(ccfg, conf)
	ccfg.Certificate.ExternalCA = conf.ExternalCA
	setIfStrConfigNotEmpty(&ccfg.Certificate.ExternalCAPath, conf.ExternalCAPath)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.CIDR, conf.Service.CIDR)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.DNSAddr, conf.Service.DNSAddr)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.Gateway, conf.Service.Gateway)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.DNS.CorednsType, conf.Service.DNS.CorednsType)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.DNS.ImageVersion, conf.Service.DNS.ImageVersion)
	ccfg.ServiceCluster.DNS.Replicas = conf.Service.DNS.Replicas
	setIfStrConfigNotEmpty(&ccfg.Network.PodCIDR, conf.NetWork.PodCIDR)
	setIfStrConfigNotEmpty(&ccfg.Network.Plugin, conf.NetWork.Plugin)
	setStrStrMap(ccfg.Network.PluginArgs, conf.NetWork.PluginArgs)
	setStrArray(&ccfg.ControlPlane.ApiConf.CertSans.DNSNames, conf.ApiServerCertSans.DNSNames)
	setStrArray(&ccfg.ControlPlane.ApiConf.CertSans.IPs, conf.ApiServerCertSans.IPs)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.ApiConf.Timeout, conf.ApiServerTimeout)
	ccfg.EtcdCluster.External = conf.EtcdExternal
	for _, node := range ccfg.Nodes {
		if (node.Type & api.ETCD) != 0 {
			ccfg.EtcdCluster.Nodes = append(ccfg.EtcdCluster.Nodes, node)
		}
	}
	setIfStrConfigNotEmpty(&ccfg.EtcdCluster.Token, conf.EtcdToken)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.KubeletConf.DnsVip, conf.DnsVip)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.KubeletConf.DnsDomain, conf.DnsDomain)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.KubeletConf.PauseImage, conf.PauseImage)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.KubeletConf.NetworkPlugin, conf.NetworkPlugin)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.KubeletConf.CniBinDir, conf.CniBinDir)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.ContainerEngineConf.Runtime, conf.Runtime)
	setIfStrConfigNotEmpty(&ccfg.WorkerConfig.ContainerEngineConf.RuntimeEndpoint, conf.RuntimeEndpoint)
	setStrArray(&ccfg.WorkerConfig.ContainerEngineConf.RegistryMirrors, conf.RegistryMirrors)
	setStrArray(&ccfg.WorkerConfig.ContainerEngineConf.InsecureRegistries, conf.InsecureRegistries)
	fillLoadBalance(&ccfg.LoadBalancer, conf.LoadBalance)
	fillAPIEndPoint(&ccfg.APIEndpoint, conf)
	fillPackageConfig(ccfg, &conf.InstallConfig)
	fillOpenPort(ccfg, conf.OpenPorts, conf.Service.DNS.CorednsType)

	ccfg.Addons = append(ccfg.Addons, conf.Addons...)

	return ccfg
}

func getHostconfigs(format string, ips []string) []*HostConfig {
	var confs []*HostConfig
	for i, ip := range ips {
		confs = append(confs, &HostConfig{
			Name: fmt.Sprintf(format, i),
			Ip:   ip,
			Port: 22,
			Arch: "amd64",
		})
	}
	return confs
}

func createDeployConfigTemplate(file string) error {
	var masters, workers, etcds []*HostConfig
	masterIP := []string{"192.168.0.2"}
	if opts.masters != nil {
		masterIP = opts.masters
	}
	workersIP := []string{"192.168.0.2", "192.168.0.3", "192.168.0.4"}
	if opts.nodes != nil {
		workersIP = opts.nodes
	}
	lbIP := "192.168.0.1"
	if opts.loadbalance != "" {
		lbIP = opts.loadbalance
	}
	etcdsIP := masterIP
	if opts.etcds != nil {
		etcdsIP = opts.etcds
	}
	masters = getHostconfigs("k8s-master-%d", masterIP)
	workers = getHostconfigs("k8s-worker-%d", workersIP)
	etcds = getHostconfigs("etcd-%d", etcdsIP)
	lb := LoadBalance{
		Name:     "k8s-loadbalance",
		Ip:       lbIP,
		Port:     22,
		Arch:     "amd64",
		BindPort: 8443,
	}

	if etcds == nil {
		etcds = masters
	}
	conf := &deployConfig{
		ClusterID:      opts.name,
		Username:       opts.username,
		Password:       opts.password,
		PrivateKeyPath: getDefaultPrivateKeyPath(),
		Masters:        masters,
		Workers:        workers,
		Etcds:          etcds,
		LoadBalance:    lb,
		ExternalCA:     false,
		ExternalCAPath: "/opt/externalca",
		Service: api.ServiceClusterConfig{
			CIDR:    "10.32.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.32.0.1",
			DNS: api.DnsConfig{
				CorednsType: "binary",
			},
		},
		NetWork: api.NetworkConfig{
			PodCIDR:    "10.244.0.0/16",
			Plugin:     "calico",
			PluginArgs: make(map[string]string),
		},
		ApiServerEndpoint: fmt.Sprintf("%s:%d", lb.Ip, lb.BindPort),
		ApiServerCertSans: api.Sans{},
		ApiServerTimeout:  "120s",
		EtcdExternal:      false,
		EtcdToken:         "etcd-cluster",
		DnsVip:            "10.32.0.10",
		DnsDomain:         "cluster.local",
		PauseImage:        "k8s.gcr.io/pause:3.2",
		NetworkPlugin:     "cni",
		CniBinDir:         "/usr/libexec/cni,/opt/cni/bin",
		Runtime:           "iSulad",
		RuntimeEndpoint:   "unix:///var/run/isulad.sock",
		OpenPorts: map[string][]*api.OpenPorts{
			"worker": {
				&api.OpenPorts{
					Port:     111,
					Protocol: "tcp",
				},
				&api.OpenPorts{
					Port:     179,
					Protocol: "tcp",
				},
			},
			"master": {
				&api.OpenPorts{
					Port:     53,
					Protocol: "tcp",
				},
				&api.OpenPorts{
					Port:     53,
					Protocol: "udp",
				},
				&api.OpenPorts{
					Port:     9153,
					Protocol: "tcp",
				},
			},
		},
		InstallConfig: InstallConfig{
			PackageSrc: &api.PackageSrcConfig{
				Type:   "tar.gz",
				ArmSrc: "/root/packages/pacakges-arm.tar.gz",
				X86Src: "/root/packages/packages-x86.tar.gz",
			},
			KubernetesMaster: []*api.PackageConfig{
				{
					Name: "kubernetes-client,kubernetes-master",
					Type: "pkg",
				},
			},
			KubernetesWorker: []*api.PackageConfig{
				{
					Name: "docker-engine,kubernetes-client,kubernetes-node,kubernetes-kubelet",
					Type: "pkg",
				},
				{
					Name: "conntrack-tools,socat",
					Type: "pkg",
				},
			},
			Container: []*api.PackageConfig{
				{
					Name: "emacs-filesystem,gflags,gpm-libs,re2,rsync,vim-filesystem,vim-common,vim-enhanced,zlib-devel",
					Type: "pkg",
				},
				{
					Name: "libwebsockets,protobuf,protobuf-devel,grpc,libcgroup",
					Type: "pkg",
				},
				{
					Name: "yajl,lxc,lxc-libs,lcr,clibcni,iSulad",
					Type: "pkg",
				},
			},
			Network: []*api.PackageConfig{
				{
					Name: "containernetworking-plugins",
					Type: "pkg",
				},
			},
			ETCD: []*api.PackageConfig{
				{
					Name: "etcd",
					Type: "pkg",
				},
			},
			LoadBalance: []*api.PackageConfig{
				{
					Name: "gd,gperftools-libs,libunwind,libwebp,libxslt",
					Type: "pkg",
				},
				{
					Name: "nginx,nginx-all-modules,nginx-filesystem,nginx-mod-http-image-filter,nginx-mod-http-perl,nginx-mod-http-xslt-filter,nginx-mod-mail,nginx-mod-stream",
					Type: "pkg",
				},
			},
			Image: []*api.PackageConfig{
				{
					Name: "pause.tar",
					Type: "image",
				},
			},
			Addition: map[string][]*api.PackageConfig{
				"master": {
					{
						Name: "calico.yaml",
						Type: "yaml",
					},
					{
						Name: "coredns",
						Type: "pkg",
					},
				},
				"worker": {
					{
						Name: "docker.service",
						Type: "file",
						Dst:  "/usr/lib/systemd/system/",
					},
				},
			},
		},
	}

	d, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("marshal template config failed: %v", err)
	}

	if err := ioutil.WriteFile(file, d, 0640); err != nil {
		return fmt.Errorf("write template config file failed: %v", err)
	}

	return nil
}

func backupDeployConfig(cc *deployConfig) error {
	d, err := yaml.Marshal(cc)
	if err != nil {
		return fmt.Errorf("marshal template config failed: %v", err)
	}

	if err = ioutil.WriteFile(backupedDeployConfigPath(), d, 0640); err != nil {
		return fmt.Errorf("write user deploy config file failed: %v", err)
	}

	return nil
}

func loadBackupedDeployConfig() (*deployConfig, error) {
	yamlStr, err := ioutil.ReadFile(backupedDeployConfigPath())
	if err != nil {
		return nil, err
	}

	conf := &deployConfig{}
	if err := yaml.Unmarshal([]byte(yamlStr), conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func loadConfig() (*deployConfig, error) {
	conf, err := loadBackupedDeployConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// try load default user's config if backup file not exist
			conf, err = loadDeployConfig(opts.deployConfig)
			if err != nil {
				return nil, fmt.Errorf("load default user's config failed: %v", err)
			}
		} else {
			return nil, fmt.Errorf("load backuped deploy config failed: %v", err)
		}
	}

	return conf, nil
}
