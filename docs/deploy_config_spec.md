# 部署部分的规范说明

当前配置信息如下：

```bash
MASTER_IPS=("192.168.1.1" "192.168.1.2")
MASTER_NAMES=("master0" "master1")
NODE_IPS=("192.168.1.3" "192.168.1.4")
NODE_NAMES=("node1" "node2")
SERVICE_CLUSTER_IP_RANGE="10.32.0.0/16"
SERVICE_CLUSTER_IP_GATEWAY="10.32.0.1"
CLUSTER_IP_RANGE="10.244.0.0/16"
CLUSTER_GATEWAY="10.244.0.1"
API_SERVER_EXPOSE_IP="192.168.1.1"
API_SERVER_EXPOSE_PORT="6443"
CA_ROOT_PATH="/etc"
EXTRA_SANS=("10.10.1.2")
BOOTSTRAP_NODE_USER="root"
BOOTSTRAP_NODE_PASSWORD="123456"
ETCD_CLUSTER_TOKEN="kubernetes_etcd"
ETCD_CLUSTER_IPS=("192.168.1.1" "192.168.1.2")
ETCD_CLUSTER_NAMES=("master0" "master1")
NODE_SERVICE_CLUSTER_DNS="10.32.0.10"
NODE_KUBE_CLUSTER_CIDR="10.244.0.0/16"
MODULE_SAVE_PATH="/root/rpms"
```

对当前部署流程以及配置信息梳理，然后整理部署集群需要遵循的配置规范。

## 信息梳理

### 节点相关信息

节点信息主要包括：

- IP地址；
- 节点hostname；
- 拓展的SANS；
- 节点的用户名和密码；
- 依赖的软件包列表；
- 需要开启的端口（不被防火墙屏蔽）；

### ETCD相关信息

- 集群token；
- 部署ETCD节点的IP和hostname列表；
- 证书配置；
- 数据存储路径；
- `map[string][string]`类型的拓展参数；

### 集群相关信息

- 集群名称；
- K8S的版本；
- 服务的子网；
- DNS服务的VIP；
- API Server的IP/VIP地址；
- 集群Pod使用的子网；
- 集群Pod使用的IP的网关；
- DNS domain配置，默认为cluster.local；
- 证书存储路径；

### 容器引擎的相关信息

默认使用iSulad作为集群的容器引擎；

- pause镜像；
- cni-conf-dir;
- cni-bin-dir;
- 镜像下载的repository；
- 代理的配置；

### API Server组件配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给apiserver的参数；
- 证书支持的SANS列表；
- 超时时间；

### controller manager组件配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给controller manager的参数；

### scheduler组件配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给scheduler的参数；

### DNS组件配置

- 使用的DNS组件名称，支持coredns；
- 拓展参数，类型为`map[string][string]`，提供传递给组件解析；

以coredns为例，拓展参数至少包括如下信息：

- 提供服务的端口；
- api server的访问url；
- 访问api server需要的kubeconfig；
- DNS服务的VIP；
- 部署coredns的节点IP列表；

### kube-proxy组件的配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给kube-proxy的参数；

### kubelet组件配置

- DNS服务的VIP；
- DNS domain配置，默认为cluster.local；
- pause镜像；
- 网络插件，默认为cni；
- 设置的hostname；
- cni-bin-dir；
- container runtime以及对应的endpoint；
- 拓展参数，类型为`map[string][string]`，提供直接透传给kubelet的参数；

### 容器网络配置

默认使用`containernetworking-plugins`提供的本地桥网络，容器网络只能本地互通；支持配置其他网络，例如calico；

- 网络驱动，默认为local，支持calico等；
- 子网段；
- 网关地址；
- 网络配置yaml文件（支持以pod的方式部署网络）；

### 二进制安装相关配置

待完善

