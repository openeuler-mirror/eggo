# eggops使用手册

### 1. 准备工作

1) 搭建元集群

通过eggo部署元集群，需要至少一个worker节点。

2) 在元集群中安装eggops

```bash
# 元集群的kubeconfig文件路径
$ export KUBECONFIG=/etc/kubernetes/admin.conf
$ kubectl apply -f eggops.yaml
```

3) 准备yaml文件

需要准备`namespace`、`secret`、`persistentvolume`、`persistentvolumeclaim`等k8s原生资源，同时也需要准备`infrastructure`、`machine`、`cluster`用户自定义资源。通过`kubectl apply`命令将其发布到k8s集群中，`controllers`便会根据用户需求分配机器拉起集群。

- namespace.yaml

创建`namespace`，后续的`secret`、`persistentvolume`、`persistentvolumeclaim`、`infrastructure`、`machine`、`cluster`均在该命名空间中创建，此处以`eggo-system` namespace为例。
 
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: eggo-system
```

> 用户可以跳过`namespace`的创建，并且在创建`secret`、`persistentvolume`等资源时不指定`namespace`字段，此时这些资源将会在默认命名空间`default`中创建。

namespace参考资料：https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/

- secret.yaml

创建机器ssh login所需的密钥。下面介绍通过username/password登录，以及通过private key登录两种方式。

`kubernetes.io/basic-auth`，基本身份认证类型。需要指定data字段，data字段必须包含username和password两个键，对应的值是通过 base64 编码的字符串。也可以指定stringData字段，其username和password对应的值是明文字符串。
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-example
  namespace: eggo-system
type: kubernetes.io/basic-auth
stringData:
  username: root
  password: 123456
```

`kubernetes.io/ssh-auth`，ssh身份认证。需要在data或者stringData字段，提供ssh-privatekey键及其对对应的值。
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-ssh-auth-example
  namespace: eggo-system
type: kubernetes.io/ssh-auth
data:
  # 此例中的实际数据被截断
  ssh-privatekey: MIIEpQIBAAKCAQEAulqb/Y ...
```

secret参考资料：https://kubernetes.io/docs/concepts/configuration/secret/

- persistentvolume.yaml与persistentvolumeclaim.yaml

创建PV与PVC，通过其可以将部署过程中所需的Package包挂载到eggo容器中去。下面介绍文件服务器nfs挂载，以及本地存储local挂载两种方式，其他PV与PVC类型可以参考k8s官方文档。

nfs挂载方式，首先需要搭建nfs文件服务器。
```bash
# 创建共享文件夹
$ mkdir -p /data
# 准备package包
$ tree /data
/data
└── packages
    ├── packages-arm64.tar.gz
    └── packages-amd64.tar.gz

1 directory, 2 files

# 安装 nfs 与 rpc 相关软件包：
$ yum install nfs-utils rpcbind -y

# NFS默认的配置文件是 /etc/exports，修改配置文件
$ cat /etc/exports
/data  *(rw,sync,no_root_squash,no_all_squash)

$ systemctl enable rpcbind && systemctl restart rpcbind
$ systemctl enable nfs && systemctl restart nfs

# 配置防火墙
$ firewall-cmd --permanent --add-service=nfs
success   
$ firewall-cmd  --reload 
success

# 查看NFS分享的资源
$ showmount -e <nfs ip>
Export list for <nfs ip>:
/data *
```

准备PV和PVC的yaml文件
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: nfs-pv-example
  namespace: eggo-system
  labels:
    type: nfs
spec:
  capacity:
    storage: 500Mi
  accessModes:
    - ReadOnlyMany
  nfs:
    server: 192.168.0.123
    path: "/data"
  persistentVolumeReclaimPolicy: Retain

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-pvc-example
  namespace: eggo-system
spec:
  accessModes:
    - ReadOnlyMany
  resources:
    requests:
      storage: 500Mi
  selector:
    matchLabels:
      type: "nfs"
```

local挂载方式。由于是本地挂载，所以读写性能比远程网络挂载的更优。但缺点是，Pod无法调度受到限制，必须与node强绑定；同时，如果节点或者磁盘异常，则使用该volume的Pod也会异常。

在所有或部分节点上准备package包
```bash
# 创建共享文件夹
$ mkdir -p /data
# 准备package包
$ tree /data
/data
└── packages
    ├── packages-arm64.tar.gz
    └── packages-amd64.tar.gz

1 directory, 2 files
```

准备PV和PVC的yaml文件
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: local-pv-example
  namespace: eggo-system
  labels:
    type: local
spec:
  capacity:
    storage: 500Mi
  volumeMode: Filesystem
  accessModes:
  - ReadOnlyMany
  persistentVolumeReclaimPolicy: Retain
  local:
    path: /data
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          # 准备/data目录及package包的节点
          - node1-example
          - node2-example

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: local-pvc-example
  namespace: eggo-system
spec:
  accessModes:
    - ReadOnlyMany
  resources:
    requests:
      storage: 500Mi
  selector:
    matchLabels:
      type: "local"
```
注：pv声明时指定了节点亲和性，使用该pv的Pod在调度时，会根据节点亲和性设置选择正确的节点去运行。

pv和pvc参考资料：https://kubernetes.io/docs/concepts/storage/volumes/

- machine.yaml

machine为eggops创建的用户自定义资源，用来描述可用的待部署机器。通过machine.yaml可以将machine的基本信息记录下来，在部署cluster时会选择合适的机器进行部署。当machine被分配给cluster使用后，则不可以再被其他cluster使用。当部署的cluster被删除，其使用的machine会被释放，可以再被其他cluster使用

```yaml
apiVersion: eggo.isula.org/v1
kind: Machine
metadata:
  name: machine1-example
  labels:
    masterRole: allow
    workerRole: allow
