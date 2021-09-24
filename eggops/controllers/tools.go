package controllers

import (
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"isula.org/eggo/cmd"
	eggov1 "isula.org/eggo/eggops/api/v1"
)

func getEndpoint(conf eggov1.APIEndpointConfig) string {
	port := "6443"
	if conf.BindPort != nil {
		port = strconv.Itoa(int(*conf.BindPort))
	}
	turl := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(conf.Advertise, port),
	}
	return turl.String()
}

func toEggoHosts(machines []*eggov1.Machine) []*cmd.HostConfig {
	var result []*cmd.HostConfig
	for _, m := range machines {
		result = append(result, &cmd.HostConfig{
			Name: m.Spec.HostName,
			Ip:   m.Spec.IP,
			Port: int(*m.Spec.Port),
			Arch: m.Spec.Arch,
		})
	}
	return result
}

func fillPackageConfig(src []*eggov1.PackageConfig) []*cmd.PackageConfig {
	var copy []*cmd.PackageConfig
	for _, pc := range src {
		copy = append(copy, &cmd.PackageConfig{
			Name:     pc.Name,
			Type:     pc.Type,
			Dst:      pc.Dst,
			Schedule: pc.Schedule,
			TimeOut:  pc.TimeOut,
		})
	}

	return copy
}

func fillInstallConfig(installConfig eggov1.InstallConfig, packagePath string) (config cmd.InstallConfig) {
	// TODO: eggo src package api changes
	if installConfig.PackageSrc != nil {
		srcPath := make(map[string]string)

		srcPath["arm"] = filepath.Join(packagePath, eggov1.DefaultPackageArmName)
		srcPath["x86"] = filepath.Join(packagePath, eggov1.DefaultPackageX86Name)
		srcPath["risc-v"] = filepath.Join(packagePath, eggov1.DefaultPackageRISCVName)
		for arch, path := range installConfig.PackageSrc.SrcPackages {
			srcPath[arch] = filepath.Join(packagePath, path)
		}

		config.PackageSrc = &cmd.PackageSrcConfig{
			Type:    installConfig.PackageSrc.Type,
			DstPath: installConfig.PackageSrc.DstPath,
			SrcPath: srcPath,
		}
	}

	config.Addition = make(map[string][]*cmd.PackageConfig)
	packageConfigs := []struct {
		src []*eggov1.PackageConfig
		dst *[]*cmd.PackageConfig
	}{
		{installConfig.KubernetesMaster, &config.KubernetesMaster},
		{installConfig.KubernetesWorker, &config.KubernetesWorker},
		{installConfig.Network, &config.Network},
		{installConfig.ETCD, &config.ETCD},
		{installConfig.LoadBalance, &config.LoadBalance},
		{installConfig.Container, &config.Container},
		{installConfig.Image, &config.Image},
	}
	for _, p := range packageConfigs {
		if len(p.src) != 0 {
			*p.dst = fillPackageConfig(p.src)
		}
	}

	if len(installConfig.Addition.Master) != 0 {
		config.Addition["master"] = fillPackageConfig(installConfig.Addition.Master)
	}
	if len(installConfig.Addition.Worker) != 0 {
		config.Addition["worker"] = fillPackageConfig(installConfig.Addition.Worker)
	}
	if len(installConfig.Addition.ETCD) != 0 {
		config.Addition["etcd"] = fillPackageConfig(installConfig.Addition.ETCD)
	}
	if len(installConfig.Addition.LoadBalance) != 0 {
		config.Addition["loadbalance"] = fillPackageConfig(installConfig.Addition.LoadBalance)
	}

	return
}

func fillOpenPorts(src []*eggov1.OpenPorts) []*cmd.OpenPorts {
	var copy []*cmd.OpenPorts
	for _, op := range src {
		copy = append(copy, &cmd.OpenPorts{
			Port:     int(*op.Port),
			Protocol: op.Protocol,
		})
	}

	return copy
}

func fillOpenPortsConfig(openPorts eggov1.OpenPortsConfig) map[string][]*cmd.OpenPorts {
	copy := make(map[string][]*cmd.OpenPorts)
	if len(openPorts.Master) != 0 {
		copy["master"] = fillOpenPorts(openPorts.Master)
	}
	if len(openPorts.Worker) != 0 {
		copy["worker"] = fillOpenPorts(openPorts.Worker)
	}
	if len(openPorts.ETCD) != 0 {
		copy["etcd"] = fillOpenPorts(openPorts.ETCD)
	}
	if len(openPorts.LoadBalance) != 0 {
		copy["loadbalance"] = fillOpenPorts(openPorts.LoadBalance)
	}

	return copy
}

