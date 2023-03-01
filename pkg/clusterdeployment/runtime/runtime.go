package runtime

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/commontools"
	"isula.org/eggo/pkg/utils/dependency"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/template"
)

var (
	mapRuntime = map[string]Runtime{
		"isulad":     &isuladRuntime{},
		"docker":     &dockerRuntime{},
		"containerd": &containerdRuntime{},
	}
)

type Runtime interface {
	GetRuntimeSoftwares() []string
	GetRuntimeClient() string
	GetRuntimeLoadImageCommand() string
	GetRuntimeService() string
	PrepareRuntimeService(r runner.Runner, workerConfig *api.WorkerConfig) error

	GetRemovedPath() []string
}

type isuladRuntime struct {
}

func (ir *isuladRuntime) GetRuntimeSoftwares() []string {
	return []string{"isula", "isulad"}
}

func (ir *isuladRuntime) GetRuntimeClient() string {
	return "isula"
}

func (ir *isuladRuntime) GetRuntimeLoadImageCommand() string {
	return "isula load -i"
}

func (ir *isuladRuntime) GetRuntimeService() string {
	return "isulad"
}

func (ir *isuladRuntime) PrepareRuntimeService(r runner.Runner, workerConfig *api.WorkerConfig) error {
	service := `[Unit]
Description=iSulad Application Container Engine
After=network.target

[Service]
Type=notify
EnvironmentFile=-/etc/sysconfig/iSulad
ExecStart=/usr/bin/isulad \
        --pod-sandbox-image {{ .pauseImage }} \
        --network-plugin cni \
        --cni-bin-dir {{ .cniBinDir }} \
        --cni-conf-dir {{ .cniConfDir }} \
{{- range $i, $v := .registry }}
        --registry-mirrors {{ $v }} \
{{- end }}
{{- range $i, $v := .insecure }}
        --insecure-registry {{ $v }} \
{{- end }}
{{- range $i, $v := .addition }}
        {{ .addition }} \
{{- end }}
        $OPTIONS
ExecReload=/bin/kill -s HUP $MAINPID
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TimeoutStartSec=0
Delegate=yes
KillMode=process
Restart=on-failure
StartLimitBurst=3
StartLimitInterval=60s
TimeoutStopSec=10

[Install]
WantedBy=multi-user.target
`

	pauseImage, cniBinDir, cniConfDir := "k8s.gcr.io/pause:3.2", "/opt/cni/bin", "/etc/cni/net.d"
	registry := []string{"docker.io"}
	insecure := []string{"quay.io", "k8s.gcr.io"}
	addition := []string{}

	if workerConfig.KubeletConf.PauseImage != "" {
		pauseImage = workerConfig.KubeletConf.PauseImage
	}
	if workerConfig.KubeletConf.CniBinDir != "" {
		cniBinDir = workerConfig.KubeletConf.CniBinDir
	}
	if workerConfig.KubeletConf.CniConfDir != "" {
		cniConfDir = workerConfig.KubeletConf.CniConfDir
	}
	if len(workerConfig.ContainerEngineConf.RegistryMirrors) != 0 || len(workerConfig.ContainerEngineConf.InsecureRegistries) != 0 {
		registry = workerConfig.ContainerEngineConf.RegistryMirrors
		insecure = workerConfig.ContainerEngineConf.InsecureRegistries
	}
	for k, v := range workerConfig.ContainerEngineConf.ExtraArgs {
		addition = append(addition, fmt.Sprintf("%s=%s", k, v))
	}

	datastore := map[string]interface{}{}
	datastore["pauseImage"] = pauseImage
	datastore["cniBinDir"] = cniBinDir
	datastore["cniConfDir"] = cniConfDir
	datastore["registry"] = registry
	datastore["insecure"] = insecure
	datastore["addition"] = addition
	serviceConf, err := template.TemplateRender(service, datastore)
	if err != nil {
		return err
	}

	serviceBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	shell, err := commontools.GetSystemdServiceShell("isulad", serviceBase64, true)
	if err != nil {
		logrus.Errorf("get isulad systemd service file failed: %v", err)
		return err
	}

	_, err = r.RunShell(shell, "isuladService")
	if err != nil {
		logrus.Errorf("create isulad service failed: %v", err)
		return err
	}
	return nil
}

func (ir *isuladRuntime) GetRemovedPath() []string {
	return []string{
		"/usr/lib/systemd/system/isulad.service",
	}
}

type dockerRuntime struct {
}

func (dr *dockerRuntime) GetRuntimeSoftwares() []string {
	return []string{"docker", "dockerd"}
}

func (dr *dockerRuntime) GetRuntimeClient() string {
	return "docker"
}

