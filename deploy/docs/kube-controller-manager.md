# Kube-controller-manager组件

在主节点上运行 [控制器](https://kubernetes.io/zh/docs/concepts/architecture/controller/) 的组件。

从逻辑上讲，每个[控制器](https://kubernetes.io/zh/docs/concepts/architecture/controller/)都是一个单独的进程， 但是为了降低复杂性，它们都被编译到同一个可执行文件，并在一个进程中运行。

这些控制器包括:

- 节点控制器（Node Controller）: 负责在节点出现故障时进行通知和响应
- 任务控制器（Job controller）: 监测代表一次性任务的 Job 对象，然后创建 Pods 来运行这些任务直至完成
- 端点控制器（Endpoints Controller）: 填充端点(Endpoints)对象(即加入 Service 与 Pod)
- 服务帐户和令牌控制器（Service Account & Token Controllers）: 为新的命名空间创建默认帐户和 API 访问令牌

## 参数配置

| 参数                               | 配置                                    | 说明                                                         |
| ---------------------------------- | --------------------------------------- | ------------------------------------------------------------ |
| --bind-address                     | 0.0.0.0                                 | 默认值：0.0.0.0；针对 --secure-port 端口上请求执行监听操作的 IP 地址。 所对应的网络接口必须从集群中其它位置可访问（含命令行及 Web 客户端）。 如果此值为空或者设定为非特定地址（0.0.0.0 或 ::），意味着所有网络接口都在监听范围。 |
| --cluster-cidr                     | 10.244.0.0/16                           | 集群中 Pods 的 CIDR 范围，如果配置CNI网络时建议保持一致。要求 --allocate-node-cidrs 标志为 true。 |
| --cluster-name                     | kubernetes                              | 默认值："kubernetes"，集群实例的前缀。                       |
| --cluster-signing-cert-file        | /etc/kubernetes/pki/ca.crt              | 默认值："/etc/kubernetes/ca/ca.pem"，包含 PEM 编码格式的 X509 CA 证书的文件名。该证书用来发放集群范围的证书。 如果设置了此标志，则不需要锦衣设置 `--cluster-signing-*` 标志。 |
| --cluster-signing-key-file         | /etc/kubernetes/pki/ca.key              | 默认值："/etc/kubernetes/ca/ca.key"，包含 PEM 编码的 RSA 或 ECDSA 私钥的文件名。该私钥用来对集群范围证书签名。 |
| --kubeconfig                       | /etc/kubernetes/controller-manager.conf | 指向 kubeconfig 文件的路径。该文件中包含主控节点位置以及鉴权凭据信息。 |
| --leader-elect                     | true                                    | 在执行主循环之前，启动领导选举（Leader Election）客户端，并尝试获得领导者身份。 在运行多副本组件时启用此标志有助于提高可用性。 |
| --root-ca-file                     | /etc/kubernetes/pki/ca.crt              | 如果此标志非空，则在服务账号的令牌 Secret 中会包含此根证书机构。 所指定标志值必须是一个合法的 PEM 编码的 CA 证书包。 |
| --service-account-private-key-file | /etc/kubernetes/pki/sa.key              | 包含 PEM 编码的 RSA 或 ECDSA 私钥数据的文件名，这些私钥用来对服务账号令牌签名 |
| --service-cluster-ip-range         | 10.32.0.0/16                            | 集群中 Service 对象的 CIDR 范围。要求 `--allocate-node-cidrs` 标志为 true。 |
| --use-service-account-credentials  | true                                    | 当此标志为 true 时，为每个控制器单独使用服务账号凭据。       |
| --authentication-kubeconfig        | /etc/kubernetes/controller-manager.conf | 此标志值为一个 kubeconfig 文件的路径名。该文件中包含与某 Kubernetes “核心” 服务器相关的信息，并支持足够的权限以创建 tokenreviews.authentication.k8s.io。 此选项是可选的。如果设置为空值，所有令牌请求都会被认作匿名请求， Kubernetes 也不再在集群中查找客户端的 CA 证书信息。 |
| --authorization-kubeconfig         | /etc/kubernetes/controller-manager.conf | 包含 Kubernetes “核心” 服务器信息的 kubeconfig 文件路径， 所包含信息具有创建 subjectaccessreviews.authorization.k8s.io 的足够权限。 此参数是可选的。如果配置为空字符串，未被鉴权模块所忽略的请求都会被禁止。 |
| --requestheader-client-ca-file     | /etc/kubernetes/pki/front-proxy-ca.crt  | 根证书包文件名。在信任通过 `--requestheader-username-headers` 所指定的任何用户名之前，要使用这里的证书来检查请求中的客户证书。 警告：一般不要依赖对请求所作的鉴权结果。 |



