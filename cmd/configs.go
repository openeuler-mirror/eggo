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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v1"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	masterPackages = map[string]*clusterdeployment.Packages{
		"kubernetes-master": {
			Type: "repo",
		},
		"kubernetes-kubeadm": {
			Type: "repo",
		},
		"kubernetes-client": {
			Type: "repo",
		},
		"coredns": {
			Type: "repo",
		},
	}

	masterExports = []*clusterdeployment.OpenPorts{
		// kube-apiserver
		{
			Port:     6443,
			Protocol: "tcp",
		},
		// kube-scheduler
		{
			Port:     10251,
			Protocol: "tcp",
		},
		// kube-controller-manager
		{
			Port:     10252,
			Protocol: "tcp",
		},
	}

	nodePackages = map[string]*clusterdeployment.Packages{
		"kubernetes-node": {
			Type: "repo",
		},
		"kubernetes-kubelet": {
			Type: "repo",
		},
		"conntrack-tools": {
			Type: "repo",
		},
		"socat": {
			Type: "repo",
		},
		"containernetworking-plugins": {
			Type: "repo",
		},
	}

	nodeExports = []*clusterdeployment.OpenPorts{
		// kubelet
		{
			Port:     10250,
			Protocol: "tcp",
		},
		// kube-proxy
		{
			Port:     10256,
			Protocol: "tcp",
		},
	}

	etcdPackages = map[string]*clusterdeployment.Packages{
		"etcd": {
			Type: "repo",
		},
	}

	etcdExports = []*clusterdeployment.OpenPorts{
		// etcd api
		{
			Port:     2379,
			Protocol: "tcp",
		},
		// etcd peer
		{
			Port:     2380,
			Protocol: "tcp",
		},
		// etcd metric
		{
			Port:     2381,
			Protocol: "tcp",
		},
	}

	loadbalancePackages = map[string]*clusterdeployment.Packages{
		"nginx": {
			Type: "repo",
		},
	}
)

type ConfigExtraArgs struct {
	Name      string            `yaml:"name"`
	ExtraArgs map[string]string `yaml:"extra-args"`
}

