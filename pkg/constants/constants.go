package constants

const (
	// certificates relate constants
	DefaultK8SRootDir      = "/etc/kubernetes"
	DefaultK8SCertDir      = "/etc/kubernetes/pki"
	DefaultK8SManifestsDir = "/etc/kubernetes/manifests"
	DefaultK8SAddonsDir    = "/etc/kubernetes/addons"

	KubeConfigFileNameAdmin      = "admin.conf"
	KubeConfigFileNameController = "controller-manager.conf"
	KubeConfigFileNameScheduler  = "scheduler.conf"
	EncryptionConfigName         = "encryption-config.yaml"

	// package manager relate constants
	TempFileHomeDir     = "/tmp/.eggo"
	DefaultPkgUntarPath = "/tmp/.eggo/pkg"
	DefaultImagePkgName = "images.tar"
	DefaultUserTempDir  = "/tmp/.eggo/temp"
)
