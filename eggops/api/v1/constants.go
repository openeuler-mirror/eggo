package v1

const (
	MachineUsageMaster = "master machine"
	MachineUsageWorker = "worker machine"
	MachineUsageEtcd   = "etcd machine"
	MachineUsageLB     = "loadbalance machine"
)

const (
	ImageVersion string = "1.0.0-alpha"

	ClusterConfigMapNameFormat    string = "eggo-cluster-%s-%s"
	ClusterConfigMapBinaryConfKey string = "eggo-binary-config"

	EggoConfigVolumeFormat string = "/%s-config"
	PrivateKeyVolumeFormat string = "/%s-privatekey"
	PackageVolumeFormat    string = "/%s-package"

	DefaultPackageArmName   string = "packages-arm.tar.gz"
	DefaultPackageX86Name   string = "packages-x86.tar.gz"
	DefaultPackageRISCVName string = "packages-risc-v.tar.gz"
)
