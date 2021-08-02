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
 * Create: 2021-06-21
 * Description: function to setup binary coredns
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
	"isula.org/eggo/pkg/clusterdeployment/binary/commontools"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils"
	"isula.org/eggo/pkg/utils/endpoint"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"
)

const (
	CoreConfigTemp = `.:53 {
	errors
	health {
		lameduck 5s
	}
	ready
	kubernetes cluster.local in-addr.arpa ip6.arpa {
		pods insecure
		endpoint {{ .Endpoint }}
		kubeconfig {{ .AdminConf }} default-system
		fallthrough in-addr.arpa ip6.arpa
	}
	prometheus :9153
	forward . /etc/resolv.conf {
		max_concurrent 1000
	}
	cache 30
	loop
	reload
	loadbalance
}
`
	ServiceTemp = `[Unit]
Description=Kubernetes Core DNS server
Documentation=https://github.com/coredns/coredns
After=network.target

[Service]
ExecStart=/usr/bin/bash -c "KUBE_DNS_SERVICE_HOST={{ .DNSHostAddr }} coredns -conf {{ .CoreConfigPath }}"

Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`
	ServerConfTemp = `apiVersion: v1
kind: Service
metadata:
  name: kube-dns
  namespace: kube-system
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "CoreDNS"
spec:
    clusterIP: {{ .DNSHostAddr }}
    ports:
      - name: dns
        port: 53
        protocol: UDP
      - name: dns-tcp
        port: 53
        protocol: TCP
      - name: metrics
        port: 9153
        protocol: TCP
`
	EndpointTemp = `apiVersion: v1
kind: Endpoints
metadata:
  name: kube-dns
  namespace: kube-system
subsets:
  - addresses:
{{- range $i, $v := .HostIPs }}
      - ip: {{ $v }}
{{- end }}
    ports:
      - name: dns-tcp
        port: 53
        protocol: TCP
      - name: dns
        port: 53
        protocol: UDP
      - name: metrics
        port: 9153
        protocol: TCP
`
)

type BinaryCorednsServerSetupTask struct {
	Cluster *api.ClusterConfig
	NodeIPs []string
}

func (cs *BinaryCorednsServerSetupTask) Name() string {
	return "CorednsSetupTask"
}

func (cs *BinaryCorednsServerSetupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	if err := createCoreServerTemplate(cs.Cluster, r); err != nil {
		return nil
	}

	return createCoreEndpointTemplate(cs.Cluster, r, cs.NodeIPs)
}

func createCoreServerTemplate(cluster *api.ClusterConfig, r runner.Runner) error {
	var sb strings.Builder
	datastore := map[string]interface{}{}
	datastore["DNSHostAddr"] = cluster.ServiceCluster.DNSAddr
	serverConfig, err := template.TemplateRender(ServerConfTemp, datastore)
	if err != nil {
		logrus.Errorf("rend core dns server failed: %v", err)
		return err
	}
	manifestDir := cluster.GetManifestDir()
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s && ", manifestDir))
	serverBase64 := base64.StdEncoding.EncodeToString([]byte(serverConfig))
	sb.WriteString(fmt.Sprintf("echo %s | base64 -d > %s/coredns_server.yaml", serverBase64, manifestDir))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl apply -f %s/coredns_server.yaml", fmt.Sprintf("%s/%s", cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin), manifestDir))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core dns server failed: %v", err)
		return err
	}
	return nil
}

func createCoreEndpointTemplate(cluster *api.ClusterConfig, r runner.Runner, ips []string) error {
	if len(ips) == 0 {
		return fmt.Errorf("core dns endpoint need one ip")
	}
	var sb strings.Builder
	datastore := map[string]interface{}{}
	datastore["HostIPs"] = ips
	epConfig, err := template.TemplateRender(EndpointTemp, datastore)
	if err != nil {
		logrus.Errorf("rend core dns endpoint failed: %v", err)
		return err
	}
	manifestDir := cluster.GetManifestDir()
	sb.WriteString("sudo -E /bin/sh -c \"")
	sb.WriteString(fmt.Sprintf("mkdir -p %s && ", manifestDir))
	epBase64 := base64.StdEncoding.EncodeToString([]byte(epConfig))
	sb.WriteString(fmt.Sprintf("echo %s | base64 -d > %s/coredns_ep.yaml", epBase64, manifestDir))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl apply -f %s/coredns_ep.yaml", fmt.Sprintf("%s/%s", cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin), manifestDir))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core dns endpoint failed: %v", err)
		return err
	}
	return nil
}

