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

package cmd

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v1"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/coredns"
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

func ToEggoPackageConfig(pcs []*PackageConfig) []*api.PackageConfig {
	var res []*api.PackageConfig
	for _, pc := range pcs {
		res = append(res, &api.PackageConfig{
			Name:     pc.Name,
			Type:     pc.Type,
			Dst:      pc.Dst,
			Schedule: api.ScheduleType(pc.Schedule),
			TimeOut:  pc.TimeOut,
		})
	}
	return res
}

func ToEggoOpenPort(ports []*OpenPorts) []*api.OpenPorts {
	var res []*api.OpenPorts
	for _, pc := range ports {
		res = append(res, &api.OpenPorts{
			Port:     pc.Port,
			Protocol: pc.Protocol,
		})
	}
	return res
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

func eggoPlaceHolderPath(ClusterID string) string {
	return filepath.Join(api.GetEggoClusterPath(), ClusterID, ".eggo.pid")
}

func savedDeployConfigPath(ClusterID string) string {
	return filepath.Join(api.GetEggoClusterPath(), ClusterID, "deploy.yaml")
}

func saveDeployConfig(cc *DeployConfig, filePath string) error {
	d, err := yaml.Marshal(cc)
	if err != nil {
		return fmt.Errorf("marshal template config failed: %v", err)
	}

	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, api.GetEggoClusterPath()) {
		return fmt.Errorf("invalid config file path %v", filePath)
	}

	if err = os.MkdirAll(filepath.Dir(cleanPath), 0750); err != nil {
		return fmt.Errorf("create dir %v to save deploy config failed: %v", filepath.Dir(cleanPath), err)
	}

	if err = ioutil.WriteFile(filePath, d, 0640); err != nil {
		return fmt.Errorf("write user deploy config file failed: %v", err)
	}

	return nil
}

func fillEtcdsIfNotExist(cc *DeployConfig) {
	if len(cc.Etcds) != 0 {
		return
	}

	cc.Etcds = append(cc.Etcds, cc.Masters...)
}

func loadDeployConfig(file string) (*DeployConfig, error) {
	yamlStr, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	conf := &DeployConfig{}
	if err := yaml.Unmarshal([]byte(yamlStr), conf); err != nil {
		return nil, err
	}

	// default install etcds to masters if etcds not configed
	fillEtcdsIfNotExist(conf)

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
				EnableServer:  false,
			},
			ContainerEngineConf: &api.ContainerEngine{
				RegistryMirrors:    []string{},
				InsecureRegistries: []string{},
				ExtraArgs:          make(map[string]string),
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
				Name:     s,
				Type:     p.Type,
				Dst:      p.Dst,
				Schedule: p.Schedule,
				TimeOut:  p.TimeOut,
			})
		}
	}

	return result
}

func fillPackageConfig(ccfg *api.ClusterConfig, icfg *InstallConfig) {
	ccfg.PackageSrc.SrcPath = make(map[string]string)
	if icfg.PackageSrc != nil {
		setIfStrConfigNotEmpty(&ccfg.PackageSrc.Type, icfg.PackageSrc.Type)
		for arch, path := range icfg.PackageSrc.SrcPath {
			ccfg.PackageSrc.SrcPath[strings.ToLower(arch)] = path
		}
	}

	software := []struct {
		pc   []*api.PackageConfig
		role uint16
		dpc  []*api.PackageConfig
	}{
		{ToEggoPackageConfig(icfg.LoadBalance), api.LoadBalance, infra.LoadbalancePackages},
		{ToEggoPackageConfig(icfg.Container), api.Worker, infra.ContainerPackages},
		{ToEggoPackageConfig(icfg.Image), api.Worker, []*api.PackageConfig{}},
		{ToEggoPackageConfig(icfg.Network), api.Worker, infra.NetworkPackages},
		{ToEggoPackageConfig(icfg.ETCD), api.ETCD, infra.EtcdPackages},
		{ToEggoPackageConfig(icfg.KubernetesMaster), api.Master, infra.MasterPackages},
		{ToEggoPackageConfig(icfg.KubernetesWorker), api.Worker, infra.WorkerPackages},
	}

	for _, s := range software {
		ccfg.RoleInfra[s.role].Softwares = appendSoftware(ccfg.RoleInfra[s.role].Softwares, s.pc, s.dpc)
	}

	if coredns.IsTypeBinary(ccfg.ServiceCluster.DNS.CorednsType) {
		ccfg.RoleInfra[api.Master].Softwares = appendSoftware(ccfg.RoleInfra[api.Master].Softwares, ToEggoPackageConfig(icfg.Dns), infra.DnsPackages)
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

		ccfg.RoleInfra[role].Softwares = appendSoftware(ccfg.RoleInfra[role].Softwares, ToEggoPackageConfig(p), []*api.PackageConfig{})
	}
}

