# Kube-APIServer组件

API 服务器是 Kubernetes [控制面](https://kubernetes.io/zh/docs/reference/glossary/?all=true#term-control-plane)的组件， 该组件公开了 Kubernetes API。 API 服务器是 Kubernetes 控制面的前端。

Kubernetes API 服务器的主要实现是 [kube-apiserver](https://kubernetes.io/zh/docs/reference/command-line-tools-reference/kube-apiserver/)。 kube-apiserver 设计上考虑了水平伸缩，也就是说，它可通过部署多个实例进行伸缩。 你可以运行 kube-apiserver 的多个实例，并在这些实例之间平衡流量（用于实现集群的高可用）。

## 参数配置

| 参数                                 | 配置                                                         | 说明                                                         |
| ------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| --advertise-address                  | 192.168.1.1                                                  | 向集群成员通知 apiserver 消息的 IP 地址。 这个地址必须能够被集群中其他成员访问。 如果 IP 地址为空，将会使用 --bind-address， 如果未指定 --bind-address，将会使用主机的默认接口地址。 |
| --allow-privileged                   | true                                                         | 如果为 true, 将允许特权容器。[默认值=false]                  |
| --authorization-mode                 | Node,RBAC                                                    | 在安全端口上进行鉴权的插件的顺序列表，默认值：[AlwaysAllow]。 逗号分隔的列表：AlwaysAllow,AlwaysDeny,ABAC,Webhook,RBAC,Node。 |
| --enable-admission-plugins           | NamespaceLifecycle,NodeRestriction,LimitRanger, ServiceAccount,DefaultStorageClass,ResourceQuota | 除了默认启用的插件之外要启用的插件<br/>该标志中插件的顺序无关紧要。支持的插件列表见官方文档 |
| --secure-port                        | 6443                                                         | 默认值：6443，带身份验证和鉴权机制的 HTTPS 服务端口。 不能用 0 关闭。 |
| --enable-bootstrap-token-auth        | true                                                         | 启用以允许将 "kube-system" 名字空间中类型为 "bootstrap.kubernetes.io/token" 的 Secret 用于 TLS 引导身份验证。 |
| --etcd-cafile                        | /etc/kubernetes/pki/etcd/ca.crt                              | 用于保护 etcd 通信的 SSL 证书颁发机构文件。                  |
| --etcd-certfile                      | /etc/kubernetes/pki/apiserver-etcd-client.crt                | 用于保护 etcd 通信的 SSL 证书文件。                          |
| --etcd-keyfile                       | /etc/kubernetes/pki/apiserver-etcd-client.key                | 用于保护 etcd 通信的 SSL 密钥文件。                          |
| --etcd-servers                       | https://192.168.1.1:2379,https://192.168.1.2:2379            | 要连接的 etcd 服务器列表（scheme://ip:port），以逗号分隔。   |
| --client-ca-file                     | /etc/kubernetes/pki/ca.crt                                   | 如果已设置，则使用与客户端证书的 CommonName 对应的标识对任何出示由 client-ca 文件中的授权机构之一签名的客户端证书的请求进行身份验证。 |
| --kubelet-client-certificate         | /etc/kubernetes/pki/apiserver-kubelet-client.crt             | 用于api-server发给kubelet通信的TLS 的客户端证书文件的路径。  |
| --kubelet-client-key                 | /etc/kubernetes/pki/apiserver-kubelet-client.key             | 用于api-server发给kubelet通信的TLS 客户端密钥文件的路径。    |
| --proxy-client-cert-file             | /etc/kubernetes/pki/front-proxy-client.crt                   | 当必须调用外部程序以处理请求时，用于证明聚合器或者 kube-apiserver 的身份的客户端证书。 包括代理转发到用户 api-server 的请求和调用 Webhook 准入控制插件的请求。 Kubernetes 期望此证书包含来自于 --requestheader-client-ca-file 标志中所给 CA 的签名。 该 CA 在 kube-system 命名空间的 "extension-apiserver-authentication" ConfigMap 中公开。 从 kube-aggregator 收到调用的组件应该使用该 CA 进行各自的双向 TLS 验证。 |
| --proxy-client-key-file              | /etc/kubernetes/pki/front-proxy-client.key                   | 当必须调用外部程序来处理请求时，用来证明聚合器或者 kube-apiserver 的身份的客户端私钥。 这包括代理转发给用户 api-server 的请求和调用 Webhook 准入控制插件的请求。 |
| --tls-cert-file                      | /etc/kubernetes/pki/apiserver.crt                            | 包含用于 HTTPS 的默认 x509 证书的文件。（CA 证书（如果有）在服务器证书之后并置）。 如果启用了 HTTPS 服务，并且未提供 --tls-cert-file 和 --tls-private-key-file， 为公共地址生成一个自签名证书和密钥，并将其保存到 --cert-dir 指定的目录中。 |
| --tls-private-key-file               | /etc/kubernetes/pki/apiserver.key                            | 包含匹配 --tls-cert-file 的 x509 证书私钥的文件。            |
| --service-cluster-ip-range           | 10.32.0.0/16                                                 | CIDR 表示的 IP 范围用来为服务分配集群 IP。 此地址不得与指定给节点或 Pod 的任何 IP 范围重叠。 |
| --service-account-issuer             | https://kubernetes.default.svc.cluster.local                 | 服务帐号令牌颁发者的标识符。 颁发者将在已办法令牌的 "iss" 声明中检查此标识符。 此值为字符串或 URI。 如果根据 OpenID Discovery 1.0 规范检查此选项不是有效的 URI，则即使特性门控设置为 true， ServiceAccountIssuerDiscovery 功能也将保持禁用状态。 强烈建议该值符合 OpenID 规范：https://openid.net/specs/openid-connect-discovery-1_0.html。 实践中，这意味着 service-account-issuer 取值必须是 HTTPS URL。 还强烈建议此 URL 能够在 {service-account-issuer}/.well-known/openid-configuration 处提供 OpenID 发现文档。 |
| --service-account-key-file           | /etc/kubernetes/pki/sa.pub                                   | 包含 PEM 编码的 x509 RSA 或 ECDSA 私钥或公钥的文件，用于验证 ServiceAccount 令牌。 指定的文件可以包含多个键，并且可以使用不同的文件多次指定标志。 如果未指定，则使用 --tls-private-key-file。 提供 --service-account-signing-key 时必须指定。 |
| --service-account-signing-key-file   | /etc/kubernetes/pki/sa.key                                   | 包含服务帐户令牌颁发者当前私钥的文件的路径。 颁发者将使用此私钥签署所颁发的 ID 令牌。 |
| --service-node-port-range            | 30000-32767                                                  | 保留给具有 NodePort 可见性的服务的端口范围。 例如："30000-32767"。范围的两端都包括在内。 |
| --requestheader-allowed-names        | front-proxy-client                                           | 此值为客户端证书通用名称（Common Name）的列表；表中所列的表项可以用来提供用户名， 方式是使用 --requestheader-username-headers 所指定的头部。 如果为空，能够通过 --requestheader-client-ca-file 中机构认证的客户端证书都是被允许的。 |
| --requestheader-client-ca-file       | /etc/kubernetes/pki/front-proxy-ca.crt                       | 在信任请求头中以 `--requestheader-username-headers` 指示的用户名之前， 用于验证接入请求中客户端证书的根证书包。 警告：一般不要假定传入请求已被授权。 |
| --requestheader-extra-headers-prefix | X-Remote-Extra-                                              | 用于查验请求头部的前缀列表。建议使用 X-Remote-Extra-。       |
| --requestheader-group-headers        | X-Remote-Group                                               | 用于查验用户组的请求头部列表。建议使用 X-Remote-Group。      |
| --requestheader-username-headers     | X-Remote-User                                                | 用于查验用户名的请求头头列表。建议使用 X-Remote-User。       |
| --encryption-provider-config         | /etc/kubernetes/encryption-config.yaml                       | 包含加密提供程序配置信息的文件，用在 etcd 中所存储的 Secret 上。 |

### encryption-config生成方法

```bash
ENCRYPTION_KEY=$(head -c 32 /dev/urandom | base64)
cat >$result_dir/encryption-config.yaml <<EOF
kind: EncryptionConfig
apiVersion: v1
resources:
  - resources:
      - secrets
    providers:
      - aescbc:
          keys:
            - name: key1
              secret: ${ENCRYPTION_KEY}
      - identity: {}
EOF
```

