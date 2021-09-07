# eggops

### 介绍
eggops是以kubebuilder生成代码框架为基础而编写的，其主要功能是通过operator实现多k8s集群的自动化部署。通过eggops，用户可以以CRD定义machine、cluster资源，通过controller自动为cluster分配machine，并部署k8s集群，从而实现集群部署集群的能力。

### 编译安装

1. 直接运行
```bash
# 指定元集群kubeconfig文件
$ export KUBECONFIG=/etc/kubernetes/admin.conf
# 部署CRDs
$ make install
# 运行controller，会一直前台执行
$ make run ENABLE_WEBHOOKS=false
# 停止运行controller
CTRL C
# 删除CRDs
$ make uninstall
```

2. 在集群中运行
```bash
# 任一台机器
# 登录镜像仓库
$ docker login <registry> -u <username>
# 构建镜像并推送
$ make docker-build docker-push IMG=<registry>/<project-name>:<tag>

# 元集群master节点
# 指定元集群kubeconfig文件
$ export KUBECONFIG=/etc/kubernetes/admin.conf
# 将控制器部署到集群中
$ make deploy IMG=<registry>/<project-name>:<tag>
# 卸载控制器
$ make undeploy
```

### 使用方法

详细的用法见 https://gitee.com/openeuler/eggo/blob/master/docs/eggops.md

### 常见问题

1. 直接运行时，eggops会监听当前机器的8080端口。如果当前机器开启了coredns服务，则可能8080端口已被其占用，`make run ENABLE_WEBHOOKS=false`失败。  
解决办法：建议将coredns服务已pod方式部署；建议eggops部署在集群中运行而非直接运行；（不推荐）修改eggops/main.go。

2. 构建镜像时，由于网络导致`gcr.io/distroless/static:nonroot`镜像无法下载，需要开启代理。同时在Dockerfile文件中，go mod下载默认配置了`GOPROXY="https://goproxy.cn,direct"`，用户可以自行修改或去除。

3. make deploy部署控制器时，除了会下载用户指定的image，还会下载镜像`gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0`，该镜像由于网络原因同样可能pull失败，需要开启代理

4. 开发过程中，如果重新build并push镜像，但镜像name:tag未改变，则在部署控制器到集群时，需要去worker节点上将上次pull下来的镜像rmi，避免使用旧的镜像。