func fillOpenPort(ccfg *api.ClusterConfig, openports map[string][]*OpenPorts, dnsType string) {
	// key: master, worker, etcd, loadbalance
	for t, p := range openports {
		role, ok := toTypeInt[t]
		if !ok {
			logrus.Warnf("invalid role %s", t)
			continue
		}

		ccfg.RoleInfra[role].OpenPorts = append(ccfg.RoleInfra[role].OpenPorts, ToEggoOpenPort(p)...)
	}

	if coredns.IsTypeBinary(dnsType) {
		ccfg.RoleInfra[api.Master].OpenPorts =
			append(ccfg.RoleInfra[api.Master].OpenPorts, infra.CorednsPorts...)
	}
}

func defaultHostName(clusterID string, nodeType string, i int) string {
	return fmt.Sprintf("%s-%s-%s", clusterID, nodeType, strconv.Itoa(i))
}

func getHostConfigByIp(nodes []*HostConfig, ip string) *HostConfig {
	for _, node := range nodes {
		if node.Ip == ip {
			return node
		}
	}
	return nil
}

func getAllHostConfigs(conf *DeployConfig) []*HostConfig {
	allHostConfigs := append(conf.Masters, conf.Workers...)
	allHostConfigs = append(allHostConfigs, conf.Etcds...)
	allHostConfigs = append(allHostConfigs, &HostConfig{
		Name: conf.LoadBalance.Name,
		Ip:   conf.LoadBalance.Ip,
		Port: conf.LoadBalance.Port,
		Arch: conf.LoadBalance.Arch,
	})

	return allHostConfigs
}

func createHostConfig(host *HostConfig, joinHost *HostConfig, defaultName string) *HostConfig {
	var hostconfig HostConfig

	if host != nil {
		hostconfig.Name = host.Name
		hostconfig.Arch = host.Arch
		hostconfig.Port = host.Port
	} else {
		hostconfig.Name = defaultName
		if joinHost.Name != "" {
			hostconfig.Name = joinHost.Name
		}
		hostconfig.Arch = "amd64"
		if joinHost.Arch != "" {
			hostconfig.Arch = joinHost.Arch
		}
		hostconfig.Port = 22
		if joinHost.Port != 0 {
			hostconfig.Port = joinHost.Port
		}
	}
	hostconfig.Ip = joinHost.Ip

	return &hostconfig
}

