#                                                eggo操作手册



## 介绍

​    Eggo项目旨在解决大规模生产环境K8S集群自动化部署问题、部署流程跟踪以及提供高度的灵活性。通过结合GitOps管理、跟踪部署配置，通过云原生的方式实现集群的部署，实现集群部署集群的能力。

​    eggo支持在多种常见的linux发行版上部署k8s集群，例如openeuler/centos/ubuntu。

​    eggo支持在集群中混合部署不同架构(x86_64/arm64)的机器。

​    eggo已实现使用命令行的方式进行集群的一键部署，并提供了多种集群部署方式：

1) 支持离线部署。将所有用到的rpm包/二进制文件/插件/容器镜像按照一定的格式打包到一个tar.gz文件中，再编写对应的yaml配置文件(后面有详细介绍)，即可执行命令一键部署。本文后面都以离线安装为例进行说明。

2) 支持在线部署。只需要编写yaml配置文件即可执行命令一键部署，所需的rpm包/二进制文件/插件/容器镜像等都在安装部署阶段自动联网下载。在线部署目前还不支持插件的在线下载安装，后续会支持插件的在线部署。

3) 通过GitOps使用元集群部署新的集群。该功能还在开发中。



## 部署集群

### 1. 准备工作

1) 待安装机器配置好机器的hostname并安装tar命令，确保能使用tar命令解压tar.gz格式的压缩包。配置ssh确保能远程访问，如果ssh登录的是普通用户，还需要确保该用户有免密执行sudo的权限。

1) 在任意一台能连接上述所有机器的机器上，根据 https://gitee.com/openeuler/eggo/blob/master/README.md 的说明编译安装eggo，也可以拷贝编译好的eggo直接使用。

2) 如果是离线部署，则需要准备k8s以及相关的离线包，以openeuler20.09为例，离线包的存放格式如下：

```
$ tree
.
├── addons
│   └── calico.yaml
├── conntrack-tools-1.4.6-1.oe1.aarch64.rpm
├── containernetworking-plugins-0.8.2-4.git485be65.oe1.aarch64.rpm
├── coredns-1.7.0-1.0.oe1.aarch64.rpm
├── docker-engine-18.09.0-115.oe1.aarch64.rpm
├── etcd-3.4.14-2.aarch64.rpm
├── images.tar
├── kubernetes-client-1.20.2-4.oe1.aarch64.rpm
├── kubernetes-kubelet-1.20.2-4.oe1.aarch64.rpm
├── kubernetes-master-1.20.2-4.oe1.aarch64.rpm
├── kubernetes-node-1.20.2-4.oe1.aarch64.rpm
├── libcgroup-0.42.2-1.oe1.aarch64.rpm
├── libnetfilter_cthelper-1.0.0-15.oe1.aarch64.rpm
├── libnetfilter_cttimeout-1.0.0-13.oe1.aarch64.rpm
├── libnetfilter_queue-1.0.5-1.oe1.aarch64.rpm
└── socat-1.7.3.2-8.oe1.aarch64.rpm

1 directory, 16 files
```

-  addons 目录存放各种插件，例如calico网络插件，该目录存放的插件列表由eggo部署时的配置文件deploy.yaml中的字段addons指定，两者必须一致。

- 目录下存放需要安装的rpm/deb包，或者二进制程序或者文件，所有这些安装包或者文件都需要在deploy.yaml中进行了配置才会真正安装。如果是在线部署，则只需要在deploy.yaml中描述对应的安装包并配置为repo方式安装即可，不需要准备对应的安装包。

