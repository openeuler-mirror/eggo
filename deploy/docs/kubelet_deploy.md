###                                                                             kubelet部署分析

kubelet运行在k8s集群的node节点上，主要用于处理Master节点下发的任务，用于运行和管理pod。

kubelet的配置可以是yaml形式的配置文件，也可以在kubelet启动命令行加上配置参数，命令行的参数和配置中的参数有冲突时对应的参数采用命令行的参数，配置文件中的参数则不生效。

常用参数如下：

| 配置参数              | 命令行参数                  | 含义                                                         |
| --------------------- | --------------------------- | ------------------------------------------------------------ |
|                       | --address                   | kubelet监听地址，默认监听0.0.0.0                             |
|                       | --config                    | 指定kubelet配置文件的路径，yaml格式                          |
|                       | --kubeconfig                | 指定kubelet配置文件的路径，该配置是kubelet和apiserver通信时的密钥和证书文件的配置，在管理节点上通过命令"kubectl config set-cluster"/"kubectl config set-credentials"/"kubectl config set-context"/"kubectl config use-context"生成相关的配置。 |
| authentication        |                             | 身份验证方式。<br/>authentication: 是否允许匿名用户访问<br/>webhook:是否允许webhook访问<br/>x509:x509方式认证的ca证书地址<br/><br/>示例:<br/>authentication:<br/>        anonymous:<br/>            enabled: false<br/>        webhook:<br/>            enabled: true<br/>        x509:<br/>            clientCAFile: /etc/kubernetes/pki/ca.crt<br/> |
| authorization         |                             | 服务授权模式。<br/><br/>authorization:<br/>   mode: Webhook  |
| clusterDNS            |                             | 集群的DNS地址，可以配置多个                                  |
| clusterDomain         |                             | 集群内使用的域名，容器内进行DNS查询时除了搜索主机配置的域还会搜索这个配置的集群域 |
| runtimeRequestTimeout |                             | 和容器运行时交互的超时时间                                   |
|                       | --network-plugin            | 指定使用什么类型的网络插件                                   |
|                       | --pod-infra-container-image | 指定pause容器的镜像名称                                      |
|                       | --register-node             | 配置为true表示采用kubelet向apiserver注册的方式连接apiserver。kubelet和apiserver有多个注册方式，一般都采用kubelet向apiserver注册的方式。 |
|                       | --hostname-override         | 节点在集群中的名称，如果配置该参数这则kube-proxy也要配置成一样的 |
|                       | --cni-bin-dir               | 存放CNI插件的二进制文件的路径                                |
|                       | --container-runtime         | 指定使用哪种运行时容器。可以配置成"docker"或者"rkt"。如果使用CRI对接，则配置成remote。 |
|                       | --container-runtime         | 容器运行时的地址。使用isulad时可以配置成unix:///var/run/isulad.sock |
|                       | -v                          | 日志级别。范围0-8。推荐使用2。4表示debug。数字越大信息越详细。 |

