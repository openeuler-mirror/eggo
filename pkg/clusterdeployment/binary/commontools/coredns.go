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
 * Create: 2021-06-07
 * Description: setup coredns service and server
 ******************************************************************************/
package commontools

import (
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/constants"
	"gitee.com/openeuler/eggo/pkg/utils/endpoint"
	"gitee.com/openeuler/eggo/pkg/utils/nodemanager"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"gitee.com/openeuler/eggo/pkg/utils/task"
	kkutil "github.com/kubesphere/kubekey/pkg/util"
	"github.com/lithammer/dedent"
	"github.com/sirupsen/logrus"
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
ExecStart=bash -c "KUBE_DNS_SERVICE_HOST={{ .DNSHostAddr }} coredns -conf {{ .CoreConfigPath }}"

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

type CorednsServerSetupTask struct {
	Cluster *api.ClusterConfig
	NodeIPs []string
}

func (cs *CorednsServerSetupTask) Name() string {
	return "CorednsSetupTask"
}

func (cs *CorednsServerSetupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	if err := cs.createCoreServerTemplate(r); err != nil {
		return nil
	}

	return cs.createCoreEndpointTemplate(r, cs.NodeIPs)
}

func (cs *CorednsServerSetupTask) createCoreServerTemplate(r runner.Runner) error {
	var sb strings.Builder
	tmpl := template.Must(template.New("CoreServer").Parse(dedent.Dedent(ServerConfTemp)))
	datastore := map[string]interface{}{}
	datastore["DNSHostAddr"] = cs.Cluster.ServiceCluster.DNSAddr
	serverConfig, err := kkutil.Render(tmpl, datastore)
	if err != nil {
		logrus.Errorf("rend core dns server failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"")
	serverBase64 := base64.StdEncoding.EncodeToString([]byte(serverConfig))
	sb.WriteString(fmt.Sprintf("echo %s | base64 -d > %s/coredns_server.yaml", serverBase64, cs.Cluster.GetManifestDir()))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl apply -f %s/coredns_server.yaml", fmt.Sprintf("%s/%s", cs.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin), cs.Cluster.GetManifestDir()))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core dns server failed: %v", err)
		return err
	}
	return nil
}

func (cs *CorednsServerSetupTask) createCoreEndpointTemplate(r runner.Runner, ips []string) error {
	if len(ips) == 0 {
		return fmt.Errorf("core dns endpoint need one ip")
	}
	var sb strings.Builder
	tmpl := template.Must(template.New("CoreServer").Parse(dedent.Dedent(EndpointTemp)))
	datastore := map[string]interface{}{}
	datastore["HostIPs"] = ips
	epConfig, err := kkutil.Render(tmpl, datastore)
	if err != nil {
		logrus.Errorf("rend core dns endpoint failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"")
	epBase64 := base64.StdEncoding.EncodeToString([]byte(epConfig))
	sb.WriteString(fmt.Sprintf("echo %s | base64 -d > %s/coredns_ep.yaml", epBase64, cs.Cluster.GetManifestDir()))
	sb.WriteString(fmt.Sprintf(" && KUBECONFIG=%s kubectl apply -f %s/coredns_ep.yaml", fmt.Sprintf("%s/%s", cs.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin), cs.Cluster.GetManifestDir()))
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core dns endpoint failed: %v", err)
		return err
	}
	return nil
}

type CorednsSetupTask struct {
	Cluster *api.ClusterConfig
}

func (ct *CorednsSetupTask) Name() string {
	return "CorednsSetupTask"
}

func (ct *CorednsSetupTask) createCoreConfigTemplate(r runner.Runner) error {
	var sb strings.Builder
	tmpl := template.Must(template.New("Coreconfig").Parse(dedent.Dedent(CoreConfigTemp)))
	datastore := map[string]interface{}{}
	useEndPoint, err := endpoint.GetAPIServerEndpoint(ct.Cluster.ControlPlane.Endpoint, ct.Cluster.LocalEndpoint)
	if err != nil {
		logrus.Errorf("get api server endpoint failed: %v", err)
		return err
	}
	datastore["Endpoint"] = useEndPoint
	datastore["AdminConf"] = fmt.Sprintf("%s/%s", ct.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)
	coreConfig, err := kkutil.Render(tmpl, datastore)
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

func (ct *CorednsSetupTask) createServiceTemplate(r runner.Runner) error {
	var sb strings.Builder
	tmpl := template.Must(template.New("CoreService").Parse(dedent.Dedent(ServiceTemp)))
	datastore := map[string]interface{}{}
	datastore["DNSHostAddr"] = ct.Cluster.ServiceCluster.DNSAddr
	datastore["CoreConfigPath"] = "/etc/dns/Corefile"
	serviceConfig, err := kkutil.Render(tmpl, datastore)
	if err != nil {
		logrus.Errorf("rend core dns service failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"")
	serviceBase64 := base64.StdEncoding.EncodeToString([]byte(serviceConfig))
	shell, err := GetSystemdServiceShell("coredns", serviceBase64, true)
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

func (ct *CorednsSetupTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	if err := ct.createCoreConfigTemplate(r); err != nil {
		return err
	}
	if err := ct.createServiceTemplate(r); err != nil {
		return err
	}
	return nil
}

func getMasterIPList(c *api.ClusterConfig) []string {
	var masters []string
	for _, n := range c.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
			continue
		}
	}

	return masters
}

func SetUpCoredns(cluster *api.ClusterConfig) error {
	masterIPs := getMasterIPList(cluster)
	if len(masterIPs) == 0 {
		return fmt.Errorf("no master host found, can not setup coredns service")
	}
	// ensure coredns service is running
	st := task.NewTaskInstance(
		&CorednsSetupTask{
			Cluster: cluster,
		},
	)

	err := nodemanager.RunTaskOnNodes(st, masterIPs)
	if err != nil {
		return err
	}

	if err = nodemanager.WaitTaskOnNodesFinished(st, masterIPs, time.Minute*5); err != nil {
		logrus.Errorf("wait to coredns service running failed: %v", err)
		return err
	}

	sst := task.NewTaskInstance(
		&CorednsServerSetupTask{
			Cluster: cluster,
			NodeIPs: masterIPs,
		},
	)

	err = nodemanager.RunTaskOnNodes(sst, masterIPs[0:1])
	if err != nil {
		return err
	}

	if err = nodemanager.WaitTaskOnNodesFinished(sst, masterIPs[0:1], time.Minute*5); err != nil {
		logrus.Errorf("wait to coredns k8s server apply failed: %v", err)
		return err
	}

	return nil
}