- images.tar存放部署时需要导入的容器镜像，例如网络插件镜像以及pause镜像。如果是在线安装，则由容器运行时自动从镜像仓库下载，不需要准备images.tar包。以calico插件依赖的容器镜像为例，可以根据calico.yaml里面定义的镜像全名进行下载导出。该tar包包含的镜像必须是使用docker或者isula-build等兼容docker的tar包格式的命令，使用docker save -o images.tar images1:tag images2:tag ......  或类似命令将所有镜像一次性导出到images.tar包中，需要确保执行load镜像时能一次将images.tar导入成功。以上述calico镜像为例，镜像导出命令为：

  ```
  $ docker save -o images.tar calico/node:v3.19.1 calico/cni:v3.19.1 calico/kube-controllers:v3.19.1 calico/pod2daemon-flexvol:v3.19.1 k8s.gcr.io/pause:3.2

- 如果coredns使用pod的方式部署，则images.tar里面需要包含coredns的镜像，而coredns对应的二进制包可以删除。

3)  准备eggo部署时使用的yaml配置文件。可以使用下面的命令生成一个模板配置，并打开yaml文件对其进行增删改来满足不同的部署需求。

```
$ eggo template -f template.yaml
```

另外，可以使用命令行修改机器列表等基本信息。

```
$ eggo template -f template.yaml -n k8s-cluster -u username -p password --masters 192.168.0.1 --masters 192.168.0.2 --nodes 192.168.0.3 --etcds 192.168.0.4 --loadbalancer 192.168.0.5
```

具体的deploy.yaml的配置见最后的"配置文件说明"章节



### 2. 执行命令安装k8s集群

```
$ eggo -d deploy -f deploy.yaml
```

- -d参数表示打印调试信息

- -f参数指定部署时使用的配置文件，不指定的话会从默认文件~/.eggo/deploy.yaml加载配置进行集群安装部署。

  说明：集群部署结束后可以执行命令`echo $?`来判断是否部署成功，输出为0则为部署成功。如果部署失败，则`echo $?`为非0,并且终端也会打印错误信息。

## 清理拆除集群

```
$ eggo -d cleanup -f deploy.yaml
```

- -d参数表示打印调试信息

- -f参数指定部署时使用的配置文件，不指定的话会从默认文件~/.eggo/deploy.yaml加载配置进行集群的清理拆除。

  说明：当前集群拆除过程不会清理容器和镜像，但如果部署时配置了需要安装容器引擎，则容器引擎会被清除，可能会导致容器本身运行异常。另外清理过程中可能会打印一些错误信息，一般都是由于清理过程中操作集群时返回了错误的结果导致，不需要过多关注，集群依然能正常拆除。



## 配置文件说明

下面的配置中，不同节点类型的节点可以同时部署在同一台机器(注意配置必须一致)。

```
cluster-id: k8s-cluster          // 集群名称
username: root                   // 需要部署k8s集群的机器的ssh登录用户名，所有机器都需要使用同一个用户名
password: 123456                 // 需要部署k8s集群的机器的ssh登录密码，所有机器都需要使用同一个密码
masters:                         // 配置master节点的列表，建议每个master节点同时作为node节点，否则master节点可以无法直接访问pod
- name: test0                    // 该节点的名称，会设置该名称为该节点的hostname并设置为k8s集群看到的该节点的名称
  ip: 192.168.0.1                // 该节点的ip地址
  port: 22                       // ssh登录的端口
  arch: arm64                    // 机器架构，x86_64的填amd64
nodes:                           // 配置worker节点的列表
- name: test0                    // 该节点的名称，会设置该名称为该节点的hostname并设置为k8s集群看到的该节点的名称
  ip: 192.168.0.2                // 该节点的ip地址
  port: 22                       // ssh登录的端口
  arch: arm64                    // 机器架构，x86_64的填amd64
- name: test1
  ip: 192.168.0.3
  port: 22
  arch: arm64
etcds:                           // 配置etcd节点的列表，如果该项为空，则将会为每个master节点部署一个etcd，否则只会部署配置的etcd节点
- name: etcd-0                   // 该节点的名称，会设置该名称为该节点的hostname并设置为k8s集群看到的该节点的名称
  ip: 192.168.0.4                // 该节点的ip地址
  port: 22                       // ssh登录的端口
  arch: amd64                    // 机器架构，x86_64的填amd64
loadbalance:                     // 配置loadbalance节点
  name: k8s-loadbalance          // 该节点的名称，会设置该名称为该节点的hostname并设置为k8s集群看到的该节点的名称
  ip: 192.168.0.5                // 该节点的ip地址
  port: 22                       // ssh登录的端口
  arch: amd64                    // 机器架构，x86_64的填amd64
  bind-port: 8443                // 负载均衡服务监听的端口 
external-ca: false                          // 是否使用外部ca证书，该功能还未实现
external-ca-path: /opt/externalca           // 外部ca证书文件的路径
service:                                    // k8s创建的service的配置
  cidr: 10.32.0.0/16                        // k8s创建的service的IP地址网段
  dnsaddr: 10.32.0.10                       // k8s创建的service的DNS地址
  gateway: 10.32.0.1                        // k8s创建的service的网关地址
  dns:                                      // k8s创建的coredns的配置
    corednstype: pod                        // k8s创建的coredns的部署类型，支持pod和binary
    imageversion: 1.8.4                     // pod部署类型的coredns镜像版本
    replicas: 2                             // pod部署类型的coredns副本数量
