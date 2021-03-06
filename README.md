# eggo

### 介绍
Eggo项目旨在解决大规模生产环境K8S集群自动化部署问题、部署流程跟踪以及提供高度的灵活性。通过结合GitOps管理、跟踪部署配置，通过云原生的方式实现集群的部署，实现集群部署集群的能力。

- 支持在多种常见的linux发行版本上部署k8s集群：例如openEuler/CentOS/Ubuntu；
- 支持多架构部署，一个集群支持多种架构(amd64/arm64等)的节点；
- 支持多种部署方式：二进制和kubeadm（待实现）；
- 支持在线部署、离线部署以及使用Gitops进行集群部署集群；


目前，eggo已实现使用命令行的方式进行集群的一键部署，以下为支持的三种集群部署方式：

1) 在线部署。只需要编写yaml配置文件即可执行命令一键部署。所需的rpm包/二进制文件/插件/容器镜像等都在安装部署阶段自动联网下载（其中有来自Google的下载源，因此使用机器需要可以访问外网）。在线部署目前还不支持插件的在线下载安装，后续会支持插件的在线部署。【具体操作见[eggo操作手册](./docs/manual.md)】

2) 离线部署。将所有用到的rpm包/二进制文件/插件/容器镜像按照一定的格式打包到一个tar.gz文件中。再编写对应的yaml配置文件([操作手册](./docs/manual.md) 中有详细介绍)，即可执行命令一键部署。

3) 通过GitOps使用元集群部署新的集群。该功能还在开发中。


### 软件架构

详见[软件架构说明](./docs/general_design.md)

### 详细用法

详见[eggo操作手册](https://docs.openeuler.org/zh/docs/21.09/docs/Kubernetes/eggo%E8%87%AA%E5%8A%A8%E5%8C%96%E9%83%A8%E7%BD%B2.html)

### 发布版本

```
# Step 1: 升级VERSION文件版本号，并且合入修改
$ vi VERSION
# Step 2: 通过脚本获取release信息
$ ./hack/releasenote.sh
```

### 感谢

本项目受[Kubekey](https://github.com/kubesphere/kubekey)的启发，感谢Kubekey的伟大工作。

### 参与贡献

1.  Fork 本仓库
2.  新建 Feat_xxx 分支
3.  提交代码
4.  新建 Pull Request
