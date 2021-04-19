####                                                                                       kube-proxy部署分析

kube-proxy是k8s的node节点上用于实现service的组件，所谓的service可以理解成多个pod的统一对外入口/出口，多个pod以一个service的方式对外提供服务。

kube-proxy常用参数：

| 配置参数         | 命令行参数           | 功能说明                                                     |
| ---------------- | -------------------- | ------------------------------------------------------------ |
|                  | --config             | kube-proxy的配置文件，yaml格式                               |
|                  | --hostname-override= | 节点在集群中的名称，该参数必须和kubelet中对应的参数配置成一样的 |
| clientConnection |                      | 连接apiserver相关配置。kubeconfig指定kube-proxy配置文件的路径，该配置是kube-proxy和apiserver通信时的密钥和证书文件的配置，在管理节点上通过命令"kubectl config set-cluster"/"kubectl config set-credentials"/"kubectl config set-context"/"kubectl config use-context"生成相关的配置。<br/><br/>clientConnection:<br/>    kubeconfig: /etc/kubernetes/kube-proxy.conf |
| clusterCIDR      |                      | 集群地址范围。这个地址指的是集群中pod的IP地址可用的网段，注意和service的CIDR做区别，两者地址网段配置不能重叠。 |
| mode             |                      | kube-proxy的代理模式。集群规模比较大时使用ipvs模式，小规模集群可以使用iptables模式。userspace模式已被废弃。 |

