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
 * Author: wangfengtu
 * Create: 2021-06-21
 * Description: provide kubectl functions
 ******************************************************************************/
package kubectl

import (
	"fmt"
	"path/filepath"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/constants"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/template"
)

var ops map[string]string

const (
	ApplyOpKey  = "apply"
	DeleteOpKey = "delete"
)

func init() {
	ops = map[string]string{
		ApplyOpKey:  "apply",
		DeleteOpKey: "delete",
	}
}

func runKubectlWithYaml(r runner.Runner, operator string, yamlFile string, cluster *api.ClusterConfig) error {
	yamlTmpl := `
#!/bin/bash
export KUBECONFIG={{ .KubeConfig }}
kubectl {{ .Operator }} -f {{ .Yaml }}
if [ $? -ne 0 ]; then
	echo "{{ .Operator }} {{ .Yaml }} failed"
	exit 1
fi
exit 0
`

	datastore := make(map[string]interface{})
	datastore["Operator"] = operator
	datastore["Yaml"] = yamlFile
	datastore["KubeConfig"] = filepath.Join(cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	cmdStr, err := template.TemplateRender(yamlTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, filepath.Base(yamlFile))
	if err != nil {
		return err
	}

	return nil
}

func OperatorByYaml(r runner.Runner, operator string, yamlFile string, cluster *api.ClusterConfig) error {
	if op, ok := ops[operator]; ok {
		return runKubectlWithYaml(r, op, yamlFile, cluster)
	}
	return fmt.Errorf("unsupport operator: %s", operator)
}

func RunKubectlCmd(r runner.Runner, subcmd string, cluster *api.ClusterConfig) error {
	cmdTmpl := `
#!/bin/bash
export KUBECONFIG={{ .KubeConfig }}
kubectl {{ .Subcmd }}
if [ $? -ne 0 ]; then
	echo "run {{ .Subcmd }} failed"
	exit 1
fi
exit 0
`
	datastore := make(map[string]interface{})
	datastore["Subcmd"] = subcmd
	datastore["KubeConfig"] = filepath.Join(cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	cmdStr, err := template.TemplateRender(cmdTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "kubectlcmd")
	if err != nil {
		return err
	}

	return nil
}

func WaitNodeJoined(r runner.Runner, name string, cluster *api.ClusterConfig) error {
	waitTmpl := `
#!/bin/bash
export KUBECONFIG={{ .KubeConfig }}
for i in $(seq 20); do
	kubectl get nodes | grep {{ .Name }}
	if [ $? -eq 0 ]; then
		exit 0
	fi
	sleep 6
done
echo "wait node: {{ .Name }} join failed"
exit 1
`
	datastore := make(map[string]interface{})
	datastore["Name"] = name
	datastore["KubeConfig"] = filepath.Join(cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	cmdStr, err := template.TemplateRender(waitTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, name)
	if err != nil {
		return err
	}

	return nil
}
