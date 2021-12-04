/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2021. All rights reserved.
 * eggo licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: zhangxiaoyu
 * Create: 2021-06-25
 * Description: eggo install implement
 ******************************************************************************/

package dependency

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/template"
)

const (
	PrmTest = "if [ x != x$(which apt 2>/dev/null) ]; then echo apt ; elif [ x != x$(which yum 2>/dev/null) ]; then echo yum ; fi"
	PmTest  = "if [ x != x$(which dpkg 2>/dev/null) ]; then echo dpkg ; elif [ x != x$(which rpm 2>/dev/null) ]; then echo rpm ; fi"
)

type managerCommand struct {
	installCommand string
	removeCommand  string
}

func getPackageRepoManager(r runner.Runner) (*managerCommand, error) {
	packageRepoManagerCommand := map[string]*managerCommand{
		"apt": {
			installCommand: "apt install -y",
			removeCommand:  "apt remove -y",
		},
		"yum": {
			installCommand: "yum install -y",
			removeCommand:  "yum remove -y",
		},
	}

	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", PrmTest))
	if err != nil {
		logrus.Errorf("package repo manager test failed: %v", err)
		return nil, err
	}

	if strings.Contains(output, "apt") {
		return packageRepoManagerCommand["apt"], nil
	}
	if strings.Contains(output, "yum") {
		return packageRepoManagerCommand["yum"], nil
	}

	return nil, fmt.Errorf("invalid package repo manager %s", output)
}

func getPackageManager(r runner.Runner) (*managerCommand, error) {
	packageManagerCommand := map[string]*managerCommand{
		"dpkg": {
			installCommand: "dpkg --force-all -i",
			removeCommand:  "apt remove -y",
		},
		"rpm": {
			installCommand: "rpm -ivh --force --nodeps",
			removeCommand:  "yum remove -y",
		},
	}

	output, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s\"", PmTest))
	if err != nil {
		logrus.Errorf("package manager test failed: %v", err)
		return nil, err
	}

	if strings.Contains(output, "dpkg") {
		return packageManagerCommand["dpkg"], nil
	}
	if strings.Contains(output, "rpm") {
		return packageManagerCommand["rpm"], nil
	}

	return nil, fmt.Errorf("invalid package manager %s", output)
}

type dependency interface {
	Install(r runner.Runner) error
	Remove(r runner.Runner) error
}

type dependencyRepo struct {
	software []*api.PackageConfig
}

func (dr *dependencyRepo) Install(r runner.Runner) error {
	if len(dr.software) == 0 {
		return nil
	}

	prManager, err := getPackageRepoManager(r)
	if err != nil {
		return err
	}

	join := ""
	for _, s := range dr.software {
		join += s.Name + " "
	}
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s %s\"", prManager.installCommand, join)); err != nil {
		return fmt.Errorf("%s failed: %v", prManager.installCommand, err)
	}

	return nil
}

func (dr *dependencyRepo) Remove(r runner.Runner) error {
	if len(dr.software) == 0 {
		return nil
	}

	prManager, err := getPackageRepoManager(r)
	if err != nil {
		return err
	}

	join := ""
	for _, s := range dr.software {
		join += s.Name + " "
	}
	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"%s remove -y %s\"", prManager.removeCommand, join)); err != nil {
		return fmt.Errorf("%s failed: %v", prManager.removeCommand, err)
	}

	return nil
}

type dependencyPkg struct {
	srcPath  string
	software []*api.PackageConfig
}

func (dp *dependencyPkg) Install(r runner.Runner) error {
	if len(dp.software) == 0 {
		return nil
	}

	pManager, err := getPackageManager(r)
	if err != nil {
		return err
	}

	join := ""
	for _, s := range dp.software {
		join += s.Name + "* "
	}

	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"cd %s && %s %s",
		dp.srcPath, pManager.installCommand, join)); err != nil {
		return fmt.Errorf("%s failed: %v", pManager.installCommand, err)
	}

	return nil
}

func (dp *dependencyPkg) Remove(r runner.Runner) error {
	if len(dp.software) == 0 {
		return nil
	}

	pManager, err := getPackageManager(r)
	if err != nil {
		return err
	}

	join := ""
	for _, s := range dp.software {
		join += s.Name + "* "
	}

	if _, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"cd %s && %s %s",
		dp.srcPath, pManager.removeCommand, join)); err != nil {
		return fmt.Errorf("%s remove failed: %v", pManager.removeCommand, err)
	}

	return nil
}

// install file and dir
type dependencyFileDir struct {
	executable bool
	srcPath    string
	software   []*api.PackageConfig
}

