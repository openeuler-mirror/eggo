# eggo集群配置

config目录存放了多个集群config模板，方便用户快速配置集群

## config模板

| name | OS | arch | masterNum | workerNum | loadbalance | Runtime | Install |
| --- | --- | --- | --- | --- | --- | --- | --- |
| centos.config | CentOS7 | amd64 | 1 | 2 | No | docker | bin |
| openEuler.config | openEuler 21.03 | arm64 | 1 | 2 | No | iSulad | repo + rpm |
| all_online_install.config | openEuler21.09 | arm64 | 1 | 2 | No | iSulad | repo |

## 使用方法

### 修改config
用户根据自己的需求，选择合适的集群配置模板，并进行修改。主要修改的内容包括：
- 用户名和密码 username/password
- 各个节点的域名、ip、架构 name/ip/arch
- apiserver-endpoint,如果设置了loadbalance则为loadbalance的IP:bind-port，如果没有则为第一个master的IP:6443
- 安装软件与压缩包 packages与packages-src

config其他详细的配置请参见[配置说明](../docs/configuration_file_description.md)

### 准备packages压缩包

用户根据config中的packages配置准备离线安装包。以openEuler.config为例，压缩包需要包含
```
$ tree /root/packages
/root/packages
├── file
│   └── calico.yaml
├── image
│   └── images.tar
└── pkg
    └── coredns-1.7.0-1.0.oe1.aarch64.rpm

3 directories, 3 files
```

其中images.tar为集群部署中使用到的镜像，包含
```
REPOSITORY                  TAG       IMAGE ID       CREATED         SIZE
calico/node                 v3.19.1   c4d75af7e098   5 weeks ago     168MB
calico/pod2daemon-flexvol   v3.19.1   5660150975fb   5 weeks ago     21.7MB
calico/cni                  v3.19.1   5749e8b276f9   5 weeks ago     146MB
calico/kube-controllers     v3.19.1   5d3d5ddc8605   5 weeks ago     60.6MB
k8s.gcr.io/pause            3.2       80d28bedfe5d   16 months ago   683kB
```
用户也可以不准备images.tar，在部署过程中由容器引擎自行pull镜像

打包压缩生成packages压缩包
```
$ cd /root/packages
$ tar -zcvf packages-arm.tar.gz ./*
```
如果混部ARM和X86两种架构，则需再准备一份X86架构的packages压缩包

### 做好notes说明文件（可选）

除了config文件和packages压缩包之外，还需要一个说明文件用于说明packages压缩包中文件的来源，便于用户溯源。以`centos.config`为例，对应的notes文件如下：

```bash
1.  ETCD
    - etcd,etcdctl
    - 架构：x86
    - 版本：3.5.0
    - 地址：https://github.com/etcd-io/etcd/releases/download/v3.5.0/etcd-v3.5.0-linux-amd64.tar.gz

2. Docker Engine
    - containerd,containerd-shim,ctr,docker,dockerd,docker-init,docker-proxy,runc
    - 架构：x86
    - 版本：19.03.0
    - 地址：https://download.docker.com/linux/static/stable/x86_64/docker-19.03.0.tgz

3. Kubernetes
    - kube-apiserver,kube-controller-manager,kube-scheduler,kubectl,kubelet,kube-proy
    - 架构：x86
    - 版本：1.21.1
    - 地址：https://www.downloadkubernetes.com/

4. network
    - bandwidth,dhcp,flannel,host-local,loopback,portmap,sbr,tuning,vrf,bridge,firewall,host-device,ipvlan,macvlan,ptp,static,vlan
    - 架构：x86
    - 版本：v0.9.1
    - 地址：https://github.com/containernetworking/plugins/releases/download/v0.9.1/cni-plugins-linux-amd64-v0.9.1.tgz

```

### 总结

Eggo的离线部署包应该包括三个部分，以`kubernetes-1.21.tar.gz`为例：

```bash
$ tar -tvf kubernetes-1.21.tar.gz
centos.config
packages/packages-x86.tar.gz
packages/packages-arm.tar.gz
notes
```

# eggops集群配置

eggops_cluster.yaml存放了通过eggops启动集群的一些资源配置，方便用户使用。

## 使用方法

### 修改eggops_cluster.yaml
用户根据实际使用需求进行修改，主要修改的内容包括：
- Machine，每一个可用的机器配置一个Machine资源，修改IP、架构、ssh登录端口等
- Secret，配置机器ssh登录所需的账号/密码
- PV与PVC，通过共享数据卷将package包挂载到容器中
- Infrastructure，集群的基础设施配置，包括package PVC、暴露端口、install安装包等等
- Cluster，集群所需的master数量、worker数量、登录密钥、基础设施等信息

eggops_cluster.yaml的详细配置可以参考docs/eggops.md文档

### 准备packages压缩包

用户准备离线安装包，步骤与eggo配置中准备packages压缩包的步骤一致，此处不再赘述。