type BinaryCorednsSetupTask struct {
	Cluster *api.ClusterConfig
}

func (ct *BinaryCorednsSetupTask) Name() string {
	return "CorednsSetupTask"
}

func (ct *BinaryCorednsSetupTask) createCoreConfigTemplate(r runner.Runner) error {
	var sb strings.Builder
	datastore := map[string]interface{}{}
	useEndPoint, err := endpoint.GetAPIServerEndpoint(ct.Cluster)
	if err != nil {
		logrus.Errorf("get api server endpoint failed: %v", err)
		return err
	}
	datastore["Endpoint"] = useEndPoint
	datastore["AdminConf"] = fmt.Sprintf("%s/%s", ct.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)
	coreConfig, err := template.TemplateRender(CoreConfigTemp, datastore)
	if err != nil {
		logrus.Errorf("rend core config failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"mkdir -p /etc/dns")
	coreBase64 := base64.StdEncoding.EncodeToString([]byte(coreConfig))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > /etc/dns/Corefile", coreBase64))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core config failed: %v", err)
		return err
	}
	return nil
}

func (ct *BinaryCorednsSetupTask) createServiceTemplate(r runner.Runner) error {
	var sb strings.Builder
	datastore := map[string]interface{}{}
	datastore["DNSHostAddr"] = ct.Cluster.ServiceCluster.DNSAddr
	datastore["CoreConfigPath"] = "/etc/dns/Corefile"
	serviceConfig, err := template.TemplateRender(ServiceTemp, datastore)
	if err != nil {
		logrus.Errorf("rend core dns service failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"")
	serviceBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConfig))
	shell, err := commontools.GetSystemdServiceShell("coredns", serviceBase64, true)
	if err != nil {
		logrus.Errorf("get coredns systemd service file failed: %v", err)
		return err
	}

	_, err = r.RunShell(shell, "setcoredns")
	if err != nil {
		logrus.Errorf("create core dns service failed: %v", err)
		return err
	}
	return nil
}

func (ct *BinaryCorednsSetupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	if err := ct.createCoreConfigTemplate(r); err != nil {
		return err
	}
	if err := ct.createServiceTemplate(r); err != nil {
		return err
	}
	return nil
}

type BinaryCoredns struct {
}

func (bc *BinaryCoredns) Setup(cluster *api.ClusterConfig) error {
	masterIPs := utils.GetMasterIPList(cluster)
	if len(masterIPs) == 0 {
		return fmt.Errorf("no master host found, can not setup coredns service")
	}
	// ensure coredns service is running
	st := task.NewTaskInstance(
		&BinaryCorednsSetupTask{
			Cluster: cluster,
		},
	)

	err := nodemanager.RunTaskOnNodes(st, masterIPs)
	if err != nil {
		return err
	}

	sst := task.NewTaskInstance(
		&BinaryCorednsServerSetupTask{
			Cluster: cluster,
			NodeIPs: masterIPs,
		},
	)

	err = nodemanager.RunTaskOnNodes(sst, masterIPs[0:1])
	if err != nil {
		return err
	}

	if err = nodemanager.WaitNodesFinish(masterIPs, time.Minute*5); err != nil {
		logrus.Errorf("coredns setup failed: %v", err)
		return err
	}

	return nil
}

type BinaryCorednsCleanupTask struct {
	Cluster   *api.ClusterConfig
	cleanYaml bool
}

