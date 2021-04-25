# 部署部分的规范说明

## 信息梳理

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

```yaml
cluster:
  name: my-k8s
  version: 1.20.2
  cluster-services-cidr: 10.32.0.0/16
  dns-service-vip: 10.32.0.10
  dns-domain: cluster.local
  api-server-entrypoint: 192.168.1.1:6443
  cluster-pod-cidr: 10.244.0.0/16
  cluster-pod-gateway: 10.244.0.1
  cert-store-path: /etc/kubernetes/pki
```

#### 节点相关信息

节点信息主要包括：

- OS信息
- 架构信息
- IP地址；
- 节点hostname；
- 拓展的SANS；
- 节点的用户名和密码；
- 依赖的软件包列表；
- 需要开启的端口（不被防火墙屏蔽）；

```yaml
nodes:
  -
    os:
      name: openeuler
      version: 21.03
    arch: arm64
    ip: 192.168.1.2
    hostname: master0
    sans:
      - 10.8.8.8
      - 9.8.8.8
    user: root
    password: 123456
    dependences:
      -
        name: etcd
        source: /root/rpms/etcd-3.4.14-2.aarch64.rpm
        command: rpm
        dst: ""
      -
        name: authorized_keys
        source: /root/rpms/authorized_keys
        command: cp
        dst: ~/.ssh/authorized_keys
    open-ports:
      - 6443
      - 12500
      - 12501
```

#### 二进制安装相关配置

二进制安装主要是为环境准备依赖组件，配置需要指明文件类型、安装方式和目标路径；

- 文件名称，例如：kubeadm二进制等；
- 文件类型，例如：file/dir/rpm等；
- 安装方式，例如：cp/rpm 等；
- 文件原始路径，例如：/root/binary等
- 安装目标路径，例如：/usr/local/bin等；

```yaml
binary:
  -
    name: kubeadm
    type: file
    install-tool: cp
    src: /root/binary
    dist: /usr/local/bin
  -
    name: etcd.rpm
    type: rpm
    install-tool: rpm
    src: /root/binary
    dist: none
```

#### 容器网络配置

默认使用`containernetworking-plugins`提供的本地桥网络，容器网络只能本地互通；支持配置其他网络，例如calico；

- 网络驱动，默认为local，支持calico等；
- 子网段；
- 网关地址；
- 网络配置yaml文件（支持以pod的方式部署网络）；

```yaml
network:
  driver: calico
  cidr: 10.244.0.0/16
  gateway: 10.244.0.1
  content: /etc/kubernetes/manifest/calico.yaml
```

### ETCD相关信息

- 集群token；
- 部署ETCD节点的IP和hostname列表；
- 证书配置；
- 数据存储路径；
- `map[string][string]`类型的拓展参数；

```yaml
etcd:
  token: my-etcd
  nodes:
    -
      ip: 192.168.2.2
      hostname: node1
    -
      ip: 192.168.2.3
      hostname: node2
  certs:
    -
      source: /root/certs/ca.crt
      dst: /etc/kubernetes/pki/etcd/ca.crt
  data-dir: /var/data/
  extral-args:
    "--etcd-name": "test"
```

### 控制面

#### API Server组件配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给apiserver的参数；
- 证书支持的SANS列表；
- 超时时间；

```yaml
api-server:
  sans:
    - 192.168.1.1
    - haozi.openeuler.org
  timeout: 5m
  extr-args:
    "--secure-port": 6443
    "--allow-privileged": true
```

#### controller manager组件配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给controller manager的参数；

```yaml
controller-manager:
  extr-args:
    "--cluster-name": "test"
    "--leader-elect": true
```

#### scheduler组件配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给scheduler的参数；

```yaml
scheduler:
  extr-args:
    "--leader-elect": true
```

#### DNS组件配置

- 使用的DNS组件名称，支持coredns；
- 拓展参数，类型为`map[string][string]`，提供传递给组件解析；

以coredns为例，拓展参数至少包括如下信息：

- 提供服务的端口；
- api server的访问endpoint；
- 访问api server需要的kubeconfig；
- DNS服务的VIP；
- 部署coredns的节点IP列表；

```yaml
dns:
  name: coredns
  extr-args:
    "port": 53
    "api-server-endpoint": "https://haozi.openeuler.org:6443"
    "kubeconfig": "/etc/kubernetes/admin.conf"
    "dns-service-vip": "10.32.0.10"
    "nodes-ip-list":
       - "192.168.1.2"
       - "192.168.1.3"
```

### 负载节点

#### kube-proxy组件的配置

- 拓展参数，类型为`map[string][string]`，提供直接透传给kube-proxy的参数；

```yaml
kube-proxy:
  extr-args:
    "--config": "/etc/kubernetes/kube-proxy-config.yaml"
    "--hostname-override": "work1"
```

#### kubelet组件配置

- DNS服务的VIP；
- DNS domain配置，默认为cluster.local；
- pause镜像；
- 网络插件，默认为cni；
- 设置的hostname；
- cni-bin-dir；
- container runtime以及对应的endpoint；
- 拓展参数，类型为`map[string][string]`，提供直接透传给kubelet的参数；

```yaml
kubelet:
  dns-vip: "10.32.0.10"
  dns-domain: "cluster-local"
  pause-image: "pause:3.2"
  network-plugin: "cni"
  cni-bin-dir: "/opt/cni/bin"
  container:
    runtime: "isulad"
    endpoint: "/var/run/isulad.sock"
  hostname: "work1"
  extr-args:
    "--config": "/etc/kubernetes/kubelet_config.yaml"
    "--register-node": "true"
```

#### 容器引擎的相关信息

默认使用iSulad作为集群的容器引擎；

- pause镜像；
- cni-conf-dir;
- cni-bin-dir;
- 镜像下载的repository；
- 代理的配置；

```yaml
container-runtime:
  pause-img: pause:3.2
  cni-conf-dir: "/etc/cni/net.d"
  cni-bin-dir: "/opt/cin/bin"
  repository:
    - docker.io
  proxy:
    http_proxy: "xxxx"
    https_proxy: "xxxx"
    no_proxy: "xxxx"
```