func (df *dependencyFileDir) Install(r runner.Runner) error {
	if len(df.software) == 0 {
		return nil
	}

	shell := `
#!/bin/bash
cd {{ .srcPath }}

{{- if .executable }}
{{- range $i, $v := .software }}
chmod +x {{ $v.Name }}
{{- end }}
{{- end }}

{{- range $i, $v := .software }}
if [ ! -e {{ JoinPath $v.Dst $v.Name }} ]; then
    mkdir -p {{ $v.Dst }} && cp -r {{ $v.Name }} {{ $v.Dst }}
fi
{{- end }}
`
	datastore := make(map[string]interface{})
	datastore["srcPath"] = df.srcPath
	datastore["software"] = df.software
	datastore["executable"] = df.executable

	shellStr, err := template.TemplateRender(shell, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(shellStr, "install_FD")
	if err != nil {
		return err
	}

	return nil
}

func (df *dependencyFileDir) Remove(r runner.Runner) error {
	if len(df.software) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for _, s := range df.software {
		path := fmt.Sprintf("%s/%s", s.Dst, s.Name)
		if !CheckPath(path) {
			return fmt.Errorf("path %s not in White List and cannot remove", path)
		}

		sb.WriteString(fmt.Sprintf("rm -rf %s ; ", path))
	}
	sb.WriteString("\"")

	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("rm dependency failed: %v", err)
	}

	return nil
}

type dependencyImage struct {
	srcPath string
	client  string
	command string
	image   []*api.PackageConfig
}

func (di *dependencyImage) Install(r runner.Runner) error {
	if len(di.image) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	for _, i := range di.image {
		sb.WriteString(fmt.Sprintf("%s %s/%s && ", di.command, di.srcPath, i.Name))
	}
	sb.WriteString("echo success\"")

	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("%s load image failed: %v", di.client, err)
	}

	return nil
}

func (di *dependencyImage) Remove(r runner.Runner) error {
	// nothing to do
	return nil
}

type dependencyYaml struct {
	srcPath    string
	kubeconfig string
	yaml       []*api.PackageConfig
}

func NewDependencyYaml(srcPath, kubeconfig string, yaml []*api.PackageConfig) *dependencyYaml {
	return &dependencyYaml{
		srcPath:    srcPath,
		kubeconfig: kubeconfig,
		yaml:       yaml,
	}
}

func (dy *dependencyYaml) Install(r runner.Runner) error {
	if len(dy.yaml) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"export KUBECONFIG=%s ", dy.kubeconfig))
	for _, y := range dy.yaml {
		if strings.HasPrefix(y.Name, "http://") || strings.HasPrefix(y.Name, "https://") {
			sb.WriteString(fmt.Sprintf("&& kubectl apply -f %s ", y.Name))
			continue
		}
		sb.WriteString(fmt.Sprintf("&& kubectl apply -f %s/%s ", dy.srcPath, y.Name))
	}
	sb.WriteString("\"")

	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("kubectl apply yaml failed: %v", err)
	}

	return nil
}

func (dy *dependencyYaml) Remove(r runner.Runner) error {
	if len(dy.yaml) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"export KUBECONFIG=%s ", dy.kubeconfig))
	for _, y := range dy.yaml {
		if strings.HasPrefix(y.Name, "http://") || strings.HasPrefix(y.Name, "https://") {
			sb.WriteString(fmt.Sprintf("&& kubectl delete -f %s ", y.Name))
			continue
		}
		sb.WriteString(fmt.Sprintf("&& kubectl delete -f %s/%s ", dy.srcPath, y.Name))
	}
	sb.WriteString("\"")

	if _, err := r.RunCommand(sb.String()); err != nil {
		return fmt.Errorf("kubectl delete yaml failed: %v", err)
	}

	return nil
}

type dependencyShell struct {
	envs    []string
	srcPath string
	shell   []*api.PackageConfig
}

func NewDependencyShell(srcPath string, shell []*api.PackageConfig) *dependencyShell {
	return &dependencyShell{
		srcPath: srcPath,
		shell:   shell,
	}
}

func (ds *dependencyShell) Install(r runner.Runner) error {
	if len(ds.shell) == 0 {
		return nil
	}

	shellTemplate := `
#!/bin/bash
{{- range $i, $v := .Envs }}
export {{ $v }}
{{- end }}

{{- $tout := .Timeouts }}
{{- range $i, $v := .Shells }}
chmod +x {{ $v }} && timeout -s SIGKILL {{index $tout $i}} {{ $v }} > /dev/null
if [ $? -ne 0 ]; then
	echo "run {{ $v }} failed"
	exit 1
fi
{{- end }}

exit 0
`
	datastore := map[string]interface{}{}
	datastore["Envs"] = ds.envs
	var shells []string
	var timeouts []string
	for _, s := range ds.shell {
		shells = append(shells, fmt.Sprintf("%s/%s", ds.srcPath, s.Name))
		timeout := s.TimeOut
		if timeout == "" {
			timeout = "30s"
		}
		timeouts = append(timeouts, timeout)
	}
	datastore["Shells"] = shells
	datastore["Timeouts"] = timeouts

	parsedShell, err := template.TemplateRender(shellTemplate, datastore)
	if err != nil {
		return err
	}

	if _, err := r.RunShell(parsedShell, "exechook"); err != nil {
		return fmt.Errorf("hook execute failed: %v", err)
	}

	return nil
}

func (ds *dependencyShell) Remove(r runner.Runner) error {
	// nothing to do
	return nil
}

type DependencyTask struct {
	dp dependency
}

func (dt *DependencyTask) Name() string {
	return "DependencyTask"
}

func (dt *DependencyTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	if err := dt.dp.Install(r); err != nil {
		logrus.Errorf("install failed for %s: %v", hcf.Address, err)
		return err
	}

	return nil
}