func (ct *BinaryCorednsCleanupTask) Name() string {
	return "BinaryCorednsCleanupTask"
}
func (ct *BinaryCorednsCleanupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	cleanTmpl := `
#!/bin/bash
export KUBECONFIG={{ .KubeConfig }}
{{- if .Endpoints }}
if [ -f {{ .Endpoints }} ]; then
	kubectl delete -f {{ .Endpoints }}
	if [ $? -ne 0 ]; then
		echo "delete {{ .Endpoints }} failed"
	fi
fi
{{- end }}

{{- if .Service }}
if [ -f {{ .Service }} ]; then
	kubectl delete -f {{ .Service }}
	if [ $? -ne 0 ]; then
		echo "delete {{ .Service }} failed"
	fi
fi
{{- end }}

systemctl stop coredns
systemctl disable coredns
rm -f /etc/dns/Corefile /usr/lib/systemd/system/coredns.service
exit 0
`
	datastore := make(map[string]interface{})
	if ct.cleanYaml {
		// TODO: should support decrease item of endpoints
		datastore["Endpoints"] = filepath.Join(ct.Cluster.GetManifestDir(), "coredns_ep.yaml")
		datastore["Service"] = filepath.Join(ct.Cluster.GetManifestDir(), "coredns_server.yaml")
	}
	datastore["KubeConfig"] = filepath.Join(ct.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	cmdStr, err := template.TemplateRender(cleanTmpl, datastore)
	if err != nil {
		return err
	}

	_, err = r.RunShell(cmdStr, "coredns_cleanup")
	if err != nil {
		return err
	}

	return nil
}

func (bc *BinaryCoredns) Cleanup(cluster *api.ClusterConfig) error {
	masterIPs := utils.GetMasterIPList(cluster)
	if len(masterIPs) == 0 {
		logrus.Warn("no master host found, can not cleanup coredns service")
		return nil
	}
	sst := task.NewTaskIgnoreErrInstance(
		&BinaryCorednsCleanupTask{
			Cluster:   cluster,
			cleanYaml: true,
		},
	)

	err := nodemanager.RunTaskOnNodes(sst, masterIPs)
	if err != nil {
		logrus.Warnf("run cleanup coredns task failed: %v", err)
		return nil
	}

	if err = nodemanager.WaitNodesFinish(masterIPs, time.Minute*5); err != nil {
		logrus.Warnf("wait to coredns cleanup failed: %v", err)
		return nil
	}

	return nil
}

type BinaryCorednsServerJoinTask struct {
	Cluster *api.ClusterConfig
	NodeIPs []string
}

func (cs *BinaryCorednsServerJoinTask) Name() string {
	return "BinaryCorednsServerJoinTask"
}

func (cs *BinaryCorednsServerJoinTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	return createCoreEndpointTemplate(cs.Cluster, r, cs.NodeIPs)
}

func (bc *BinaryCoredns) JoinNode(nodeAddr string, cluster *api.ClusterConfig) error {
	// TODO: should get coredns ip list from status of cluster
	masterIPs := utils.GetMasterIPList(cluster)
	if len(masterIPs) == 0 {
		return fmt.Errorf("no master host found, can not setup coredns service")
	}
	// ensure coredns service is running
	st := task.NewTaskInstance(
		&BinaryCorednsSetupTask{
			Cluster: cluster,
		},
	)

	err := nodemanager.RunTaskOnNodes(st, []string{nodeAddr})
	if err != nil {
		return err
	}

	if err = nodemanager.WaitNodesFinish([]string{nodeAddr}, time.Minute*5); err != nil {
		logrus.Errorf("wait to coredns service running failed: %v", err)
		return err
	}

	sst := task.NewTaskInstance(
		&BinaryCorednsServerJoinTask{
			Cluster: cluster,
			NodeIPs: append(masterIPs, nodeAddr),
		},
	)

	useMaster, err := nodemanager.RunTaskOnOneNode(sst, masterIPs)
	if err != nil {
		return err
	}

	if err = nodemanager.WaitNodesFinish([]string{useMaster}, time.Minute*5); err != nil {
		logrus.Errorf("wait to join new coredns node failed: %v", err)
		return err
	}

	return nil
}

func (bc *BinaryCoredns) CleanNode(nodeAddr string, cluster *api.ClusterConfig) error {
	sst := task.NewTaskInstance(
		&BinaryCorednsCleanupTask{
			Cluster:   cluster,
			cleanYaml: false,
		},
	)

	err := nodemanager.RunTaskOnNodes(sst, []string{nodeAddr})
	if err != nil {
		logrus.Warnf("run cleanup coredns task failed: %v", err)
		return nil
	}

	if err = nodemanager.WaitNodesFinish([]string{nodeAddr}, time.Minute*5); err != nil {
		logrus.Warnf("wait to coredns cleanup failed: %v", err)
		return nil
	}

	return nil
}
