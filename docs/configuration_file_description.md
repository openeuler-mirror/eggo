# 配置文件说明

下面的配置中，不同节点类型的节点可以同时部署在同一台机器(注意配置必须一致)。

```
cluster-id: k8s-cluster           // 集群名称
username: root                    // 需要部署k8s集群的机器的ssh登录用户名，所有机器都需要使用同一个用户名
password: 123456                  // 需要部署k8s集群的机器的ssh登录密码，所有机器都需要使用同一个密码
private-key-path: ~/.ssh/pri.key  // ssh免密登录的密钥，可以替代password防止密码泄露
masters:                          // 配置master节点的列表，建议每个master节点同时作为worker节点，否则master节点可以无法直接访问pod
- name: test0                     // 该节点的名称，为k8s集群看到的该节点的名称
  ip: 192.168.0.1                 // 该节点的ip地址
  port: 22                        // ssh登录的端口
  arch: arm64                     // 机器架构，x86_64的填amd64
workers:                          // 配置worker节点的列表
- name: test0                     // 该节点的名称，为k8s集群看到的该节点的名称
  ip: 192.168.0.1                 // 该节点的ip地址
  port: 22                        // ssh登录的端口
  arch: arm64                     // 机器架构，x86_64的填amd64
- name: test1
  ip: 192.168.0.3
  port: 22
  arch: arm64
etcds:                            // 配置etcd节点的列表，如果该项为空，则将会为每个master节点部署一个etcd，否则只会部署配置的etcd节点
- name: etcd-0                    // 该节点的名称，为k8s集群看到的该节点的名称
  ip: 192.168.0.4                 // 该节点的ip地址
  port: 22                        // ssh登录的端口
  arch: amd64                     // 机器架构，x86_64的填amd64
loadbalance:                      // 配置loadbalance节点
  name: k8s-loadbalance           // 该节点的名称，为k8s集群看到的该节点的名称
  ip: 192.168.0.5                 // 该节点的ip地址
  port: 22                        // ssh登录的端口
  arch: amd64                     // 机器架构，x86_64的填amd64
  bind-port: 8443                 // 负载均衡服务监听的端口 
external-ca: false                // 是否使用外部ca证书
external-ca-path: /opt/externalca // 外部ca证书文件的路径
service:                          // k8s创建的service的配置
  cidr: 10.32.0.0/16              // k8s创建的service的IP地址网段
  dnsaddr: 10.32.0.10             // k8s创建的service的DNS地址
  gateway: 10.32.0.1              // k8s创建的service的网关地址
  dns:                            // k8s创建的coredns的配置
    corednstype: pod              // k8s创建的coredns的部署类型，支持pod和binary
    imageversion: 1.8.4           // pod部署类型的coredns镜像版本
    replicas: 2                   // pod部署类型的coredns副本数量
network:                          // k8s集群网络配置
  podcidr: 10.244.0.0/16          // k8s集群网络的IP地址网段
  plugin: calico                  // k8s集群部署的网络插件
  plugin-args: {"NetworkYamlPath": "/etc/kubernetes/addons/calico.yaml"}   // k8s集群网络的网络插件的配置
apiserver-endpoint: 192.168.122.222:6443      // 对外暴露的APISERVER服务的地址或域名，如果配置了loadbalances则填loadbalance地址，否则填写第1个master节点地址
apiserver-cert-sans:                          // apiserver相关的证书中需要额外配置的ip和域名
  dnsnames: []                                // apiserver相关的证书中需要额外配置的域名列表
  ips: []                                     // apiserver相关的证书中需要额外配置的ip地址列表
apiserver-timeout: 120s                       // apiserver响应超时时间
etcd-external: false                          // 使用外部etcd，该功能还未实现
etcd-token: etcd-cluster                      // etcd集群名称
dns-vip: 10.32.0.10                           // dns的虚拟ip地址
dns-domain: cluster.local                     // DNS域名后缀
pause-image: k8s.gcr.io/pause:3.2             // 容器运行时的pause容器的容器镜像名称
network-plugin: cni                           // 网络插件类型
cni-bin-dir: /usr/libexec/cni,/opt/cni/bin    // 网络插件地址，使用","分隔多个地址
runtime: docker                               // 使用哪种容器运行时，目前支持docker和iSulad
runtime-endpoint: unix:///var/run/docker.sock // 容器运行时endpoint，docker可以不指定
registry-mirrors: []                          // 下载容器镜像时使用的镜像仓库的mirror站点地址
insecure-registries: []                       // 下载容器镜像时运行使用http协议下载镜像的镜像仓库地址
config-extra-args:                            // 各个组件(kube-apiserver/etcd等)服务启动配置的额外参数
  - name: kubelet                             // name支持："etcd","kube-apiserver","kube-controller-manager","kube-scheduler","kube-proxy","kubelet"
    extra-args:
      "--cgroup-driver": systemd              // 注意key对应的组件的参数，需要带上"-"或者"--"
open-ports:                                   // 配置需要额外打开的端口，k8s自身所需端口不需要进行配置，额外的插件的端口需要进行额外配置
  worker:                                     // 指定在那种类型的节点上打开端口，可以是master/worker/etcd/loadbalance
  - port: 111                                 // 端口地址
    protocol: tcp                             // 端口类型，tcp/udp
  - port: 179
    protocol: tcp
install:                                      // 配置各种类型节点上需要安装的安装包或者二进制文件的详细信息，注意将对应文件放到在tar.gz安装包中
  package-source:                                // 配置安装包的详细信息
    type: tar.gz                              // 安装包的压缩类型，目前只支持tar.gz类型的安装包
    dstpath: ""                               // 安装包在对端机器上的路径，必须是合法绝对路径
    srcpath:                                  // 不同架构安装包的存放路径，架构必须与机器架构相对应，必须是合法绝对路径
      arm64: /root/rpms/packages-arm64.tar.gz // arm64架构安装包的路径，配置的机器中存在arm64机器场景下需要配置，必须是合法绝对路径
      amd64: /root/rpms/packages-x86.tar.gz   // amd64类型安装包的路径，配置的机器中存在amd64机器场景下需要配置，必须是合法绝对路径                                 
  etcd:                                       // etcd类型节点需要安装的包或二进制文件列表
  - name: etcd                                // 需要安装的包或二进制文件的名称，如果是安装包则只写名称，不填写具体的版本号，安装时会使用`$name*`来识别
    type: pkg                                 // package的类型，pkg/repo/bin/file/dir/image/yaml七种类型，如果配置为repo请在对应节点上配置好repo源
    dst: ""                                   // 目的文件夹路径，bin/file/dir类型下需要配置，表示将文件(夹)放到节点的哪个目录下，为了防止用户误配置路径，导致cleanup时删除重要文件，此配置必须符合白名单，参见下一小节
  kubernetes-master:                          // k8s master类型节点需要安装的包或二进制文件列表
  - name: kubernetes-client,kubernetes-master
    type: pkg
  kubernetes-worker:                          // k8s worker类型节点需要安装的包或二进制文件列表
  - name: docker-engine,kubernetes-client,kubernetes-node,kubernetes-kubelet
    type: pkg
    dst: ""
  - name: conntrack-tools,socat
    type: pkg
    dst: ""
  network:                                    // 网络需要安装的包或二进制文件列表
  - name: containernetworking-plugins
    type: pkg
    dst: ""
  loadbalance:                                // loadbalance类型节点需要安装的包或二进制文件列表
  - name: gd,gperftools-libs,libunwind,libwebp,libxslt
    type: pkg
    dst: ""
  - name: nginx,nginx-all-modules,nginx-filesystem,nginx-mod-http-image-filter,nginx-mod-http-perl,nginx-mod-http-xslt-filter,nginx-mod-mail,nginx-mod-stream
    type: pkg
    dst: ""
  container:                                  // 容器需要安装的包或二进制文件列表
  - name: emacs-filesystem,gflags,gpm-libs,re2,rsync,vim-filesystem,vim-common,vim-enhanced,zlib-devel
    type: pkg
    dst: ""
  - name: libwebsockets,protobuf,protobuf-devel,grpc,libcgroup
    type: pkg
    dst: ""
  - name: yajl,lxc,lxc-libs,lcr,clibcni,iSulad
    type: pkg
    dst: ""
  image:                                      // 容器镜像tar包
  - name: pause.tar
    type: image
    dst: ""
  dns:                                        // k8s coredns安装包。如果corednstype配置为pod，此处无需配置
  - name: coredns
    type: pkg
    dst: ""
  addition:                                   // 额外的安装包或二进制文件列表
    master:
    - name: prejoin.sh
      type: shell                             // shell脚本
      schedule: "prejoin"                     // 执行时间master节点加入集群前
      TimeOut:  "30s"                         // 脚本执行时间，超时则被杀死，未配置默认30s
    - name: calico.yaml
      type: yaml
      dst: ""
    worker:
    - name: docker.service
      type: file
      dst: /usr/lib/systemd/system/
    - name: postjoin.sh
      type: shell                             // shell脚本
      schedule: "postjoin"                    // 执行时间worker节点加入集群后
```


### dst 白名单
dst可以配置为白名单中的目录，或者其子目录
```
"/usr/bin", "/usr/local/bin", "/opt/cni/bin", "/usr/libexec/cni",
"/etc/kubernetes",
"/usr/lib/systemd/system", "/etc/systemd/system",
"/tmp",
```