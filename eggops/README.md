# eggops

### 介绍

eggops是以kubebuilder生成代码框架为基础而编写的，其主要功能是通过operator实现多k8s集群的自动化部署。通过eggops，用户可以以CRD定义machine、cluster资源，通过controller自动为cluster分配machine，并部署k8s集群，从而实现集群部署集群的能力。

### 调试

直接运行controller，仅用于调试，不适用生产场景
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

### 编译安装

1. 制作controller镜像。在任意一台机器上，准备eggops源码、docker容器引擎
```bash
# 登录镜像仓库
$ docker login <registry> -u <username>
# 构建amd64/arm64架构的 eggops controller 镜像并推送到镜像仓库.
$ make docker-build docker-push ARCH=<arch> IMG=<registry>/<project-name>:<tag>
```

2. 生成eggops controller yaml文件
```bash
# 生成eggops.yaml
$ make yaml IMG=<registry>/<project-name>:<tag>
```

3. 在集群中运行controller

- 方式一
```bash
# 在元集群master节点
# 指定元集群kubeconfig文件
$ export KUBECONFIG=/etc/kubernetes/admin.conf
# 将控制器部署到集群中
$ kubectl apply -f eggops.yaml
# 卸载控制器
$ kubectl delete -f eggops.yaml
```

- 方式二，不推荐
```bash
# 元集群master节点，准备eggops源码，可跳过eggops controller yaml文件的生成
# 指定元集群kubeconfig文件
$ export KUBECONFIG=/etc/kubernetes/admin.conf
# 将控制器部署到集群中
$ make deploy IMG=<registry>/<project-name>:<tag>
# 卸载控制器
$ make undeploy
```

### 使用方法

详细的用法见 https://gitee.com/openeuler/eggo/blob/master/docs/eggops.md

### 开发注意事项

1. 每次eggops修改完成之后，需要更新仓库eggops.yaml文件，更新方法`cd eggops && make yaml IMG=<registry>/<project-name>:<tag> && cp eggops.yaml ../`
2. 使用eggops.yaml部署eggops-controller，默认情况下，关闭`eggops-controller leader-elect`能力，恢复可修改文件`eggops/config/manager/manager.yaml`
3. 使用eggops.yaml部署eggops-controller，默认情况下，关闭`kube-rbac-proxy`组件，恢复可修改文件`eggops/config/default/kustomization.yaml`
4. 每次eggops修改完成之后，需要更新eggops镜像，更新方法` make docker-build docker-push ARCH=<arch> IMG=hub.oepkgs.net/haozi007/eggops-amd64:<tag>`

### 常见问题

1. 直接运行时，eggops会监听当前机器的8080端口。如果当前机器开启了coredns服务，则可能8080端口已被其占用，`make run ENABLE_WEBHOOKS=false`失败。  
解决办法：建议将coredns服务已pod方式部署；建议eggops部署在集群中运行而非直接运行；（不推荐）修改eggops/main.go。

2. 构建镜像时，由于网络导致`gcr.io/distroless/static:nonroot`镜像无法下载，需要开启代理。同时在Dockerfile文件中，go mod下载默认配置了`GOPROXY="https://goproxy.cn,direct"`，用户可以自行修改或去除。

3. make deploy部署控制器时，除了会下载用户指定的image，还会下载镜像`gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0`，该镜像由于网络原因同样可能pull失败，需要开启代理

4. 开发过程中，如果重新build以及push `controller`镜像，但镜像name:tag未改变，则在部署控制器到集群时，需要在worker节点上`rmi`旧的镜像，保证部署时重新拉取`controller`镜像。

5. `kubebuilder`部署`controller`时，同时会部署一个`gcr.io/kubebuilder/kube-rbac-proxy`容器。k8s集群中Pod可以向集群中的任何Pod发送request，而该容器可以通过RBAC认证或者TLS证书来严格控制外部请求访问Pod。这是一个可选的组件，eggops controller默认将其关闭，通过在eggops/config/default/kustomization.yaml文件中添加`- manager_auth_proxy_patch.yaml`，可以将该组件开启。
