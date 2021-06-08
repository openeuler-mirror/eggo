package bootstrap

import (
	"encoding/base64"
	"fmt"
	"strings"

	"gitee.com/openeuler/eggo/pkg/api"
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
	PrepareRuntimeJson(r runner.Runner, ccfg *api.ClusterConfig) error
}

type isuladRuntime struct {
}

func (ir *isuladRuntime) GetRuntimeSoftwares() []string {
	return []string{"isula", "isulad"}
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
	if len(ccfg.WorkerConfig.ContainerEngineConf.RegistryMirrors) != 0 {
		registry = ccfg.WorkerConfig.ContainerEngineConf.RegistryMirrors
	}
	if len(ccfg.WorkerConfig.ContainerEngineConf.InsecureRegistries) != 0 {
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
systemctl restart isulad
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

func GetRuntime(runtime string) Runtime {
	if runtime == "" {
		return mapRuntime["docker"]
	}

	return mapRuntime[strings.ToLower(runtime)]
}
