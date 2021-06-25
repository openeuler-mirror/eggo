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
 * Create: 2021-05-22
 * Description: eggo loadbalance binary implement
 ******************************************************************************/

package loadbalance

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/clusterdeployment/binary/commontools"
	"isula.org/eggo/pkg/clusterdeployment/binary/infrastructure"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"
)

const (
	// TODO: support loadbalance by other software
	LoadBalanceSoftware = "nginx"
)

type LoadBalanceTask struct {
	ccfg *api.ClusterConfig
}

func (it *LoadBalanceTask) Name() string {
	return "LoadBalanceTask"
}

func (it *LoadBalanceTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("prepare loadbalancer...\n")

	// check loadbalancer dependences
	path, err := check(r, it.ccfg)
	if err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if _, err := r.RunCommand("sudo -E /bin/sh -c \"mkdir -p /etc/kubernetes\""); err != nil {
		return fmt.Errorf("mkdir failed")
	}

	// prepare nginx config
	if err := prepareConfig(r, it.ccfg); err != nil {
		logrus.Errorf("prepare config failed: %v", err)
		return err
	}

	// prepare and start nginx service
	if err := commontools.SetupLoadBalanceServices(r, it.ccfg, path); err != nil {
		logrus.Errorf("run service failed: %v", err)
		return err
	}

	// expose port
	p := it.ccfg.LoadBalancer.Port + "/tcp"
	if err := infrastructure.ExposePorts(r, p); err != nil {
		logrus.Errorf("expose port failed: %v", err)
		return err
	}

	logrus.Info("prepare loadbalancer success\n")
	return nil
}

func check(r runner.Runner, ccfg *api.ClusterConfig) (string, error) {
	if ccfg.LoadBalancer.IP == "" || ccfg.LoadBalancer.Port == "" {
		return "", fmt.Errorf("invalid loadbalance %s:%s", ccfg.LoadBalancer.IP, ccfg.LoadBalancer.Port)
	}

	path, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"which %s\"", LoadBalanceSoftware))
	if err != nil {
		logrus.Errorf("check software: %s, failed: %v\n", LoadBalanceSoftware, err)
		return "", err
	}
	logrus.Debugf("check software: %s success\n", LoadBalanceSoftware)

	return path, nil
}

func prepareConfig(r runner.Runner, ccfg *api.ClusterConfig) error {
	nginxConfig := `load_module {{ .modulesPath }}/ngx_stream_module.so;

worker_processes 1;

events {
    worker_connections  1024;
}

stream {
    upstream backend {
        hash $remote_addr consistent;
        {{- range $i, $v := .IPs }}
        server {{ $v }}:6443 max_fails=3 fail_timeout=30s;
        {{- end }}
    }

    server {
        listen 0.0.0.0:{{ .port }};
        proxy_connect_timeout 1s;
        proxy_pass backend;
    }
}
`

	modulesPath, err := getModulePath(r)
	if err != nil {
		logrus.Errorf("get nginx modules path failed: %v", err)
	}
	masterIPs := utils.GetMasterIPList(ccfg)
	if len(masterIPs) == 0 {
		return fmt.Errorf("no master host found, can not setup loadbalance")
	}

	datastore := map[string]interface{}{}
	datastore["modulesPath"] = modulesPath
	datastore["IPs"] = masterIPs
	datastore["port"] = ccfg.LoadBalancer.Port
	config, err := template.TemplateRender(nginxConfig, datastore)
	if err != nil {
		return err
	}

	var sb strings.Builder
	configBase64 := base64.StdEncoding.EncodeToString([]byte(config))
	sb.WriteString(fmt.Sprintf("sudo -E /bin/sh -c \"echo %s | base64 -d > %s\"", configBase64, "/etc/kubernetes/kube-nginx.conf"))
	_, err = r.RunCommand(sb.String())
	if err != nil {
		return err
	}

	logrus.Debugf("prepare nginx config success")

	return nil
}

func getModulePath(r runner.Runner) (string, error) {
	path, err := r.RunCommand("sudo -E /bin/sh -c \"nginx -V 2>&1 | tr ' ' '\\n' | grep modules-path | cut -d '=' -f2\"")
	if err != nil {
		return "", err
	}

	if path == "" {
		path = "/usr/lib64/nginx/modules"
	}

	return path, nil
}

func SetupLoadBalancer(config *api.ClusterConfig, loadBalancer string) error {
	if config == nil {
		return fmt.Errorf("empty cluster config")
	}

	if loadBalancer == "" {
		logrus.Info("no loadbalance")
		return nil
	}

	taskSetupLoadBalancer := task.NewTaskInstance(
		&LoadBalanceTask{
			ccfg: config,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskSetupLoadBalancer, []string{loadBalancer}); err != nil {
		return err
	}

	if err := nodemanager.WaitNodesFinish([]string{loadBalancer}, time.Minute*2); err != nil {
		logrus.Errorf("wait to deploy loadbalancer finish failed: %v", err)
		return err
	}

	return nil
}

func Init(config *api.ClusterConfig) error {
	loadbalancer := ""
	for _, node := range config.Nodes {
		if node.Type&api.LoadBalance != 0 {
			loadbalancer = node.Address
			break
		}
	}

	return SetupLoadBalancer(config, loadbalancer)
}

func UpdateLoadBalancer() {
	// TODO: update loadbalance when join/drop master
}