spec:
  hostname: machine1
  arch: arm64
  ip: 192.168.0.1
  port: 22
```

machine的基本信息包括hostname、架构、ip、ssh登录端口，与eggo config中的节点的配置是一致的，详细说明可以参考manual.md文档中的eggo配置。

- infrastructure.yaml

infrastructure为eggops创建的用户自定义资源，用来描述cluster的基础设施，包括package包的共享存储卷、安装配置、暴露端口等等。大多数集群的基础设施配置是一样的，因此不同的cluster可以指定相同的infrastructure。

```yaml

apiVersion: eggo.isula.org/v1
kind: Infrastructure
metadata:
  name: infrastructure-example
  namespace: eggo-system
spec:
  # 用于将package包挂载到容器中，部署集群时使用
  packagePersistentVolumeClaim:
    name: nfs-pvc-example
  # 暴露端口，可选项
  open-ports:
    worker:
    - port: 111
      protocol: tcp
    - port: 179
      protocol: tcp
  # 指定所需的安装包
  install:
    package-source:
      type: tar.gz
      srcPackages:
        # package包在nfs下的路径
        arm64: packages/packages-arm.tar.gz
    image:
    - name: images.tar
      type: image
    dns:
    - name: coredns
      type: pkg
    addition:
      master:
      - name: calico.yaml
        type: yaml

```

open-ports暴露端口、install包安装配置与eggo config中的open-ports、install配置是一致的，详细说明可以参考manual.md文档中的eggo配置。

- cluster.yaml

cluster为eggops创建的用户自定义资源，用来描述k8s集群的信息等等。根据配置的k8s集群信息，元集群中会选择合适的machine，创建一个job，拉起一个Pod，通过eggo deploy命令部署一个k8s集群。当delete cluster时，与创建的流程相似，创建job，拉起Pod，通过eggo cleanup命令清除部署的k8s集群。

```yaml
apiVersion: eggo.isula.org/v1
kind: Cluster
metadata:
  name: cluster-example
  namespace: eggo-system
spec:
  # 所需master节点的描述。此处描述为，选取有masterRole: allow这一label的machine中的1台作为master节点
  masterRequire:
    # master节点的数量
    number: 1
    # master节点的features
    features: 
      masterRole: allow
  # 所需worker节点的描述
  workerRequire:
    number: 1
    features:
      workerRole: allow
  # 所需loadbalance节点的描述，可选项
  loadbalanceRequires:
    number: 1
    features:
      lbRole: allow
  # loadbalance服务监听端口，可选项
  loadbalance-bindport: 8443
  # Pod亲和性调度，可选项
  eggoAffinity:
    nodeAffinity:
      requiredDuringSchedulingRequiredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/arch
            operator: In
            values:
            - arm64
  machineLoginSecret:
    name: secret-example
  infrastructure:
    name: infrastructure-example
  runtime:
    runtime: iSulad
    runtime-endpoint: unix:///var/run/isulad.sock
  # 启用kubelet serving证书
  enableKubeletServing: true
  # 集群网络配置，可选项
  network:
    # k8s创建的service的IP地址网段
    service-cidr: 10.32.0.0/16
    # k8s创建的service的DNS地址
    service-dns-ip: 10.32.0.10
    # k8s创建的service的网关地址
    service-gateway: 10.32.0.1
    # k8s集群网络的IP地址网段
    pod-cidr: 10.244.0.0/16
    # k8s集群部署的网络插件
    pod-plugin: calico
    # k8s集群网络的网络插件的配置
    pod-plugin-args:
      NetworkYamlPath: /etc/kubernetes/addons/calico.yaml
  # eggo镜像版本，可选项，默认为eggo:<version>
  eggoImageVersion: "eggo:latest"
```

masterRequire、workerRequire与loadbalanceRequires中的features字段，可以在选择machine时通过LabelSelector筛选出合适的机器。eggoAffinity，设置亲和性调度，可以将执行eggo命令的Pod调度到某些特定机器上运行。

其他未特殊说明的配置与eggo config中的配置是一致的，详细说明可以参考manual.md文档中的eggo配置。

Pod亲和性调度参考资料：https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/

4) 部署集群

```bash
$ kubectl apply -f namespace.yaml
$ kubectl apply -f secret.yaml
$ kubectl apply -f persistentvolume.yaml
$ kubectl apply -f persistentvolumeclaim.yaml
$ kubectl apply -f machine.yaml
$ kubectl apply -f infrastructure.yaml
$ kubectl apply -f cluster.yaml

# 也可以将其放在同一个文件中apply一次
$ kubectl apply -f eggops_cluster.yaml
```

5) 销毁集群
```bash
# wait=false不会在前端等待cluster删除完成
kubectl delete -f cluster.yaml --wait=false

# 其他资源根据用户需求可以自行选择删除，或者不删除待下次使用
```

6) 卸载CRD与controller 

在卸载CRD与controller之前，需要将所有创建的资源全部删除。

```bash
# 元集群的kubeconfig文件路径
$ export KUBECONFIG=/etc/kubernetes/admin.conf
$ kubectl delete -f eggops.yaml
```
