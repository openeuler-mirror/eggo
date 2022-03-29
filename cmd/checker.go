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
 * Create: 2021-08-18
 * Description: checker for cluster config
 ******************************************************************************/
package cmd

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/validation"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/endpoint"
	chain "isula.org/eggo/pkg/utils/responsibilitychain"
)

type ClusterConfigResponsibility struct {
	next chain.Responsibility
	conf *DeployConfig
}

func (ccr *ClusterConfigResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *ClusterConfigResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func (ccr *ClusterConfigResponsibility) Execute() error {
	if ccr.conf == nil {
		return fmt.Errorf("empty cluster config")
	}
	// check cluster id
	if errs := validation.IsDNS1123Subdomain(ccr.conf.ClusterID); len(errs) > 0 {
		return fmt.Errorf("invalid cluster id: %v", errs)
	}
	// check certificate of ssh
	if ccr.conf.PrivateKeyPath == "" {
		if ccr.conf.Username == "" || ccr.conf.Password == "" {
			return fmt.Errorf("no ceritificate of ssh set")
		}
	} else {
		if !filepath.IsAbs(ccr.conf.PrivateKeyPath) {
			return fmt.Errorf("cluster private key path: %s is not abosulate", ccr.conf.PrivateKeyPath)
		}
	}
	// check nodes of cluster
	if len(ccr.conf.Masters) == 0 {
		return fmt.Errorf("no master, master node is require for cluster")
	}
	// check extral ca path
	if ccr.conf.ExternalCAPath != "" {
		if !filepath.IsAbs(ccr.conf.ExternalCAPath) {
			return fmt.Errorf("cluster external ca path: %s is not abosulate", ccr.conf.ExternalCAPath)
		}
	}
	// check api server endpoint
	if ccr.conf.ApiServerEndpoint != "" {
		if host, port, err := net.SplitHostPort(ccr.conf.ApiServerEndpoint); err != nil {
			return fmt.Errorf("cluster api server endpoint: %s, err: %v", ccr.conf.ApiServerEndpoint, err)
		} else {
			if ip := net.ParseIP(host); ip == nil {
				errs := validation.IsDNS1123Subdomain(host)
				if len(errs) > 0 {
					return fmt.Errorf("invalid domain: '%s' for RFC-1123 subdomain", host)
				}
			}
			if port != "" {
				if _, err := strconv.Atoi(port); err != nil {
					return fmt.Errorf("invalid api server endpoint: %s", ccr.conf.ApiServerEndpoint)
				}
			}
		}
	}
	// check ApiServerTimeout
	if ccr.conf.ApiServerTimeout != "" {
		if _, err := time.ParseDuration(ccr.conf.ApiServerTimeout); err != nil {
			return fmt.Errorf("invalid timeout format: %s", ccr.conf.ApiServerTimeout)
		}
	}
	// check dns ip
	if ccr.conf.DnsVip != "" {
		if ip := net.ParseIP(ccr.conf.DnsVip); ip == nil {
			return fmt.Errorf("invalid dns ip: %s", ccr.conf.DnsVip)
		}
	}
	// check dns domain
	if ccr.conf.DnsDomain != "" {
		if errs := validation.IsDNS1123Subdomain(ccr.conf.DnsDomain); len(errs) > 0 {
			return fmt.Errorf("invalid dns domain: %v", errs)
		}
	}
	// check CniBinDir
	if ccr.conf.CniBinDir != "" {
		if !filepath.IsAbs(ccr.conf.CniBinDir) {
			return fmt.Errorf("cni bin dir: %s is not abosulate", ccr.conf.CniBinDir)
		}
	}
	// check RuntimeEndpoint
	if ccr.conf.RuntimeEndpoint != "" {
		if _, err := url.Parse(ccr.conf.RuntimeEndpoint); err != nil {
			return fmt.Errorf("invalid runtime endpoint: %s, err: %v", ccr.conf.RuntimeEndpoint, err)
		}
	}

	return nil
}

type NodesResponsibility struct {
	next chain.Responsibility
	conf *DeployConfig
}

func (ccr *NodesResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *NodesResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func checkHostconfig(h *HostConfig) error {
	if h == nil {
		return fmt.Errorf("empty hostconfig")
	}
	if h.Name == "" {
		return fmt.Errorf("empty host name")
	}
	if errs := validation.IsDNS1123Subdomain(h.Name); len(errs) > 0 {
		return fmt.Errorf("invalid host name: %v", errs)
	}
	if h.Ip == "" {
		return fmt.Errorf("host: %s ip is null", h.Name)
	}
	if ip := net.ParseIP(h.Ip); ip == nil {
		return fmt.Errorf("invalid host ip: %s", h.Ip)
	}
	if !endpoint.ValidPort(h.Port) {
		return fmt.Errorf("invalid host port: %v", h.Port)
	}
	return nil
}

func compareHost(a, b *HostConfig) bool {
	return a.Ip == b.Ip && a.Name == b.Name && a.Port == b.Port && a.Arch == b.Arch
}

func checkNodeList(nodes []*HostConfig, allHosts map[string]*HostConfig) error {
	useIPs := make(map[string]bool)
	useNames := make(map[string]bool)
	for _, m := range nodes {
		if err := checkHostconfig(m); err != nil {
			return fmt.Errorf("invalid master %s, err: %v", m.Name, err)
		}
		if _, ok := useIPs[m.Ip]; ok {
			return fmt.Errorf("duplicate ip: %s", m.Ip)
		}
		useIPs[m.Ip] = true
		if _, ok := useNames[m.Name]; ok {
			return fmt.Errorf("duplicate name: %s", m.Name)
		}
		useNames[m.Name] = true
		if fh, ok := allHosts[m.Ip]; ok {
			if !compareHost(m, fh) {
				return fmt.Errorf("same ip in different host: %v, %v", fh, m)
			}
		} else {
			allHosts[m.Ip] = m
		}
	}
	return nil
}

func (ccr *NodesResponsibility) Execute() error {
	allHosts := make(map[string]*HostConfig, len(ccr.conf.Masters)+len(ccr.conf.Workers))
	if err := checkNodeList(ccr.conf.Masters, allHosts); err != nil {
		return err
	}
	if err := checkNodeList(ccr.conf.Workers, allHosts); err != nil {
		return err
	}
	if err := checkNodeList(ccr.conf.Etcds, allHosts); err != nil {
		return err
	}

	if ccr.conf.LoadBalance.Ip != "" {
		if ip := net.ParseIP(ccr.conf.LoadBalance.Ip); ip == nil {
			return fmt.Errorf("invalid loadbalance ip: %s", ccr.conf.LoadBalance.Ip)
		}
		if ccr.conf.LoadBalance.Port == 0 || ccr.conf.LoadBalance.BindPort == 0 {
			return fmt.Errorf("loadbalance ip set, must set port and bindport")
		}
	}
	if ccr.conf.LoadBalance.Port != 0 {
		if !endpoint.ValidPort(ccr.conf.LoadBalance.Port) {
			return fmt.Errorf("invalid loadbalance port: %v", ccr.conf.LoadBalance.Port)
		}
	}
	if ccr.conf.LoadBalance.BindPort != 0 {
		if !endpoint.ValidPort(ccr.conf.LoadBalance.BindPort) {
			return fmt.Errorf("invalid loadbalance bind port: %v", ccr.conf.LoadBalance.BindPort)
		}
	}

	return nil
}

type ServiceClusterResponsibility struct {
	next chain.Responsibility
	conf ServiceClusterConfig
}

func (ccr *ServiceClusterResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *ServiceClusterResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func (ccr *ServiceClusterResponsibility) Execute() error {
	if ccr.conf.CIDR != "" {
		if _, _, err := net.ParseCIDR(ccr.conf.CIDR); err != nil {
			return fmt.Errorf("invalid service cidr: %s, err: %v", ccr.conf.CIDR, err)
		}
	}

	if ccr.conf.DNSAddr != "" {
		if ip := net.ParseIP(ccr.conf.DNSAddr); ip == nil {
			return fmt.Errorf("invalid dns address: %s", ccr.conf.DNSAddr)
		}
	}
	if ccr.conf.Gateway != "" {
		if ip := net.ParseIP(ccr.conf.Gateway); ip == nil {
			return fmt.Errorf("invalid dns gateway: %s", ccr.conf.Gateway)
		}
	}

	return nil
}

type NetworkResponsibility struct {
	next chain.Responsibility
	conf NetworkConfig
}

func (ccr *NetworkResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *NetworkResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func (ccr *NetworkResponsibility) Execute() error {
	if ccr.conf.PodCIDR != "" {
		if _, _, err := net.ParseCIDR(ccr.conf.PodCIDR); err != nil {
			return fmt.Errorf("invalid pod cidr: %s, err: %v", ccr.conf.PodCIDR, err)
		}
	}

	return nil
}

type ApiSansResponsibility struct {
	next chain.Responsibility
	conf Sans
}

func (ccr *ApiSansResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *ApiSansResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func (ccr *ApiSansResponsibility) Execute() error {
	for _, name := range ccr.conf.DNSNames {
		if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
			return fmt.Errorf("invalid api san dns: %v", errs)
		}
	}
	for _, sip := range ccr.conf.IPs {
		if ip := net.ParseIP(sip); ip == nil {
			return fmt.Errorf("invalid api san ip: %s", sip)
		}
	}

	return nil
}

type OpenPortResponsibility struct {
	next chain.Responsibility
	conf *DeployConfig
}

func (ccr *OpenPortResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *OpenPortResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func (ccr *OpenPortResponsibility) Execute() error {
	supportProtocal := map[string]bool{
		"udp": true,
		"tcp": true,
	}
	for name, v := range ccr.conf.OpenPorts {
		for _, port := range v {
			if !endpoint.ValidPort(port.Port) {
				return fmt.Errorf("invalid port: %v for %s", port.Port, name)
			}
			if _, ok := supportProtocal[port.Protocol]; !ok {
				return fmt.Errorf("invalid protocol: %s for %s", port.Protocol, name)
			}
		}
	}

	return nil
}

type InstallConfigResponsibility struct {
	next chain.Responsibility
	conf InstallConfig
	arch map[string]bool
}

func (ccr *InstallConfigResponsibility) SetNexter(nexter chain.Responsibility) {
	ccr.next = nexter
}

func (ccr *InstallConfigResponsibility) Nexter() chain.Responsibility {
	return ccr.next
}

func checkPackageConfig(pc *PackageConfig) error {
	if pc == nil {
		return errors.New("empty package config")
	}
	if pc.Name == "" {
		return errors.New("empty package name")
	}
	if pc.Type == "" {
		return fmt.Errorf("empty package type for package: %s", pc.Name)
	}
	if pc.Dst != "" {
		if !filepath.IsAbs(pc.Dst) {
			return fmt.Errorf("package config dst path: %s must be absolute", pc.Dst)
		}
	}
	if pc.TimeOut != "" {
		if _, err := time.ParseDuration(pc.TimeOut); err != nil {
			return fmt.Errorf("invalid timeout: %s for package: %s", pc.TimeOut, pc.Name)
		}
	}
	if pc.Schedule != "" {
		_, err := api.ParseScheduleType(pc.Schedule)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkCmdHooksParameter(pa ...string) error {
	for _, v := range pa {
		if v == "" {
			continue
		}
		res := strings.Split(v, ",")
		if len(res) < 1 || len(res) > 2 {
			return fmt.Errorf("invalid hook parameter with:%s\n", v)
		}
	}

	return nil
}

func checkHookFile(fileName string) error {
	file, err := os.Stat(fileName)
	if err != nil {
		return err

	}

	if !path.IsAbs(fileName) {
		return fmt.Errorf("%s is not Abs path", fileName)
	}
	if !file.Mode().IsRegular() {
		return fmt.Errorf("%s is not regular file", file.Name())
	}
	if file.Mode().Perm() != constants.HookFileMode {
		return fmt.Errorf("file mode of %s is incorrect", file.Name())
	}
	if file.Size() > constants.MaxHookFileSize || file.Size() == 0 {
		return fmt.Errorf("%s is too large or small", file.Name())
	}
	if !(strings.HasSuffix(fileName, ".sh") || strings.HasSuffix(fileName, ".bash")) {
		return fmt.Errorf("%s is not shell file", file.Name())
	}

	user, group, err := utils.GetUserIDAndGroupID(fileName)
	if err != nil {
		return fmt.Errorf("get user ID and group ID with file %s failed", file.Name())
	}
	if user != os.Getuid() && group != os.Getgid() {
		return fmt.Errorf("user id and group id of %s mismatch with process", file.Name())
	}

	return nil
}

func (ccr *InstallConfigResponsibility) Execute() error {
	if ccr.conf.PackageSrc != nil {
		if ccr.conf.PackageSrc.DstPath != "" {
			if !filepath.IsAbs(ccr.conf.PackageSrc.DstPath) {
				return fmt.Errorf("srcpackage dst path: %s must be absolute", ccr.conf.PackageSrc.DstPath)
			}
		}

		for arch, path := range ccr.conf.PackageSrc.SrcPath {
			if !filepath.IsAbs(path) {
				return fmt.Errorf("srcpackage %s path: %s must be absolute", arch, path)
			}
			if _, ok := ccr.arch[arch]; ok {
				exist, err := utils.CheckPathExist(path)
				if err != nil {
					return err
				}
				if !exist {
					return fmt.Errorf("have arch: %s node, but src package: %s is not exist", arch, path)
				}
			}
		}

		if len(ccr.conf.PackageSrc.SrcPath) != 0 {
			for a := range ccr.arch {
				if _, ok := ccr.conf.PackageSrc.SrcPath[a]; !ok {
					return fmt.Errorf("no source package for arch %s", a)
				}
			}
		}
	}

	for _, km := range ccr.conf.KubernetesMaster {
		if err := checkPackageConfig(km); err != nil {
			return err
		}
	}
	for _, kw := range ccr.conf.KubernetesWorker {
		if err := checkPackageConfig(kw); err != nil {
			return err
		}
	}
	for _, net := range ccr.conf.Network {
		if err := checkPackageConfig(net); err != nil {
			return err
		}
	}
	for _, etcd := range ccr.conf.ETCD {
		if err := checkPackageConfig(etcd); err != nil {
			return err
		}
	}
	for _, lb := range ccr.conf.LoadBalance {
		if err := checkPackageConfig(lb); err != nil {
			return err
		}
	}
	for _, c := range ccr.conf.Container {
		if err := checkPackageConfig(c); err != nil {
			return err
		}
	}
	for _, img := range ccr.conf.Image {
		if err := checkPackageConfig(img); err != nil {
			return err
		}
	}
	for key, adds := range ccr.conf.Addition {
		for _, add := range adds {
			if err := checkPackageConfig(add); err != nil {
				return fmt.Errorf("addition: %s, err: %v", key, err)
			}
		}
	}

	return nil
}

func RunChecker(conf *DeployConfig) error {
	if conf == nil {
		return errors.New("deploy config is nil")
	}

	arch := make(map[string]bool)
	for _, m := range conf.Masters {
		arch[m.Arch] = true
	}
	for _, w := range conf.Workers {
		arch[w.Arch] = true
	}
	for _, m := range conf.Masters {
		arch[m.Arch] = true
	}
	for _, m := range conf.Masters {
		arch[m.Arch] = true
	}
	if conf.LoadBalance.Arch != "" {
		arch[conf.LoadBalance.Arch] = true
	}

	install := InstallConfigResponsibility{
		conf: conf.InstallConfig,
		arch: arch,
	}
	openport := OpenPortResponsibility{
		next: &install,
		conf: conf,
	}
	sans := ApiSansResponsibility{
		next: &openport,
		conf: conf.ApiServerCertSans,
	}
	network := NetworkResponsibility{
		next: &sans,
		conf: conf.NetWork,
	}
	service := ServiceClusterResponsibility{
		next: &network,
		conf: conf.Service,
	}
	nodes := NodesResponsibility{
		next: &service,
		conf: conf,
	}
	cluster := ClusterConfigResponsibility{
		next: &nodes,
		conf: conf,
	}
	return chain.RunChainOfResponsibility(&cluster)
}
