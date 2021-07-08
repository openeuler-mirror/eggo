package infra

import "isula.org/eggo/pkg/api"

var (
	//etcd
	EtcdPackages = []*api.PackageConfig{
		{
			Name: "etcd",
			Type: "repo",
		},
	}
	EtcdPorts = []*api.OpenPorts{
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

	// kubernetes master
	MasterPackages = []*api.PackageConfig{
		{
			Name: "kubernetes-client",
			Type: "repo",
		},
		{
			Name: "kubernetes-master",
			Type: "repo",
		},
	}
	MasterPorts = []*api.OpenPorts{
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

	// kubernetes worker
	WorkerPackages = []*api.PackageConfig{
		{
			Name: "kubernetes-client",
			Type: "repo",
		},
		{
			Name: "kubernetes-node",
			Type: "repo",
		},
		{
			Name: "kubernetes-kubelet",
			Type: "repo",
		},
	}
	WorkerPorts = []*api.OpenPorts{
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

	// container engine
	ContainerPackages = []*api.PackageConfig{
		{
			Name: "docker-engine",
			Type: "repo",
		},
	}

	// network plugin
	NetworkPackages = []*api.PackageConfig{
		{
			Name: "containernetworking-plugins",
			Type: "repo",
		},
	}

	// loadbalance
	LoadbalancePackages = []*api.PackageConfig{
		{
			Name: "nginx",
			Type: "repo",
		},
	}

	// coredns
	CorednsPorts = []*api.OpenPorts{
		{
			Port:     111,
			Protocol: "tcp",
		},
		{
			Port:     179,
			Protocol: "tcp",
		},
	}
)

func RegisterInfra() map[uint16]*api.RoleInfra {
	return map[uint16]*api.RoleInfra{
		api.Master: {
			Softwares: []*api.PackageConfig{},
			OpenPorts: MasterPorts,
		},
		api.Worker: {
			Softwares: []*api.PackageConfig{},
			OpenPorts: WorkerPorts,
		},
		api.ETCD: {
			Softwares: []*api.PackageConfig{},
			OpenPorts: EtcdPorts,
		},
		api.LoadBalance: {
			Softwares: []*api.PackageConfig{},
			OpenPorts: []*api.OpenPorts{},
		},
	}
}