func (dr *dockerRuntime) GetRuntimeLoadImageCommand() string {
	return "docker load -i"
}

func (dr *dockerRuntime) GetRuntimeService() string {
	return "docker"
}

func (dr *dockerRuntime) PrepareRuntimeService(r runner.Runner, workerConfig *api.WorkerConfig) error {
	service := `[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
After=network.target

[Service]
Type=notify
EnvironmentFile=-/etc/sysconfig/docker
ExecStart=/usr/bin/dockerd \
{{- range $i, $v := .registry }}
        --registry-mirror {{ $v }} \
{{- end }}
{{- range $i, $v := .insecure }}
        --insecure-registry {{ $v }} \
{{- end }}
{{- range $i, $v := .addition }}
        {{ .addition }} \
{{- end }}
        $OPTIONS
ExecReload=/bin/kill -s HUP $MAINPID
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
# Uncomment TasksMax if your systemd version supports it.
# Only systemd 226 and above support this version.
#TasksMax=infinity
TimeoutStartSec=0
# set delegate yes so that systemd does not reset the cgroups of docker containers
Delegate=yes
# kill only the docker process, not all processes in the cgroup
KillMode=process

[Install]
WantedBy=multi-user.target
`

	registry := workerConfig.ContainerEngineConf.RegistryMirrors
	insecure := workerConfig.ContainerEngineConf.InsecureRegistries
	addition := []string{}
	for k, v := range workerConfig.ContainerEngineConf.ExtraArgs {
		addition = append(addition, fmt.Sprintf("%s=%s", k, v))
	}

	datastore := map[string]interface{}{}
	datastore["registry"] = registry
	datastore["insecure"] = insecure
	datastore["addition"] = addition
	serviceConf, err := template.TemplateRender(service, datastore)
	if err != nil {
		return err
	}

	serviceBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConf))
	shell, err := commontools.GetSystemdServiceShell("docker", serviceBase64, true)
	if err != nil {
		logrus.Errorf("get docker systemd service file failed: %v", err)
		return err
	}

	_, err = r.RunShell(shell, "dockerService")
	if err != nil {
		logrus.Errorf("create docker service failed: %v", err)
		return err
	}
	return nil
}

func (dr *dockerRuntime) GetRemovedPath() []string {
	return []string{
		"/usr/lib/systemd/system/docker.service",
	}
}

type containerdRuntime struct {
}

func (cr *containerdRuntime) GetRuntimeSoftwares() []string {
	return []string{"ctr", "containerd"}
}

func (cr *containerdRuntime) GetRuntimeClient() string {
	return "ctr"
}

func (cr *containerdRuntime) GetRuntimeLoadImageCommand() string {
	return "ctr cri load"
}

func (cr *containerdRuntime) GetRuntimeService() string {
	return "containerd"
}

func (cr *containerdRuntime) PrepareRuntimeService(r runner.Runner, workerConfig *api.WorkerConfig) error {
	if err := prepareContainerdConfig(r, workerConfig); err != nil {
		return err
	}

	service := `[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/bin/containerd 

Type=notify
Delegate=yes
KillMode=process
Restart=always
RestartSec=5
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNPROC=infinity
LimitCORE=infinity
LimitNOFILE=infinity
# Comment TasksMax if your systemd version does not supports it.
# Only systemd 226 and above support this version.
TasksMax=infinity
OOMScoreAdjust=-999

[Install]
WantedBy=multi-user.target
`

	serviceBase64 := base64.StdEncoding.EncodeToString([]byte(service))
	shell, err := commontools.GetSystemdServiceShell("containerd", serviceBase64, true)
	if err != nil {
		logrus.Errorf("get containerd systemd service file failed: %v", err)
		return err
	}

	_, err = r.RunShell(shell, "containerdService")
	if err != nil {
		logrus.Errorf("create containerd service failed: %v", err)
		return err
	}
	return nil
}

func (cr *containerdRuntime) GetRemovedPath() []string {
	return []string{
		"/usr/lib/systemd/system/containerd.service",
		"/etc/containerd/config.toml",
	}
}

