package controllers

import (
	"encoding/base64"
	"net"
	"net/url"
	"strconv"

	"gopkg.in/yaml.v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	eggov1 "isula.org/eggo/eggops/api/v1"
	eggoapi "isula.org/eggo/eggops/controllers/eggo"
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

func toEggoHosts(machines []*eggov1.Machine) []*eggoapi.HostConfig {
	var result []*eggoapi.HostConfig
	for _, m := range machines {
		result = append(result, &eggoapi.HostConfig{
			Name: m.Spec.HostName,
			Ip:   m.Spec.IP,
			Port: int(*m.Spec.Port),
			Arch: m.Spec.Arch,
		})
	}
	return result
}

func ConvertClusterToEggoConfig(cluster *eggov1.Cluster, mb *eggov1.MachineBinding) ([]byte, error) {
	conf := eggoapi.DeployConfig{}
	// set cluster config
	conf.ClusterID = cluster.GetName()
	// TODO: get user and password from cluster
	conf.Username = "root"

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
	for _, set := range mb.Spec.MachineSets {
		if set.MatchType(eggov1.UsageMaster) {
			conf.Masters = toEggoHosts(set.Machines)
		} else if set.MatchType(eggov1.UsageWorker) {
			conf.Workers = toEggoHosts(set.Machines)
		} else if set.MatchType(eggov1.UsageEtcd) {
			conf.Etcds = toEggoHosts(set.Machines)
		} else if set.MatchType(eggov1.UsageLoadbalance) {
			if len(set.Machines) != 1 {
				continue
			}
			conf.LoadBalance = eggoapi.LoadBalance{
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
	data := base64.StdEncoding.EncodeToString(d)
	return []byte(data), nil
}

func ReferenceToNamespacedName(ref *v1.ObjectReference) types.NamespacedName {
	return types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
}