type Package struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`    // repo, pkg, binary
	Dst  string `yaml:"dstpath"` // used only when type is binary
}

type HostConfig struct {
	Name string `yaml:"name"`
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
	Arch string `yaml:"arch"` // amd64, aarch64, default amd64
}

type deployConfig struct {
	ClusterID         string                                 `yaml:"cluster-id"`
	Username          string                                 `yaml:"username"`
	Password          string                                 `yaml:"password"`
	Masters           []*HostConfig                          `yaml:"masters"`
	Nodes             []*HostConfig                          `yaml:"nodes"`
	Etcds             []*HostConfig                          `yaml:"etcds"`
	LoadBalances      []*HostConfig                          `yaml:"loadbalances"`
	ConfigDir         string                                 `yaml:"config-dir"`
	CertificateDir    string                                 `yaml:"certificate-dir"`
	ExternalCA        bool                                   `yaml:"external-ca"`
	ExternalCAPath    string                                 `yaml:"external-ca-path"`
	Service           clusterdeployment.ServiceClusterConfig `yaml:"service"`
	NetWork           clusterdeployment.NetworkConfig        `yaml:"network"`
	ApiServerEndpoint string                                 `yaml:"apiserver-endpoint"`
	ApiServerCertSans clusterdeployment.Sans                 `yaml:"apiserver-cert-sans"`
	ApiServerTimeout  string                                 `yaml:"apiserver-timeout"`
	EtcdExternal      bool                                   `yaml:"etcd-external"`
	EtcdToken         string                                 `yaml:"etcd-token"`
	EtcdDataDir       string                                 `yaml:"etcd-data-dir"`
	ConfigExtraArgs   []*ConfigExtraArgs                     `yaml:"config-extra-args"`
	PackageSrc        clusterdeployment.PackageSrcConfig     `yaml:"package-src"`
	Packages          map[string][]*Package                  `yaml:"pacakges"` // key: master, node, etcd, loadbalance
}

func getEggoDir() string {
	return filepath.Join(utils.GetSysHome(), ".eggo")
}

func init() {
	if _, err := os.Stat(getEggoDir()); err == nil {
		return
	}

	if err := os.Mkdir(getEggoDir(), 0700); err != nil {
		logrus.Errorf("mkdir eggo directory %v failed", getEggoDir())
	}
}

func getDefaultDeployConfig() string {
	return filepath.Join(getEggoDir(), "deploy.yaml")
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

func getDefaultClusterdeploymentConfig() *clusterdeployment.ClusterConfig {
	return &clusterdeployment.ClusterConfig{
		Name:      "k8s-cluster",
		ConfigDir: utils.DefaultK8SRootDir,
		Certificate: clusterdeployment.CertificateConfig{
			SavePath: utils.DefaultK8SCertDir,
		},
		ServiceCluster: clusterdeployment.ServiceClusterConfig{
			CIDR:    "10.244.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.244.0.1",
		},
		Network: clusterdeployment.NetworkConfig{
			PodCIDR:    "10.244.64.0/16",
			PluginArgs: make(map[string]string),
		},
		LocalEndpoint: clusterdeployment.APIEndpoint{
			AdvertiseAddress: "127.0.0.1",
			BindPort:         6443,
		},
		ControlPlane: clusterdeployment.ControlPlaneConfig{
			ApiConf: &clusterdeployment.ApiServer{
				Timeout: "120s",
			},
		},
		PackageSrc: &clusterdeployment.PackageSrcConfig{
			Type:   "tar.gz",
			ArmSrc: "./pacakges-arm.tar.gz",
			X86Src: "./packages-x86.tar.gz",
		},
		EtcdCluster: clusterdeployment.EtcdClusterConfig{
			Token:    "etcd-cluster",
			DataDir:  "/var/lib/datadir",
			CertsDir: utils.DefaultK8SCertDir,
			External: false,
		},
	}
}

func createCommonHostConfig(userHostconfig *HostConfig, defaultName string, username string,
	password string) *clusterdeployment.HostConfig {
	arch, name, port := "amd64", defaultName, 22
	if userHostconfig.Arch != "" {
		arch = userHostconfig.Arch
	}
	if userHostconfig.Name != "" {
		name = userHostconfig.Name
	}
	if userHostconfig.Port != 0 {
		port = userHostconfig.Port
	}

	hostconfig := &clusterdeployment.HostConfig{
		Arch:     arch,
		Name:     name,
		Address:  userHostconfig.Ip,
		Port:     port,
		UserName: username,
		Password: password,
	}

	return hostconfig
}

func portExist(openPorts []*clusterdeployment.OpenPorts, port *clusterdeployment.OpenPorts) bool {
	for _, p := range openPorts {
		if p.Protocol == port.Protocol && p.Port == port.Port {
			return true
		}
	}
	return false
}

func addPackagesAndExports(hostconfig *clusterdeployment.HostConfig, pkgs map[string]*clusterdeployment.Packages,
	openPorts []*clusterdeployment.OpenPorts) {
	if hostconfig.Packages == nil {
		hostconfig.Packages = make(map[string]clusterdeployment.Packages)
	}

	for name, pkg := range pkgs {
		hostconfig.Packages[name] = *pkg
	}

	for _, port := range openPorts {
		if portExist(hostconfig.OpenPorts, port) {
			continue
		}
		hostconfig.OpenPorts = append(hostconfig.OpenPorts, port)
	}
}

func addUserPackages(hostconfig *clusterdeployment.HostConfig, pkgs []*Package) {
	if hostconfig.Packages == nil {
		hostconfig.Packages = make(map[string]clusterdeployment.Packages)
	}
	for _, pkg := range pkgs {
		hostconfig.Packages[pkg.Name] = clusterdeployment.Packages{
			Type: pkg.Type,
			Dst:  pkg.Dst,
		}
	}
}

func getPortFromEndPoint(endpoint string) int {
	defaultPort := 6443

	// endpoint:
	// https://192.168.0.11:6443
	uri, err := url.Parse(endpoint)
	if err != nil {
		return defaultPort
	}

	// host:
	// 192.168.0.11:6443
	items := strings.Split(uri.Host, ":")
	if len(items) < 2 {
		return defaultPort
	}

	port, err := strconv.Atoi(items[len(items)-1])
	if err != nil {
		return defaultPort
	}

	return port
}

func fillHostConfig(ccfg *clusterdeployment.ClusterConfig, conf *deployConfig) {
	var hostconfig *clusterdeployment.HostConfig
	var exist bool
	cache := make(map[string]*clusterdeployment.HostConfig)

	for i, master := range conf.Masters {
		hostconfig = createCommonHostConfig(master, "k8s-master-"+strconv.Itoa(i), conf.Username, conf.Password)
		hostconfig.Type |= clusterdeployment.Master
		addPackagesAndExports(hostconfig, masterPackages, masterExports)
		addUserPackages(hostconfig, conf.Packages["master"])
		cache[hostconfig.Address] = hostconfig
	}

	for i, node := range conf.Nodes {
		hostconfig, exist = cache[node.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(node, "k8s-node-"+strconv.Itoa(i), conf.Username,
				conf.Password)
		}
		hostconfig.Type |= clusterdeployment.Worker
		addPackagesAndExports(hostconfig, nodePackages, nodeExports)
		addUserPackages(hostconfig, conf.Packages["node"])
		cache[hostconfig.Address] = hostconfig
	}

	// if no etcd configed, default to install to master
	var etcds []*HostConfig
	if len(conf.Etcds) == 0 {
		etcds = conf.Masters
	}
	for i, etcd := range etcds {
		hostconfig, exist = cache[etcd.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(etcd, "etcd-"+strconv.Itoa(i), conf.Username, conf.Password)
		}
		hostconfig.Type |= clusterdeployment.ETCD
		addPackagesAndExports(hostconfig, etcdPackages, etcdExports)
		addUserPackages(hostconfig, conf.Packages["etcd"])
		cache[hostconfig.Address] = hostconfig
	}

	for i, lb := range conf.LoadBalances {
		hostconfig, exist = cache[lb.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(lb, "k8s-loadbalance-"+strconv.Itoa(i), conf.Username,
				conf.Password)
		}
		hostconfig.Type |= clusterdeployment.LoadBalance

		addPackagesAndExports(hostconfig, loadbalancePackages, []*clusterdeployment.OpenPorts{
			{
				Port:     getPortFromEndPoint(conf.ApiServerEndpoint),
				Protocol: "tcp",
			},
		})
		addUserPackages(hostconfig, conf.Packages["loadbalance"])
		cache[hostconfig.Address] = hostconfig
	}

	for _, v := range cache {
		ccfg.Nodes = append(ccfg.Nodes, v)
	}
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

func setStrArray(config []string, userConfig []string) {
	for _, v := range userConfig {
		if notInStrArray(config, v) {
			config = append(config, v)
		}
	}
}

func toClusterdeploymentConfig(conf *deployConfig) *clusterdeployment.ClusterConfig {
	ccfg := getDefaultClusterdeploymentConfig()

	setIfStrConfigNotEmpty(&ccfg.Name, conf.ClusterID)
	fillHostConfig(ccfg, conf)
	setIfStrConfigNotEmpty(&ccfg.ConfigDir, conf.ConfigDir)
	setIfStrConfigNotEmpty(&ccfg.Certificate.SavePath, conf.CertificateDir)
	ccfg.Certificate.ExternalCA = conf.ExternalCA
	setIfStrConfigNotEmpty(&ccfg.Certificate.ExternalCAPath, conf.ExternalCAPath)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.CIDR, conf.Service.CIDR)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.DNSAddr, conf.Service.DNSAddr)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.Gateway, conf.Service.Gateway)
	setIfStrConfigNotEmpty(&ccfg.Network.PodCIDR, conf.NetWork.PodCIDR)
	setIfStrConfigNotEmpty(&ccfg.Network.Plugin, conf.NetWork.Plugin)
	setStrStrMap(ccfg.Network.PluginArgs, conf.NetWork.PluginArgs)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.Endpoint, conf.ApiServerEndpoint)
	setStrArray(ccfg.ControlPlane.ApiConf.CertSans.DNSNames, conf.ApiServerCertSans.DNSNames)
	setStrArray(ccfg.ControlPlane.ApiConf.CertSans.IPs, conf.ApiServerCertSans.IPs)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.ApiConf.Timeout, conf.ApiServerTimeout)
	ccfg.EtcdCluster.External = conf.EtcdExternal
	setIfStrConfigNotEmpty(&ccfg.EtcdCluster.Token, conf.EtcdToken)
	setIfStrConfigNotEmpty(&ccfg.EtcdCluster.DataDir, conf.EtcdDataDir)
	setIfStrConfigNotEmpty(&ccfg.PackageSrc.Type, conf.PackageSrc.Type)
	setIfStrConfigNotEmpty(&ccfg.PackageSrc.ArmSrc, conf.PackageSrc.ArmSrc)
	setIfStrConfigNotEmpty(&ccfg.PackageSrc.X86Src, conf.PackageSrc.X86Src)

	return ccfg
}

func createDeployConfigTemplate(file string) error {
	conf := &deployConfig{
		ClusterID: "k8s-cluster",
		Username:  "root",
		Password:  "password",
		Masters: []*HostConfig{
			{
				Name: "k8s-master-0",
				Ip:   "192.168.0.11",
				Port: 22,
				Arch: "amd64",
			},
		},
		Nodes: []*HostConfig{
			{
				Name: "k8s-node-0",
				Ip:   "192.168.0.12",
				Port: 22,
				Arch: "amd64",
			},
		},
		Etcds: []*HostConfig{
			{
				Name: "etcd-0",
				Ip:   "192.168.0.13",
				Port: 22,
				Arch: "amd64",
			},
		},
		LoadBalances: []*HostConfig{
			{
				Name: "k8s-loadbalance-0",
				Ip:   "192.168.0.14",
				Port: 22,
				Arch: "amd64",
			},
		},
		ConfigDir:      utils.DefaultK8SRootDir,
		CertificateDir: utils.DefaultK8SCertDir,
		ExternalCA:     false,
		ExternalCAPath: "/opt/externalca",
		Service: clusterdeployment.ServiceClusterConfig{
			CIDR:    "10.244.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.244.0.1",
		},
		NetWork: clusterdeployment.NetworkConfig{
			PodCIDR:    "10.244.64.0/16",
			PluginArgs: make(map[string]string),
		},
		ApiServerEndpoint: "https://192.168.0.11:6443",
		ApiServerCertSans: clusterdeployment.Sans{},
		ApiServerTimeout:  "120s",
		EtcdExternal:      false,
		EtcdToken:         "etcd-cluster",
		EtcdDataDir:       "/var/lib/datadir",
		PackageSrc: clusterdeployment.PackageSrcConfig{
			Type:   "tar.gz",
			ArmSrc: "./pacakges-arm.tar.gz",
			X86Src: "./packages-x86.tar.gz",
		},
		Packages: map[string][]*Package{
			"master": {
				&Package{
					Name: "kubernetes-master",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-kubeadm",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-client",
					Type: "pkg",
				},
				&Package{
					Name: "coredns",
					Type: "pkg",
				},
			},
			"node": {
				&Package{
					Name: "kubernetes-node",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-kubelet",
					Type: "pkg",
				},
				&Package{
					Name: "conntrack-tools",
					Type: "pkg",
				},
				&Package{
					Name: "socat",
					Type: "pkg",
				},
				&Package{
					Name: "containernetworking-plugins",
					Type: "pkg",
				},
			},
			"etcd": {
				&Package{
					Name: "etcd",
					Type: "pkg",
				},
			},
			"nginx": {
				&Package{
					Name: "nginx",
					Type: "pkg",
				},
				&Package{
					Name: "gd",
					Type: "pkg",
				},
				&Package{
					Name: "gperftools-libs",
					Type: "pkg",
				},
				&Package{
					Name: "libunwind",
					Type: "pkg",
				},
				&Package{
					Name: "libwebp",
					Type: "pkg",
				},
				&Package{
					Name: "libxslt",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-all-modules",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-filesystem",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-http-image-filter",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-http-perl",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-http-xslt-filter",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-mail",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-stream",
					Type: "pkg",
				},
			},
		},
	}

	d, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("marshal template config failed: %v", err)
	}

	if err := ioutil.WriteFile(file, d, 0700); err != nil {
		return fmt.Errorf("write template config file failed: %v", err)
	}

	return nil
}
