存放部署自动化相关的脚本

```bash
# 根据实际集群机器配置configs文件
$ cat configs
MASTER_IPS=("192.168.1.1" "192.168.1.2") # 两台master机器，ip分别为"192.168.1.1"和"192.168.1.2"
MASTER_NAMES=("master0" "master1") # 两台master机器，hostname分别为"master0"和"master1"
NODE_IPS=("192.168.1.3" "192.168.1.4") # 两台node机器，ip分别为"192.168.1.3"和"192.168.1.4"
NODE_NAMES=("node1" "node2") # 两台node机器，hostname分别为"node1"和"node2"
SERVICE_CLUSTER_IP_RANGE="10.32.0.0/16" # 集群服务的IP地址区间
SERVICE_CLUSTER_IP_GATEWAY="10.32.0.1" # 集群的服务IP网关地址
CLUSTER_IP_RANGE="10.244.0.0/16" # 集群运行Pod的IP地址区间，如果使用CNI网络，则会被CNI的网络分配替换
API_SERVER_EXPOSE_IP="192.168.1.1" # api server暴露的IP，可以为master节点IP或者LB IP
API_SERVER_EXPOSE_PORT="6443" # api server暴露的端口，master apiserver服务默认端口为6443，LB端口由用户定义
CA_ROOT_PATH="/etc" # 证书生成的目录
EXTRA_SANS=("10.10.1.2") # 其他api server的SANS，例如LB的VIP或者对外IP、域名等等
BOOTSTRAP_NODE_USER="root" # 节点的ssh登录的用户名
BOOTSTRAP_NODE_PASSWORD="123456" # 节点ssh登录的密码
ETCD_CLUSTER_TOKEN="kubernetes_etcd" # ETCD集群的token
ETCD_CLUSTER_IPS=("192.168.1.1" "192.168.1.2") # ETCD集群的节点IP地址列表
ETCD_CLUSTER_NAMES=("master0" "master1") # ETCD集群的节点hostname地址列表
NODE_SERVICE_CLUSTER_DNS="10.32.0.10" # K8S集群的服务集群的DNS服务所在的IP地址
NODE_KUBE_CLUSTER_CIDR="10.244.0.0/16" # K8S集群的IP所在区间
MODULE_SAVE_PATH="/root/rpms" # 存放手动安装的二进制资源，如etcd，k8s等rpm

# 1. 修改上面的configs文件为期望集群的配置
# 2. 在master节点，准备好当前的部署工具
# 3. cd eggo/deploy/tools
# 4. 在第一台master节点执行如下面命令
$ ./deploy.sh install-cluster
# 5. 等待集群完成部署

# 清理整个集群
$ ./deploy.sh clean-cluster
```

注意：
1. 当前openeuler的20.09版本，暂时没有k8s,etcd,coredns等组件，所以需要手动从[社区的repo](https://repo.openeuler.org/openEuler-21.03/everything/)下载这些组件的rpm包放到$MODULE_SAVE_PATH。

在$MODULE_SAVE_PATH目录下面应该存在如下rpm包：
```bash
$ ls $MODULE_SAVE_PATH
coredns-1.7.0-1.0.oe1.aarch64.rpm kubernetes-kubeadm-1.20.2-3.oe1.aarch64.rpm
etcd-3.4.14-2.aarch64.rpm kubernetes-kubelet-1.20.2-3.oe1.aarch64.rpm
kubernetes-client-1.20.2-3.oe1.aarch64.rpm kubernetes-master-1.20.2-3.oe1.aarch64.rpm
kubernetes-help-1.20.2-3.oe1.aarch64.rpm kubernetes-node-1.20.2-3.oe1.aarch64.rpm
```

2. 部署过程中关闭所以节点的代理，或者将机器ip加入环境变量no_proxy中

# 辅助信息
## 节点的dnf软件包源

如果使用的虚拟机，有挂载openeuler的ISO文件的话，可以直接使用cdrom作为dnf的软件源。

```bash
$ mkdir /mnt/cdrom
$ mount /dev/sr0 /mnt/cdrom
$ cp /etc/yum.repos.d/openEuler.repo /etc/yum.repos.d/openEuler.repo.bak
# 修改/etc/yum.repos.d/openEuler.repo如下
$ cat /etc/yum.repos.d/openEuler.repo
[cdrom]
name=cdrom
baseurl=file:///mnt/cdrom
enable=1
gpgcheck=0
```

## 部署calico网络

通过pod的方式部署calico的网络插件，具体方案如下：

```bash
$ wget --no-check-certificate https://docs.projectcalico.org/manifests/calico.yaml
# 开始部署
kubectl apply -f calico.yaml
```

## kube-apiserver与loadbalancer
访问kube-apiserver可以直接访问master节点的6443端口，也可以通过访问loadbalancer将请求转发给kube-apiserver，这取决于configs的配置
- 直接访问apiserver  
将API_SERVER_EXPOSE_IP配置为master节点的IP，将API_SERVER_EXPOSE_PORT配置为apiserver默认的暴露端口6443。通过 ./deploy.sh install-cluster 部署k8s集群
- 访问外部准备好的loadbalancer  
将API_SERVER_EXPOSE_IP配置为loadbalancer节点的IP，将API_SERVER_EXPOSE_PORT配置为loadbalancer的暴露端口，在EXTRA_SANS中加入loadbalancer节点的IP。通过 ./deploy.sh install-cluster 部署k8s集群
- 通过本脚本部署一个loadbalancer并访问
将API_SERVER_EXPOSE_IP配置为想要部署loadbalancer节点的IP，将API_SERVER_EXPOSE_PORT配置为想要暴露端口，在EXTRA_SANS中加入节点IP。通过 ./deploy.sh install-cluster-with-lb 部署loadbalancer与k8s集群。清理时通过./deploy.sh clean-cluster-with-lb清理loadbalancer与k8s集群

## 故障排查
- 生成的日志文件位于执行脚本机器的`/tmp/.k8s`目录下，如部署过程中遇到错误，可以在该目录下查看日志，进行定位。
- 如果部署calico网络，calico-node网络报`BGP not establised...`，需要打开calico相关的端口：
    ```bash
    firewall-cmd --zone=public --add-port=111/tcp
    firewall-cmd --zone=public --add-port=179/tcp
    ```
