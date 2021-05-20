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

const (
	Master = 0x1
	Worker = 0x2
	ETCD   = 0x4
)

type OpenPorts struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp/udp
}

type Packages struct {
	Type string `json:"type"` // repo, pkg, binary
	Dst  string `json:"dstpath"`
}

type HostConfig struct {
	Arch           string              `json:"arch"`
	Name           string              `json:"name"`
	Address        string              `json:"address"`
	Port           int                 `json:"port"`
	ExtraIPs       []string            `json:"extra-ips"`
	OpenPorts      []*OpenPorts        `json:"open-ports"`
	UserName       string              `json:"username"`
	Password       string              `json:"password"`
	PrivateKey     string              `json:"private-key"`
	PrivateKeyPath string              `json:"private-key-path"`
	Packages       map[string]Packages `json:"packages"`

	// 0x1 is master, 0x2 is worker, 0x4 is etcd
	// 0x3 is master and worker
	// 0x7 is master, worker and etcd
	Type uint16 `json:"type"`

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

type APIEndpoint struct {
	AdvertiseAddress string `json:"advertise-address,omitempty"`
	BindPort         int32  `json:"bind-port,omitempty"`
}
type ControlPlaneConfig struct {
	Endpoint      string          `json:"endpoint,omitempty"`
	ApiConf       *ApiServer      `json:"apiconf,omitempty"`
	ManagerConf   *ControlManager `json:"managerconf,omitempty"`
	SchedulerConf *Scheduler      `json:"schedulerconf,omitempty"`
}

type CertificateConfig struct {
	// default is "/etc/kubernetes"
	// certificate save in "${SavePath}/pki"
	// kube config save in "${SavePath}"
	SavePath string `json:"savepath"`
}

type ServiceClusterConfig struct {
	CIDR    string `json:"cidr"`
	Gateway string `json:"gateway"`
}

type PackageSrcConfig struct {
	Type   string `json:"type"` // tar.gz...
	ArmSrc string `json:"arm-srcpath"`
	X86Src string `json:"x86-srcPath"`
}

type ClusterConfig struct {
	Certificate    CertificateConfig    `json:"certificate,omitempty"`
	ServiceCluster ServiceClusterConfig `json:"servicecluster,omitempty"`
	LocalEndpoint  APIEndpoint          `json:"local-endpoint,omitempty"`
	ControlPlane   ControlPlaneConfig   `json:"controlplane,omitempty"`
	PackageSrc     *PackageSrcConfig    `json:"packagesource,omitempty"`
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
