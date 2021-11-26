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

### resolv.conf缺失导致kubelet启动失败

#### 现象

kubelet启动不了，详细报错可以参考[issue](https://gitee.com/openeuler/eggo/issues/I457S5?from=project-issue)。

#### 解决方法

创建/etc/resolv.conf文件，并且设置合理的配置。

### 容器引擎下载pause镜像失败

报错信息：`SSL certificate problem: unable to get local issuer certificate`

#### 原因

由于机器没有安装相关证书，可以获取curl官网提供了ca证书。

#### 解决方法

```
$ wget https://curl.se/ca/cacert.pem
$ mv cacert.pem /etc/pki/ca-trust/source/anchors/
$ update-ca-trust
```

### 证书校验失败

在WaitNodeRegister阶段，报错Get "https://10.253.173.8:6443/api/v1/nodes/10.253.173.8": x509: certificate has expired or is not yet valid: current time 2021-11-24T17:12:19Z is before 2021-11-24T23:54:02Z 

#### 原因

机器时间未同步，生成证书的节点时间晚于worker节点的时间，导致证书校验失败

#### 解决方法

同步机器时间
