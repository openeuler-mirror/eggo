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
 * Author: haozi007
 * Create: 2021-06-22
 * Description: function to setup pod coredns
 ******************************************************************************/
package coredns

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/utils/kubectl"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"
)

const (
	defaultCorednsImageVersion = "1.8.4"
	defaultCorednsReplicas     = 2
)

type PodCorednsSetupTask struct {
	Cluster *api.ClusterConfig
}

func (ct *PodCorednsSetupTask) Name() string {
	return "PodCorednsSetupTask"
}

func (ct *PodCorednsSetupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	datastore := make(map[string]interface{})
	datastore["Replicas"] = defaultCorednsReplicas
	datastore["ImageVersion"] = defaultCorednsImageVersion
	if ct.Cluster.ServiceCluster.DNS.ImageVersion != "" {
		datastore["ImageVersion"] = ct.Cluster.ServiceCluster.DNS.ImageVersion
	}
	if ct.Cluster.ServiceCluster.DNS.Replicas > 0 {
		datastore["Replicas"] = ct.Cluster.ServiceCluster.DNS.Replicas
	}
	datastore["ClusterIP"] = ct.Cluster.ServiceCluster.DNSAddr
	corednsYaml, err := template.TemplateRender(podCorednsTmpl, datastore)
	if err != nil {
		return err
	}
	manifestDir := ct.Cluster.GetManifestDir()
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s", manifestDir))
	roleBase64 := base64.StdEncoding.EncodeToString([]byte(corednsYaml))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s", roleBase64, filepath.Join(manifestDir, "coredns.yaml")))
	sb.WriteString("\"")
	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("[corends] create yaml for pod corends failed: %v", err)
		return err
	}

	err = kubectl.OperatorByYaml(r, kubectl.ApplyOpKey, filepath.Join(manifestDir, "coredns.yaml"), ct.Cluster)
	if err != nil {
		logrus.Errorf("[corends] apply pod corends failed: %v", err)
		return err
	}

	return nil
}

type PodCorednsCleanupTask struct {
	Cluster *api.ClusterConfig
}

func (ct *PodCorednsCleanupTask) Name() string {
	return "PodCorednsCleanupTask"
}

func (ct *PodCorednsCleanupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	datastore := make(map[string]interface{})
	datastore["Replicas"] = defaultCorednsReplicas
	datastore["ImageVersion"] = defaultCorednsImageVersion
	if ct.Cluster.ServiceCluster.DNS.ImageVersion != "" {
		datastore["ImageVersion"] = ct.Cluster.ServiceCluster.DNS.ImageVersion
	}
	if ct.Cluster.ServiceCluster.DNS.Replicas > 0 {
		datastore["Replicas"] = ct.Cluster.ServiceCluster.DNS.Replicas
	}
	datastore["ClusterIP"] = ct.Cluster.ServiceCluster.DNSAddr
	corednsYaml, err := template.TemplateRender(podCorednsTmpl, datastore)
	if err != nil {
		return err
	}
	manifestDir := ct.Cluster.GetManifestDir()
	var sb strings.Builder
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s", manifestDir))
	roleBase64 := base64.StdEncoding.EncodeToString([]byte(corednsYaml))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > %s/%s", roleBase64, manifestDir, "coredns.yaml"))
	sb.WriteString("\"")
	err = kubectl.OperatorByYaml(r, kubectl.DeleteOpKey, filepath.Join(manifestDir, "coredns.yaml"), ct.Cluster)
	if err != nil {
		return err
	}

	return nil
}

type PodCoredns struct {
}

func (pc *PodCoredns) Setup(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}
	t := task.NewTaskInstance(&PodCorednsSetupTask{Cluster: cluster})
	var masters []string
	for _, n := range cluster.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
		}
	}

	useMaster, err := nodemanager.RunTaskOnOneNode(t, masters)
	if err != nil {
		return err
	}
	err = nodemanager.WaitNodesFinish([]string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}

	logrus.Infof("[cluster] setup corends use pod success")
	return nil
}

func (pc *PodCoredns) Cleanup(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}
	t := task.NewTaskInstance(&PodCorednsCleanupTask{Cluster: cluster})
	var masters []string
	for _, n := range cluster.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
		}
	}

	useMaster, err := nodemanager.RunTaskOnOneNode(t, masters)
	if err != nil {
		return err
	}
	err = nodemanager.WaitNodesFinish([]string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}

	logrus.Infof("[cluster] cleanup corends use pod success")
	return nil
}

func (bc *PodCoredns) JoinNode(node string, cluster *api.ClusterConfig) error {
	// nothing need to do
	return nil
}

func (bc *PodCoredns) CleanNode(node string, cluster *api.ClusterConfig) error {
	// nothing need to do
	return nil
}
