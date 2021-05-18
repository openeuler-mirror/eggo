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
 * Create: 2021-05-11
 * Description: cluster deploy types
 ******************************************************************************/

package clusterdeployment

type HostConfig struct {
	Arch           string   `json:"arch"`
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Port           int      `json:"port"`
	ExtraIPs       []string `json:"extra-ips"`
	OpenPorts      []int    `json:"open-ports"`
	UserName       string   `json:"username"`
	Password       string   `json:"password"`
	PrivateKey     string   `json:"private-key"`
	PrivateKeyPath string   `json:"private-key-path"`

	Type string `json:"type"`

	Labels map[string]string `json:"labels"`
}

type Sans struct {
	DNSNames []string `json:"dns-names"`
	IPs      []string `json:"ips"`
}
type ApiServer struct {
	Sans      Sans              `json:"sans,omitempty"`
	Timeout   string            `json:"timeout,omitempty"`
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type ControlManager struct {
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type Scheduler struct {
	ExtraArgs map[string]string `json:"extra-args,omitempty"`
}

type ControlPlaneConfig struct {
	ApiConf       ApiServer      `json:"apiconf,omitempty"`
	ManagerConf   ControlManager `json:"managerconf,omitempty"`
	SchedulerConf Scheduler      `json:"schedulerconf,omitempty"`
}

type CertificateConfig struct {
	SavePath string `json:"savepath"`
}

type ServiceClusterConfig struct {
	CIDR    string `json:"cidr"`
	Gateway string `json:"gateway"`
}

type ClusterConfig struct {
	Certificate    CertificateConfig    `json:"certificate,omitempty"`
	ServiceCluster ServiceClusterConfig `json:"servicecluster,omitempty"`
	ControlPlane   *ControlPlaneConfig  `json:"controlplane,omitempty"`
	Nodes          []*HostConfig        `json:"nodes,omitempty"`
	// TODO: add other configurations at here
}

type ClusterDeploymentAPI interface {
	PrepareInfrastructure() error
	DeployEtcdCluster() error
	InitControlPlane() error
	JoinBootstrap() error
	UpgradeCluster() error
	CleanupCluster() error
}
