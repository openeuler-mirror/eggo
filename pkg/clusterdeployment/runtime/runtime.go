package runtime

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/clusterdeployment/binary/infrastructure"
	"gitee.com/openeuler/eggo/pkg/constants"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/template"
	"github.com/sirupsen/logrus"
)

var (
	mapRuntime = map[string]Runtime{
		"isulad": &isuladRuntime{},
		"docker": &dockerRuntime{},
	}
)

type Runtime interface {
	GetRuntimeSoftwares() []string
	GetRuntimeClient() string
	GetRuntimeService() string
	PrepareRuntimeJson(r runner.Runner, ccfg *api.ClusterConfig) error
}

type isuladRuntime struct {
}

func (ir *isuladRuntime) GetRuntimeSoftwares() []string {
	return []string{"isula", "isulad"}
}

func (ir *isuladRuntime) GetRuntimeClient() string {
	return "isula"
}

func (ir *isuladRuntime) GetRuntimeService() string {
	return "isulad"
}

func (ir *isuladRuntime) PrepareRuntimeJson(r runner.Runner, ccfg *api.ClusterConfig) error {
	pauseImage, cniBinDir := "k8s.gcr.io/pause:3.2", "/usr/libexec/cni"
	registry := []string{"docker.io"}
	insecure := []string{"quay.io", "k8s.gcr.io"}

	if ccfg.WorkerConfig.KubeletConf.PauseImage != "" {
		pauseImage = ccfg.WorkerConfig.KubeletConf.PauseImage
	}
	if ccfg.WorkerConfig.KubeletConf.CniBinDir != "" {
		cniBinDir = ccfg.WorkerConfig.KubeletConf.CniBinDir
	}
	if len(ccfg.WorkerConfig.ContainerEngineConf.RegistryMirrors) != 0 || len(ccfg.WorkerConfig.ContainerEngineConf.InsecureRegistries) != 0 {
		registry = ccfg.WorkerConfig.ContainerEngineConf.RegistryMirrors
		insecure = ccfg.WorkerConfig.ContainerEngineConf.InsecureRegistries
	}

	isuladConfig := `
#!/bin/bash
sed -i "s#network-plugin\": \"#network-plugin\": \"cni#g" /etc/isulad/daemon.json
sed -i "s#pod-sandbox-image\": \"#pod-sandbox-image\": \"{{ .pauseImage }}#g" /etc/isulad/daemon.json
sed -i "s#cni-bin-dir\": \"#cni-bin-dir\": \"{{ .cniBinDir }}#g" /etc/isulad/daemon.json
{{- range $i, $v := .registry }}
sed -i "/registry-mirrors/a\    \t\"{{ $v }}\"{{ if ne $i 0 }},{{ end }}" /etc/isulad/daemon.json
{{- end }}
{{- range $i, $v := .insecure }}
sed -i "/insecure-registries/a\    \t\"{{ $v }}\"{{ if ne $i 0 }},{{ end }}" /etc/isulad/daemon.json
{{- end }}
`
	datastore := map[string]interface{}{}
	datastore["pauseImage"] = pauseImage
	datastore["cniBinDir"] = cniBinDir
	datastore["registry"] = registry
	datastore["insecure"] = insecure
	shell, err := template.TemplateRender(isuladConfig, datastore)
	if err != nil {
		return err
	}
	output, err := r.RunShell(shell, "isuladShell")
	if err != nil {
		logrus.Errorf("modify isulad daemon.json failed: %v", err)
		return err
	}
	logrus.Debugf("modify isulad daemon.json success: %s", output)

	return nil
}

type dockerRuntime struct {
}

func (dr *dockerRuntime) GetRuntimeSoftwares() []string {
	return []string{"docker", "dockerd"}
}

func (dr *dockerRuntime) GetRuntimeClient() string {
	return "docker"
}

func (dr *dockerRuntime) GetRuntimeService() string {
	return "docker"
}

