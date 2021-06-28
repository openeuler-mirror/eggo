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
	lbConfig *api.LoadBalancer
	masters  []string
}

func (it *LoadBalanceTask) Name() string {
	return "LoadBalanceTask"
}

func (it *LoadBalanceTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("prepare loadbalancer...\n")

	// check loadbalancer dependences
	path, err := check(r, it.lbConfig, it.masters)
	if err != nil {
		logrus.Errorf("check failed: %v", err)
		return err
	}

	if _, err := r.RunCommand("sudo -E /bin/sh -c \"mkdir -p /etc/kubernetes\""); err != nil {
		return fmt.Errorf("mkdir failed")
	}

	// prepare nginx config
	if err := prepareConfig(r, it.lbConfig, it.masters); err != nil {
		logrus.Errorf("prepare config failed: %v", err)
		return err
	}

	// prepare and start nginx service
	if err := commontools.SetupLoadBalanceServices(r, path); err != nil {
		logrus.Errorf("run service failed: %v", err)
		return err
	}

	logrus.Info("prepare loadbalancer success\n")
	return nil
}

func check(r runner.Runner, lbConfig *api.LoadBalancer, masters []string) (string, error) {
	if lbConfig.IP == "" || lbConfig.Port == "" {
		return "", fmt.Errorf("invalid loadbalance %s:%s", lbConfig.IP, lbConfig.Port)
	}
	if len(masters) == 0 {
		return "", fmt.Errorf("empty apiserver address")
	}

	path, err := r.RunCommand(fmt.Sprintf("sudo -E /bin/sh -c \"which %s\"", LoadBalanceSoftware))
	if err != nil {
		logrus.Errorf("check software: %s, failed: %v\n", LoadBalanceSoftware, err)
		return "", err
	}
	logrus.Debugf("check software: %s success\n", LoadBalanceSoftware)

	return path, nil
}

func prepareConfig(r runner.Runner, lbConfig *api.LoadBalancer, masters []string) error {
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

	datastore := map[string]interface{}{}
	datastore["modulesPath"] = modulesPath
	datastore["IPs"] = masters
	datastore["port"] = lbConfig.Port
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

func SetupLoadBalancer(config *api.ClusterConfig, lb *api.HostConfig) error {
	masterIPs := utils.GetMasterIPList(config)
	if len(masterIPs) == 0 {
		return fmt.Errorf("no master host found, can not setup loadbalance")
	}

	taskSetupLoadBalancer := task.NewTaskInstance(
		&LoadBalanceTask{
			lbConfig: &config.LoadBalancer,
			masters:  masterIPs,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskSetupLoadBalancer, []string{lb.Address}); err != nil {
		return err
	}

	if err := nodemanager.WaitNodesFinish([]string{lb.Address}, time.Minute*2); err != nil {
		logrus.Errorf("wait to deploy loadbalancer finish failed: %v", err)
		return err
	}

	return nil
}

type UpdateLoadBalanceTask struct {
	lbConfig *api.LoadBalancer
	masters  []string
}

func (it *UpdateLoadBalanceTask) Name() string {
	return "LoadBalanceTask"
}

func (it *UpdateLoadBalanceTask) Run(r runner.Runner, hcg *api.HostConfig) error {
	logrus.Info("update loadbalancer...\n")

	// remove nginx config
	if _, err := r.RunCommand("sudo -E /bin/sh -c \"rm -rf /etc/kubernetes/kube-nginx.conf\""); err != nil {
		logrus.Errorf("remove config failed: %v", err)
		return err
	}

	// prepare nginx config
	if err := prepareConfig(r, it.lbConfig, it.masters); err != nil {
		logrus.Errorf("prepare config failed: %v", err)
		return err
	}

	// restart nginx service
	if _, err := r.RunCommand("sudo -E /bin/sh -c \"systemctl restart nginx\""); err != nil {
		logrus.Errorf("restart service failed: %v", err)
		return err
	}

	logrus.Info("update loadbalancer success\n")
	return nil
}

func UpdateLoadBalancer(config *api.ClusterConfig, lb *api.HostConfig) error {
	// update loadbalance when join/drop master

	masterIPs := utils.GetMasterIPList(config)
	if len(masterIPs) == 0 {
		return fmt.Errorf("no master host found, can not update loadbalance")
	}

	taskUpdateLoadBalancer := task.NewTaskInstance(
		&UpdateLoadBalanceTask{
			lbConfig: &config.LoadBalancer,
			masters:  masterIPs,
		},
	)

	if err := nodemanager.RunTaskOnNodes(taskUpdateLoadBalancer, []string{lb.Address}); err != nil {
		return err
	}

	if err := nodemanager.WaitNodesFinish([]string{lb.Address}, time.Minute*2); err != nil {
		logrus.Errorf("wait to update loadbalancer finish failed: %v", err)
		return err
	}

	return nil
}