network:                                    // k8s集群网络配置
  podcidr: 10.244.0.0/16                    // k8s集群网络的IP地址网段
  plugin: calico                            // k8s集群部署的网络插件
  plugin-args: {"NetworkYamlPath": "/etc/kubernetes/addons/calico.yaml"}   // k8s集群网络的网络插件的配置
apiserver-endpoint: 192.168.122.222:6443    // 对外暴露的APISERVER服务的地址或域名，如果配置了loadbalances则填loadbalance地址，否则填写第1个master节点地址
apiserver-cert-sans:                        // apiserver相关的证书中需要额外配置的ip和域名
  dnsnames: []                              // apiserver相关的证书中需要额外配置的域名列表
  ips: []                                   // apiserver相关的证书中需要额外配置的ip地址列表
apiserver-timeout: 120s                     // apiserver响应超时时间
etcd-external: false                        // 使用外部etcd，该功能还未实现
etcd-token: etcd-cluster                    // etcd集群名称
dns-vip: 10.32.0.10                         // dns的虚拟ip地址
dns-domain: cluster.local                   // DNS域名后缀
pause-image: k8s.gcr.io/pause:3.2           // 容器运行时的pause容器的容器镜像名称
network-plugin: cni                         // 网络插件类型
cni-bin-dir: /usr/libexec/cni,/opt/cni/bin  // 网络插件地址，使用","分隔多个地址
runtime: docker                             // 使用哪种容器运行时，目前支持docker和iSulad
registry-mirrors: []                        // 下载容器镜像时使用的镜像仓库的mirror站点地址
insecure-registries: []                     // 下载容器镜像时运行使用http协议下载镜像的镜像仓库地址
config-extra-args: []                       // 各个组件(kube-apiserver/etcd等)服务启动配置的额外参数
addons:                                     // 配置第三方插件
- type: file                                // 插件类型，目前只支持file类型
  filename: xxx.yaml                        // 插件名称，注意需要将对应插件放到tar.gz包中的addons目录下
open-ports:                                 // 配置需要额外打开的端口，k8s自身所需端口不需要进行配置，额外的插件的端口需要进行额外配置
  node:                                     // 指定在那种类型的节点上打开端口，可以是master/node/etcd/loadbalance
  - port: 111                               // 端口地址
    protocol: tcp                           // 端口类型，tcp/udp
  - port: 179
    protocol: tcp
package-src:                                // 配置安装包的详细信息
  type: tar.gz                              // 安装包的压缩类型，目前只支持tar.gz类型的安装包
  armsrc: /root/rpms/pacakges-arm.tar.gz    // arm类型安装包的路径，配置的机器中存在arm机器场景下需要配置
  x86src: /root/rpms/packages-x86.tar.gz    // x86类型安装包的路径，配置的机器中存在x86机器场景下需要配置
