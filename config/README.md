# 集群配置

config目录存放了多个集群config模板，方便用户快速配置集群

## config模板

| name | OS | arch | masterNum | workerNum | loadbalance | Runtime | Install |
| --- | --- | --- | --- | --- | --- | --- | --- |
| centos.config | CentOS7 | amd64 | 1 | 2 | No | docker | bin |
| openEuler.config | openEuler 21.03 | arm64 | 1 | 2 | No | iSulad | repo + rpm |

## 使用方法

### 修改config
用户根据自己的需求，选择合适的集群配置模板，并进行修改。主要修改的内容包括：
- 用户名和密码 username/password
- 各个节点的域名、ip、架构 name/ip/arch
- apiserver-endpoint,如果设置了loadbalance则为loadbalance的IP:bind-port，如果没有则为第一个master的IP:6443
- 安装软件与压缩包 packages与packages-src

config其他详细的配置请参见eggo操作手册

### 准备packages压缩包
用户根据config中的packages配置准备离线安装包。以openEuler.config为例，压缩包需要包含
```
$ tree /root/pacakges
/root/pacakges
├── dir
│   └── addons
│       └── calico.yaml
├── image
│   └── images.tar
└── pkg
    └── coredns-1.7.0-1.0.oe1.aarch64.rpm

4 directories, 3 files
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
