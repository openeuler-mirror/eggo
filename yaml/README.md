# eggops集群配置

本目录存放了一些通过eggops启动集群的资源配置，方便用户使用。

## 使用方法

用户根据实际使用需求进行修改，主要修改的内容包括：

### 修改base.yaml
- Secret，配置机器ssh登录所需的账号/密码
- PV与PVC，配置共享数据卷将package包挂载到容器中
- Infrastructure，集群的基础设施配置，包括package PVC、暴露端口、install安装包等等，详细配置可以参考docs/eggops.md文档

### 修改machines.yaml
- Machine，配置IP、架构、ssh登录端口等信息，每一个Machine对应一台可用的机器，详细配置可以参考docs/eggops.md文档

### 修改cluster.yaml
- Cluster，配置集群所需的master数量、worker数量、登录密钥、基础设施等信息，详细配置可以参考docs/eggops.md文档

### 创建集群

```bash
$ kubectl apply -f base.yaml
$ kubectl apply -f machines.yaml
$ kubectl apply -f cluster.yaml
```

### 删除集群

```bash
$ kubectl delete -f cluster.yaml
$ kubectl delete -f machines.yaml
$ kubectl delete -f base.yaml
```