func ConvertClusterToEggoConfig(cluster *eggov1.Cluster, mb *eggov1.MachineBinding, secret *v1.Secret, infrastructure *eggov1.Infrastructure) ([]byte, error) {
	conf := cmd.DeployConfig{}
	// set cluster config
	conf.ClusterID = cluster.GetName()

	if secret.Type == v1.SecretTypeSSHAuth {
		conf.PrivateKeyPath = filepath.Join(fmt.Sprintf(eggov1.PrivateKeyVolumeFormat, cluster.Name), v1.SSHAuthPrivateKey)
	} else {
		conf.Username = string(secret.Data[v1.BasicAuthUsernameKey])
		conf.Password = string(secret.Data[v1.BasicAuthPasswordKey])
	}

	packagePath := fmt.Sprintf(eggov1.PackageVolumeFormat, cluster.Name)
	conf.InstallConfig = fillInstallConfig(infrastructure.Spec.InstallConfig, packagePath)

	conf.OpenPorts = fillOpenPortsConfig(infrastructure.Spec.OpenPorts)

	conf.EnableKubeletServing = false
	if cluster.Spec.EnableKubeletServing {
		conf.EnableKubeletServing = true
	}

	if cluster.Spec.ApiEndpoint.Advertise != "" {
		conf.ApiServerEndpoint = getEndpoint(cluster.Spec.ApiEndpoint)
	}
	// set runtime
	if cluster.Spec.Runtime.Runtime != "" {
		conf.Runtime = cluster.Spec.Runtime.Runtime
	}
	if cluster.Spec.Runtime.RuntimeEndpoint != "" {
		conf.RuntimeEndpoint = cluster.Spec.Runtime.RuntimeEndpoint
	}
	// set network of service
	if cluster.Spec.Network.ServiceCidr != "" {
		conf.Service.CIDR = cluster.Spec.Network.ServiceCidr
	}
	if cluster.Spec.Network.ServiceDnsIp != "" {
		conf.Service.DNSAddr = cluster.Spec.Network.ServiceDnsIp
	}
	if cluster.Spec.Network.ServiceGateway != "" {
		conf.Service.Gateway = cluster.Spec.Network.ServiceGateway
	}
	// set network of pod
	if cluster.Spec.Network.PodCidr != "" {
		conf.NetWork.PodCIDR = cluster.Spec.Network.PodCidr
	}
	if cluster.Spec.Network.PodPlugin != "" {
		conf.NetWork.Plugin = cluster.Spec.Network.PodPlugin
	}
	if cluster.Spec.Network.PodPluginArgs != nil {
		conf.NetWork.PluginArgs = cluster.Spec.Network.PodPluginArgs
	}

	// set machines
	conf.Workers = make([]*cmd.HostConfig, 0)
	for _, set := range mb.Spec.MachineSets {
		if set.MatchType(eggov1.UsageMaster) {
			conf.Masters = toEggoHosts(set.Machines)
			// set master machines as worker machines
			conf.Workers = append(conf.Workers, conf.Masters...)
		} else if set.MatchType(eggov1.UsageWorker) {
			conf.Workers = append(conf.Workers, toEggoHosts(set.Machines)...)
		} else if set.MatchType(eggov1.UsageEtcd) {
			conf.Etcds = toEggoHosts(set.Machines)
		} else if set.MatchType(eggov1.UsageLoadbalance) {
			if len(set.Machines) != 1 {
				continue
			}
			conf.LoadBalance = cmd.LoadBalance{
				Name:     set.Machines[0].Spec.HostName,
				Ip:       set.Machines[0].Spec.IP,
				Port:     int(*set.Machines[0].Spec.Port),
				Arch:     set.Machines[0].Spec.Arch,
				BindPort: int(cluster.Spec.LoadbalanceBindPort),
			}
		}
	}

	d, err := yaml.Marshal(conf)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func ReferenceToNamespacedName(ref *v1.ObjectReference) types.NamespacedName {
	return types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
}

func GetEggoImageVersion(cluster *eggov1.Cluster) string {
	if cluster.Spec.EggoImageVersion != "" {
		return cluster.Spec.EggoImageVersion
	}

	return "eggo:" + eggov1.ImageVersion
}