func prepareContainerdConfig(r runner.Runner, workerConfig *api.WorkerConfig) error {
	containerdConfig := `
[plugins.cri]
  sandbox_image = "{{ .pauseImage }}"
{{- $alen := len .registryAggregate }}
{{- if ne $alen 0 }}
[plugins."io.containerd.grpc.v1.cri".registry]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
{{- range $i, $v := .registryAggregate }}
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."{{ $v }}"]
      endpoint = ["https://{{ $v }}"]
{{- end }}
{{- end }}
{{- $alen := len .insecure }}
{{- if ne $alen 0 }}
  [plugins."io.containerd.grpc.v1.cri".registry.configs]
{{- range $i, $v := .insecure }}
    [plugins."io.containerd.grpc.v1.cri".registry.configs."{{ $v }}".tls]
      insecure_skip_verify = true
{{- end }}
{{- end }}
{{- range $i, $v := .addition }}
{{ .addition }}
{{- end }}
`

	pauseImage := "k8s.gcr.io/pause:3.2"
	registry := []string{"docker.io"}
	insecure := []string{"quay.io", "k8s.gcr.io"}
	addition := []string{}

	if workerConfig.KubeletConf.PauseImage != "" {
		pauseImage = workerConfig.KubeletConf.PauseImage
	}
	if len(workerConfig.ContainerEngineConf.RegistryMirrors) != 0 || len(workerConfig.ContainerEngineConf.InsecureRegistries) != 0 {
		registry = workerConfig.ContainerEngineConf.RegistryMirrors
		insecure = workerConfig.ContainerEngineConf.InsecureRegistries
	}
	for k, v := range workerConfig.ContainerEngineConf.ExtraArgs {
		addition = append(addition, fmt.Sprintf("%s = %s", k, v))
	}

	registryAggregate := []string{}
	insecureTmp := []string{}
	for _, r := range registry {
		deflash := strings.TrimPrefix(strings.TrimPrefix(r, "http://"), "https://")
		registryAggregate = append(registryAggregate, deflash)
	}
	for _, i := range insecure {
		deflash := strings.TrimPrefix(strings.TrimPrefix(i, "http://"), "https://")
		registryAggregate = append(registryAggregate, deflash)
		insecureTmp = append(insecureTmp, deflash)
	}

	datastore := map[string]interface{}{}
	datastore["pauseImage"] = pauseImage
	datastore["registryAggregate"] = registryAggregate
	datastore["insecure"] = insecureTmp
	datastore["addition"] = addition
	containerdConf, err := template.TemplateRender(containerdConfig, datastore)
	if err != nil {
		return err
	}

	var sb strings.Builder
	containerdBase64 := base64.StdEncoding.EncodeToString([]byte(containerdConf))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p /etc/containerd && echo %s | base64 -d > %s\"",
		containerdBase64, "/etc/containerd/config.toml"))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}

	return nil
}

type DeployRuntimeTask struct {
	runtime      Runtime
	workerConfig *api.WorkerConfig
	workerInfra  *api.RoleInfra
	packageSrc   *api.PackageSrcConfig
}

func NewDeployRuntimeTask(ccfg *api.ClusterConfig) *DeployRuntimeTask {
	return &DeployRuntimeTask{
		workerConfig: &ccfg.WorkerConfig,
		workerInfra:  ccfg.RoleInfra[api.Worker],
		packageSrc:   &ccfg.PackageSrc,
	}
}

func (ct *DeployRuntimeTask) Name() string {
	return "DeployRuntimeTask"
}

func (ct *DeployRuntimeTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("do deploy container engine...\n")

	ct.runtime = GetRuntime(ct.workerConfig.ContainerEngineConf.Runtime)
	if ct.runtime == nil {
		return fmt.Errorf("unsupport container engine %s", ct.workerConfig.ContainerEngineConf.Runtime)
	}

	// check container engine softwares
	if err := ct.check(r); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if _, err := r.RunCommand("sudo -E /bin/sh -c \"rm -rf /etc/docker/daemon.json\""); err != nil {
		logrus.Errorf("rm docker daemon.json failed: %v", err)
		return err
	}

	if err := ct.runtime.PrepareRuntimeService(r, ct.workerConfig); err != nil {
		logrus.Errorf("prepare container engine service failed: %v", err)
		return err
	}

	if err := dependency.InstallImageDependency(r, ct.workerInfra, ct.packageSrc, ct.runtime.GetRuntimeService(),
		ct.runtime.GetRuntimeClient(), ct.runtime.GetRuntimeLoadImageCommand()); err != nil {
		logrus.Errorf("load images failed: %v", err)
		return err
	}

	logrus.Info("deploy container engine success\n")
	return nil
}

func (ct *DeployRuntimeTask) check(r runner.Runner) error {
	if ct.workerConfig == nil {
		return fmt.Errorf("empty worker config")
	}
	if ct.workerInfra == nil {
		return fmt.Errorf("empty worker infra")
	}

	if err := dependency.CheckDependency(r, ct.runtime.GetRuntimeSoftwares()); err != nil {
		return err
	}

	return nil
}

func GetRuntime(runtime string) Runtime {
	if runtime == "" {
		return mapRuntime["docker"]
	}

	return mapRuntime[strings.ToLower(runtime)]
}
