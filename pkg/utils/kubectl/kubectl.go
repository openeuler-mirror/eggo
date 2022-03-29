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
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/template"
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
	echo "{{ .Operator }} {{ .Yaml }} failed" 1>&2
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

	splits := strings.Split(filepath.Base(yamlFile), ".")
	name := "defaultshell"
	if len(splits) > 0 {
		name = splits[0]
	}
	_, err = r.RunShell(cmdStr, name)
	if err != nil {
		return err
	}

	return nil
}

func OperatorByYaml(r runner.Runner, operator string, yamlFile string, cluster *api.ClusterConfig) error {
	// TODO: current apply addons on node of master, future maybe apply in eggo
	if op, ok := ops[operator]; ok {
		return runKubectlWithYaml(r, op, yamlFile, cluster)
	}
	return fmt.Errorf("unsupport operator: %s", operator)
}

func GetKubeClient(configPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func WaitNodeRegister(name string, cluster string) error {
	path := filepath.Join(api.GetClusterHomePath(cluster), constants.KubeConfigFileNameAdmin)
	cs, err := GetKubeClient(path)
	if err != nil {
		logrus.Errorf("get kube client for cluster: %s failed: %v", cluster, err)
		return err
	}

	const timeout = 120
	finish := time.After(time.Second * timeout)
	for {
		select {
		case t := <-finish:
			return fmt.Errorf("timeout %s for wait node: %s", t.String(), name)
		default:
			n, err := cs.CoreV1().Nodes().Get(context.TODO(), name, v1.GetOptions{})
			if err != nil {
				logrus.Debugf("get node %s, failed: %s", name, err)
				break
			}
			if n != nil {
				logrus.Debugf("get node: %s success", name)
				return nil
			}
		}
		time.Sleep(time.Second)
	}
}
