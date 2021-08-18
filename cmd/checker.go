package cmd

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"time"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/endpoint"
	chain "isula.org/eggo/pkg/utils/responsibilitychain"
	"k8s.io/apimachinery/pkg/util/validation"
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

func checkNodeList(nodes []*HostConfig) error {
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
	}
	return nil
}

func (ccr *NodesResponsibility) Execute() error {
	if err := checkNodeList(ccr.conf.Masters); err != nil {
		return err
	}
	if err := checkNodeList(ccr.conf.Workers); err != nil {
		return err
	}
	if err := checkNodeList(ccr.conf.Etcds); err != nil {
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
				return fmt.Errorf("invalid protocal: %s for %s", port.Protocol, name)
			}
		}
	}

	return nil
}

type InstallConfigResponsibility struct {
	next chain.Responsibility
	conf InstallConfig
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

func (ccr *InstallConfigResponsibility) Execute() error {
	if ccr.conf.PackageSrc != nil {
		if ccr.conf.PackageSrc.DstPath != "" {
			if !filepath.IsAbs(ccr.conf.PackageSrc.DstPath) {
				return fmt.Errorf("srcpackage dst path: %s must be absolute", ccr.conf.PackageSrc.DstPath)
			}
		}
		if ccr.conf.PackageSrc.ArmSrc != "" {
			if !filepath.IsAbs(ccr.conf.PackageSrc.ArmSrc) {
				return fmt.Errorf("srcpackage arm path: %s must be absolute", ccr.conf.PackageSrc.ArmSrc)
			}
		}
		if ccr.conf.PackageSrc.X86Src != "" {
			if !filepath.IsAbs(ccr.conf.PackageSrc.X86Src) {
				return fmt.Errorf("srcpackage x86 path: %s must be absolute", ccr.conf.PackageSrc.X86Src)
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
	install := InstallConfigResponsibility{
		conf: conf.InstallConfig,
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