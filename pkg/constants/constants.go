package constants

import "os"

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
	DefaultHookPath    = "/file/cmdhook"
	DefaultDirPath     = "/dir"
	DefaultImagePath   = "/image"

	// user home dir formats
	UserHomeFormat                = "/home/%s"
	DefaultUserCopyTempHomeFormat = "/home/%s/.eggo"
	DefaultRootCopyTempDirHome    = "/root/.eggo"

	// network plugin arguments key
	NetworkPluginArgKeyYamlPath = "NetworkYamlPath"

	MaxHookFileSize = int64(1 << 20)

	HookFileMode             os.FileMode = 0750
	EggoHomeDirMode          os.FileMode = 0750
	EggoDirMode              os.FileMode = 0700
	DeployConfigFileMode     os.FileMode = 0640
	ProcessFileMode          os.FileMode = 0640
	EncryptionConfigFileMode os.FileMode = 0600

	// default task wait time in minute
	DefaultTaskWaitMinutes = 5
)
