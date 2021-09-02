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
	DefaultPackagePath = "/root/.eggo/package"
	DefaultPkgPath     = "/pkg"
	DefaultBinPath     = "/bin"
	DefaultFilePath    = "/file"
	DefaultDirPath     = "/dir"
	DefaultImagePath   = "/image"

	// user home dir formats
	UserHomeFormat                = "/home/%s"
	DefaultUserCopyTempHomeFormat = "/home/%s/.eggo"
	DefaultRootCopyTempDirHome    = "/root/.eggo"

	// network plugin arguments key
	NetworkPluginArgKeyYamlPath = "NetworkYamlPath"
)