func fillHostConfig(ccfg *api.ClusterConfig, conf *DeployConfig) {
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

	for i, etcd := range conf.Etcds {
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

func fillAPIEndPoint(APIEndpoint *api.APIEndpoint, conf *DeployConfig) {
	host, port := "", ""
	if conf.ApiServerEndpoint != "" {
		var err error
		host, port, err = net.SplitHostPort(conf.ApiServerEndpoint)
		if err != nil {
			logrus.Errorf("invalid api endpoint %s: %v", conf.ApiServerEndpoint, err)
		}
	}

	if host == "" || port == "" {
		host, port = conf.LoadBalance.Ip, strconv.Itoa(conf.LoadBalance.BindPort)
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

func fillExtrArgs(ccfg *api.ClusterConfig, eargs []*ConfigExtraArgs) {
	for _, ea := range eargs {
		switch ea.Name {
		case "etcd":
			api.WithEtcdExtrArgs(ea.ExtraArgs)(ccfg)
		case "kube-apiserver":
			api.WithAPIServerExtrArgs(ea.ExtraArgs)(ccfg)
		case "kube-controller-manager":
			api.WithControllerManagerExtrArgs(ea.ExtraArgs)(ccfg)
		case "kube-scheduler":
			api.WithSchedulerExtrArgs(ea.ExtraArgs)(ccfg)
		case "kube-proxy":
			api.WithKubeProxyExtrArgs(ea.ExtraArgs)(ccfg)
		case "kubelet":
			api.WithKubeletExtrArgs(ea.ExtraArgs)(ccfg)
		case "container-engine":
			api.WithContainerEngineExtrArgs(ea.ExtraArgs)(ccfg)
		default:
			logrus.Warnf("unknow extra args key: %s", ea.Name)
		}
	}
}

func toClusterdeploymentConfig(conf *DeployConfig, hooks []*api.ClusterHookConf) *api.ClusterConfig {
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
	ccfg.WorkerConfig.KubeletConf.EnableServer = conf.EnableKubeletServing

	fillExtrArgs(ccfg, conf.ConfigExtraArgs)
	ccfg.HooksConf = hooks

	return ccfg
}

func getClusterHookConf(op api.HookOperator) ([]*api.ClusterHookConf, error) {
	var hooks []*api.ClusterHookConf

	if opts.clusterPrehook != "" {
		hook, err := getCmdHooks(opts.clusterPrehook, api.ClusterPrehookType, op)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, hook)
	}

	if opts.clusterPosthook != "" {
		hook, err := getCmdHooks(opts.clusterPosthook, api.ClusterPosthookType, op)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, hook)
	}

	if opts.prehook != "" {
		hook, err := getCmdHooks(opts.prehook, api.PreHookType, op)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, hook)
	}

	if opts.posthook != "" {
		hook, err := getCmdHooks(opts.posthook, api.PostHookType, op)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, hook)
	}
	return hooks, nil
}

func getCmdHooks(hopts string, ty api.HookType, op api.HookOperator) (*api.ClusterHookConf, error) {
	path, target, err := getHookPathAndTarget(hopts)
	if err != nil {
		return nil, err
	}
	hook, err := getResolvedHook(path, ty, op, target)
	if err != nil {
		return nil, err
	}
	return hook, nil
}

func getHookPathAndTarget(hook string) (string, uint16, error) {
	pathAndTarget := strings.Split(hook, ",")
	if len(pathAndTarget) == 1 {
		pathAndTarget = append(pathAndTarget, "master")
	}
	target, ok := toTypeInt[pathAndTarget[1]]
	if !ok {
		return "", 0x0, fmt.Errorf("invalid role:%s", pathAndTarget[1])
	}

	return pathAndTarget[0], target, nil
}

func getResolvedHook(path string, ty api.HookType, op api.HookOperator, target uint16) (*api.ClusterHookConf, error) {

	dir, shells, err := getDirAndShells(path)
	if err != nil {
		return nil, err
	}

	return &api.ClusterHookConf{
		Type:       ty,
		Operator:   op,
		Target:     target,
		HookSrcDir: dir,
		HookFiles:  shells,
	}, nil
}

func getDirAndShells(path string) (string, []string, error) {
	file, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}

	if !file.IsDir() {
		return resolveFile(path)
	}

	return resolvePath(path)
}

func resolveFile(p string) (string, []string, error) {
	dir := path.Dir(p)
	fileName := path.Base(p)
	if err := checkHookFile(p); err != nil {
		return "", nil, err
	}

	return dir, []string{fileName}, nil
}

