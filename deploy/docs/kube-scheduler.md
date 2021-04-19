# Kube-scheduler组件

控制平面组件，负责监视新创建的、未指定运行[节点（node）](https://kubernetes.io/zh/docs/concepts/architecture/nodes/)的 [Pods](https://kubernetes.io/docs/concepts/workloads/pods/pod-overview/)，选择节点让 Pod 在上面运行。

调度决策考虑的因素包括单个 Pod 和 Pod 集合的资源需求、硬件/软件/策略约束、亲和性和反亲和性规范、数据位置、工作负载间的干扰和最后时限。

## 参数配置

| 参数                        | 配置                           | 说明                                                         |
| --------------------------- | ------------------------------ | ------------------------------------------------------------ |
| --kubeconfig                | /etc/kubernetes/scheduler.conf | 已弃用: 包含鉴权和主节点位置信息的 kubeconfig 文件的路径。   |
| --authentication-kubeconfig | /etc/kubernetes/scheduler.conf | 指向具有足够权限以创建 `tokenaccessreviews.authentication.k8s.io` 的 Kubernetes 核心服务器的 kubeconfig 文件。 这是可选的。如果为空，则所有令牌请求均被视为匿名请求，并且不会在集群中查找任何客户端 CA。 |
| --authorization-kubeconfig  | /etc/kubernetes/scheduler.conf | 指向具有足够权限以创建 subjectaccessreviews.authorization.k8s.io 的 Kubernetes 核心服务器的 kubeconfig 文件。这是可选的。 如果为空，则所有未被鉴权机制略过的请求都会被禁止。 |
| --leader-elect              | true                           | 在执行主循环之前，开始领导者选举并选出领导者。 使用多副本来实现高可用性时，可启用此标志。 |