pacakges:                                   // 配置各种类型节点上需要安装的安装包或者二进制文件的详细信息，注意将对应文件放到在tar.gz安装包中
  etcd:                                     // etcd类型节点需要安装的包或二进制文件列表
  - name: etcd                              // 需要安装的包或二进制文件的名称，如果是安装包则只写名称，不填写具体的版本号，安装时会使用`$name*`来识别
    type: pkg                               // package的类型，pkg/repo/binary三种类型，如果配置为repo请在对应节点上配置好repo源
    dstpath: ""                             // 目的文件夹路径，binary类型下需要配置，表示将二进制放到节点的哪个目录下，为了防止用户误配置路径，此配置必须符合白名单，参见下一小节
  master:                                   // master类型节点需要安装的包或二进制文件列表
  - name: kubernetes-client
    type: pkg
    dstpath: ""
  - name: kubernetes-master
    type: pkg
    dstpath: ""
  - name: coredns
    type: pkg
    dstpath: ""
  - name: addons
    type: binary
    dstpath: /etc/kubernetes
  node:                                     // worker类型节点需要安装的包或二进制文件列表
  - name: libnetfilter_cthelper
    type: pkg
    dstpath: ""
  - name: libnetfilter_cttimeout
    type: pkg
    dstpath: ""
  - name: libnetfilter_queue
    type: pkg
    dstpath: ""
  - name: conntrack-tools
    type: pkg
    dstpath: ""
  - name: socat
    type: pkg
    dstpath: ""
  - name: containernetworking-plugins
    type: pkg
    dstpath: ""
  - name: libcgroup
    type: pkg
    dstpath: ""
  - name: docker-engine
    type: pkg
    dstpath: ""
  - name: kubernetes-client
    type: pkg
    dstpath: ""
  - name: kubernetes-node
    type: pkg
    dstpath: ""
  - name: kubernetes-kubelet
    type: pkg
    dstpath: ""
  loadbalance:                               // loadbalance类型节点需要安装的包或二进制文件列表
  - name: nginx
    type: pkg
    dstpath: ""
  - name: gd
    type: pkg
    dstpath: ""
  - name: gperftools-libs
    type: pkg
    dstpath: ""
  - name: libunwind
    type: pkg
    dstpath: ""
  - name: libwebp
    type: pkg
    dstpath: ""
  - name: libxslt
    type: pkg
    dstpath: ""
  - name: nginx-all-modules
    type: pkg
    dstpath: ""
  - name: nginx-filesystem
    type: pkg
    dstpath: ""
  - name: nginx-mod-http-image-filter
    type: pkg
    dstpath: ""
  - name: nginx-mod-http-perl
    type: pkg
    dstpath: ""
  - name: nginx-mod-http-xslt-filter
    type: pkg
    dstpath: ""
  - name: nginx-mod-mail
    type: pkg
    dstpath: ""
  - name: nginx-mod-stream
    type: pkg
    dstpath: ""
```

### dstpath 白名单
dstpath可以配置为白名单中的目录，或者其子目录
```
"/usr/bin", "/usr/local/bin", "/opt/cni/bin", "/usr/libexec/cni",
"/etc/kubernetes",
"/usr/lib/systemd/system", "/etc/systemd/system",
"/tmp",
```

## 部署相关问题汇总

### calico网络CIDR配置问题

下载[calico官网的配置](https://docs.projectcalico.org/manifests/calico.yaml)，然后直接部署calico网络，会发现pod的网络区间不是我们eggo配置的`podcidr`。而如果通过kubeadm部署集群，然后使用同样的calico配置部署calico网络，会发现pod的网络区间为kubeadm设置的`pod-network-cidr`。

#### 原因

由于`calico/node`容器镜像对kubeadm做了适配，会自动读取kubeadm设置的kubeadm-config的configmaps的值，然后自动更新cidr。

#### 解决方法

eggo的二进制部署方式没办法修改`calico/node`进行配置，因此，建议直接修改`calico.yaml`配置。使能`CALICO_IPV4POOL_CIDR`项，并且设置为k8s集群的`podcidr`相同的值。

`calico.yaml`默认值如下：
```bash
containers:
  # Runs calico-node container on each Kubernetes node. This
  # container programs network policy and routes on each
  # host.
  - name: calico-node
    image: docker.io/calico/node:v3.19.1
    envFrom:
    - configMapRef:
        # Allow KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT to be overridden for eBPF mode.
        name: kubernetes-services-endpoint
        optional: true
    env:
      # The default IPv4 pool to create on startup if none exists. Pod IPs will be
      # chosen from this range. Changing this value after installation will have
      # no effect. This should fall within `--cluster-cidr`.
      # - name: CALICO_IPV4POOL_CIDR
      #   value: "192.168.0.0/16"
```

按照集群cidr（例如为"10.244.0.0/16"），那么修改`calico.yaml`如下：
```bash
containers:
  # Runs calico-node container on each Kubernetes node. This
  # container programs network policy and routes on each
  # host.
  - name: calico-node
    image: docker.io/calico/node:v3.19.1
    envFrom:
    - configMapRef:
        # Allow KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT to be overridden for eBPF mode.
        name: kubernetes-services-endpoint
        optional: true
    env:
      # The default IPv4 pool to create on startup if none exists. Pod IPs will be
      # chosen from this range. Changing this value after installation will have
      # no effect. This should fall within `--cluster-cidr`.
      - name: CALICO_IPV4POOL_CIDR
         value: "10.244.0.0/16"
```