func resolvePath(p string) (string, []string, error) {
	var files []string
	rd, err := ioutil.ReadDir(p)
	if err != nil {
		return "", nil, err
	}

	for _, fi := range rd {
		if err := checkHookFile(path.Join(p, fi.Name())); err == nil {
			files = append(files, fi.Name())
		} else {
			logrus.Debugf("check hook file failed:%v", err)
		}
	}
	if len(files) == 0 {
		return "", nil, fmt.Errorf("empty folder:%s", p)
	}
	return p, files, nil
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
		etcds = getHostconfigs("etcd-%d", etcdsIP)
	} else {
		etcds = getHostconfigs("k8s-master-%d", etcdsIP)
	}
	masters = getHostconfigs("k8s-master-%d", masterIP)
	workers = getHostconfigs("k8s-worker-%d", workersIP)
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
	conf := &DeployConfig{
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
		Service: ServiceClusterConfig{
			CIDR:    "10.32.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.32.0.1",
			DNS: DnsConfig{
				CorednsType: "binary",
			},
		},
		NetWork: NetworkConfig{
			PodCIDR:    "10.244.0.0/16",
			Plugin:     "calico",
			PluginArgs: make(map[string]string),
		},
		ApiServerEndpoint: fmt.Sprintf("%s:%d", lb.Ip, lb.BindPort),
		ApiServerCertSans: Sans{},
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
		OpenPorts: map[string][]*OpenPorts{
			"worker": {
				&OpenPorts{
					Port:     111,
					Protocol: "tcp",
				},
				&OpenPorts{
					Port:     179,
					Protocol: "tcp",
				},
			},
			"master": {
				&OpenPorts{
					Port:     53,
					Protocol: "tcp",
				},
				&OpenPorts{
					Port:     53,
					Protocol: "udp",
				},
				&OpenPorts{
					Port:     9153,
					Protocol: "tcp",
				},
			},
		},
		InstallConfig: InstallConfig{
			PackageSrc: &PackageSrcConfig{
				Type: "tar.gz",
				SrcPath: map[string]string{
					"arm64": "/root/packages/packages-arm64.tar.gz",
					"amd64": "/root/packages/packages-amd64.tar.gz",
				},
			},
			KubernetesMaster: []*PackageConfig{
				{
					Name: "kubernetes-client,kubernetes-master",
					Type: "pkg",
				},
			},
			KubernetesWorker: []*PackageConfig{
				{
					Name: "docker-engine,kubernetes-client,kubernetes-node,kubernetes-kubelet",
					Type: "pkg",
				},
				{
					Name: "conntrack-tools,socat",
					Type: "pkg",
				},
			},
			Container: []*PackageConfig{
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
			Network: []*PackageConfig{
				{
					Name: "containernetworking-plugins",
					Type: "pkg",
				},
			},
			ETCD: []*PackageConfig{
				{
					Name: "etcd",
					Type: "pkg",
				},
			},
			LoadBalance: []*PackageConfig{
				{
					Name: "gd,gperftools-libs,libunwind,libwebp,libxslt",
					Type: "pkg",
				},
				{
					Name: "nginx,nginx-all-modules,nginx-filesystem,nginx-mod-http-image-filter,nginx-mod-http-perl,nginx-mod-http-xslt-filter,nginx-mod-mail,nginx-mod-stream",
					Type: "pkg",
				},
			},
			Image: []*PackageConfig{
				{
					Name: "pause.tar",
					Type: "image",
				},
			},
			Dns: []*PackageConfig{
				{
					Name: "coredns",
					Type: "pkg",
				},
			},
			Addition: map[string][]*PackageConfig{
				"master": {
					{
						Name:     "prejoin.sh",
						Type:     "shell",
						Schedule: string(api.SchedulePreJoin),
						TimeOut:  "30s",
					},
					{
						Name: "calico.yaml",
						Type: "yaml",
					},
				},
				"worker": {
					{
						Name:     "postjoin.sh",
						Type:     "shell",
						Schedule: string(api.SchedulePostJoin),
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