func (dr *dockerRuntime) PrepareRuntimeJson(r runner.Runner, ccfg *api.ClusterConfig) error {
	registry := ccfg.WorkerConfig.ContainerEngineConf.RegistryMirrors
	insecure := ccfg.WorkerConfig.ContainerEngineConf.InsecureRegistries

	if len(registry) == 0 && len(insecure) == 0 {
		return nil
	}

	dockerConfig := `
{
{{- if .registry}}
    {{- $alen := len .registry }}
    "registry-mirrors": [
        {{- range $i, $v := .registry }}
        "{{ $v }}"{{ if NotLast $i $alen }},{{ end }}
        {{- end }}
    ]{{- if .insecure }},{{- end }}
{{- end }}
{{- if .insecure}}
    {{- $alen := len .insecure }}
    "insecure-registries": [
        {{- range $i, $v := .insecure }}
        "{{ $v }}"{{ if NotLast $i $alen }},{{ end }}
        {{- end }}
    ]
{{- end }}
}
`

	datastore := map[string]interface{}{}
	datastore["registry"] = registry
	datastore["insecure"] = insecure
	json, err := template.TemplateRender(dockerConfig, datastore)
	if err != nil {
		return err
	}

	var sb strings.Builder
	jsonBase64 := base64.StdEncoding.EncodeToString([]byte(json))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", jsonBase64, "/etc/docker/daemon.json"))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}

	logrus.Debugf("write docker daemon.json success")

	return nil
}

type DeployRuntimeTask struct {
	ccfg    *api.ClusterConfig
	runtime Runtime
}

func NewDeployRuntimeTask(ccfg *api.ClusterConfig) *DeployRuntimeTask {
	return &DeployRuntimeTask{
		ccfg: ccfg,
	}
}

func (ct *DeployRuntimeTask) Name() string {
	return "DeployRuntimeTask"
}

func (ct *DeployRuntimeTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("do deploy container engine...\n")

	ct.runtime = GetRuntime(ct.ccfg.WorkerConfig.ContainerEngineConf.Runtime)
	if ct.runtime == nil {
		return fmt.Errorf("unsupport container engine %s", ct.ccfg.WorkerConfig.ContainerEngineConf.Runtime)
	}

	// check container engine softwares
	if err := ct.check(r); err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if err := ct.runtime.PrepareRuntimeJson(r, ct.ccfg); err != nil {
		logrus.Errorf("prepare container engine json failed: %v", err)
		return err
	}

	// start service
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"systemctl restart %s\"", ct.runtime.GetRuntimeService())); err != nil {
		logrus.Errorf("start %s failed: %v", ct.runtime.GetRuntimeService(), err)
		return err
	}

	if err := loadImages(r, &ct.ccfg.PackageSrc, ct.runtime.GetRuntimeClient()); err != nil {
		logrus.Errorf("load images failed: %v", err)
		return err
	}

	logrus.Info("deploy container engine success\n")
	return nil
}

func (ct *DeployRuntimeTask) check(r runner.Runner) error {
	if err := infrastructure.CheckDependences(r, ct.runtime.GetRuntimeSoftwares()); err != nil {
		return err
	}

	return nil
}

func loadImages(r runner.Runner, conf *api.PackageSrcConfig, client string) error {
	logrus.Info("do load images...")
	if conf == nil {
		return fmt.Errorf("can not found dist path failed")
	}

	imagePkgPath := filepath.Join(conf.GetPkgDistPath(), constants.DefaultImagePkgName)

	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"stat %s\"", imagePkgPath)); err != nil {
		logrus.Debugf("no image package found on path %v", imagePkgPath)
		return nil
	}

	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s load -i %s\"", client, imagePkgPath)); err != nil {
		return fmt.Errorf("isula load -i %v failed: %v", imagePkgPath, err)
	}

	logrus.Info("load images success")

	return nil
}

func IsISulad(engine string) bool {
	return strings.ToLower(engine) == "isulad"
}

func IsDocker(engine string) bool {
	// default engine
	return engine == "" || strings.ToLower(engine) == "docker"
}

func GetRuntime(runtime string) Runtime {
	if runtime == "" {
		return mapRuntime["docker"]
	}

	return mapRuntime[strings.ToLower(runtime)]
}
