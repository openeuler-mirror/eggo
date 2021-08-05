package v1

const (
	EggoNamespaceName = "eggo-system"

	MachineUsageMaster = "master machine"
	MachineUsageWorker = "worker machine"
	MachineUsageEtcd   = "etcd machine"
	MachineUsageLB     = "loadbalance machine"
)

const (
	ImageVersion string = "0.9.1"

	ClusterConfigMapNameFormat    string = "eggo-cluster-%s-%s"
	ClusterConfigMapBinaryConfKey string = "eggo-binary-config"
)
