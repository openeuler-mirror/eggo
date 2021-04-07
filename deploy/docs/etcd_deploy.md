#                                             etcd部署分析

etcd是一种开源的分布式键值存储，用于分布式系统或者计算机集群的共享配置/服务发现和调度协调。

etcd本身支持集群模式部署。根据k8s文档中的描述，建议在生产环境中使用5个etcd实例组成的集群。

etcd可以通过静态部署或者动态部署(又包括etcd动态发现和DNS动态发现)，一般采用静态部署。

arm环境需要配置环境变量*ETCD_UNSUPPORTED_ARCH=arm64*



常用的配置参数：

| 选项                                                         | 命令行                        | 含义                                                         |
| ------------------------------------------------------------ | ----------------------------- | ------------------------------------------------------------ |
| ETCD_ADVERTISE_CLIENT_URLS="https://192.168.122.43:2379"     | --advertise-client-urls       | 广播给其它etcd实例的当前etcd实例的HTTP/HTTPS API地址。实际提供HTTP/HTTPS API接口的地址可能会有多个，但是仅这个参数配置的地址会广播给其它etcd实例。配置的地址必须在--listen-client-urls中并且地址对于其他etcd实例可达。 |
| ETCD_LISTEN_CLIENT_URLS="https://127.0.0.1:2379,https://192.168.122.43:2379" | --listen-client-urls          | 接收HTTP/HTTPS API请求的地址。                               |
| ETCD_CERT_FILE="/etc/kubernetes/pki/etcd/server.crt"         | --cert-file                   | 该etcd实例用于HTTP/HTTPS API服务的TLS证书文件的路径          |
| ETCD_CLIENT_CERT_AUTH=false                                  | --client-cert-auth            | 该etcd实例用于HTTP/HTTPS API服务时是否验证客户端证书         |
| ETCD_DATA_DIR="/var/lib/etcd/default.etcd"                   | --data-dir                    | etcd数据的存放目录                                           |
| ETCD_INITIAL_ADVERTISE_PEER_URLS="https://192.168.122.43:2380" | --initial-advertise-peer-urls | 广播给其它etcd实例的用于和当前etcd实例通信的地址。注意该地址和提供API服务的地址不同，该地址是用于和其他etcd实例通信的，不是用于提供API服务的。实际提供通信接口的地址可能会有多个，但是仅这个参数配置的地址会广播给其它etcd实例。配置的地址必须在--listen-peer-urls中并且地址对于其他etcd实例可达。 |
| ETCD_LISTEN_PEER_URLS="https://192.168.122.43:2380"          | --listen-peer-urls            | 和其它实例通信的地址。注意这个是etcd集群的各个etcd间通信的地址，并不是提供API接口的地址。 |
| ETCD_INITIAL_CLUSTER="localhost.localdomain=https://192.168.122.43:2380" | --initial-cluster             | 集群中所有etcd通信端口的列表。key=value格式，key是etcd在集群中的名称(即--name对应的值)，value是名称对应的通信地址，多个之间使用","分隔。 |
| ETCD_KEY_FILE="/etc/kubernetes/pki/etcd/server.key"          | --key-file                    | 该etcd实例用于HTTP/HTTPS API服务的TLS密钥文件的路径          |
| ETCD_LISTEN_METRICS_URLS="https://192.168.122.43:2381"       | --listen-metrics-urls         | 提供额外的URL地址用于响应/metrics和/health查询               |
| ETCD_NAME="localhost.localdomain"                            | --name                        | etcd实例在集群中的名称                                       |
| ETCD_PEER_CERT_FILE="/etc/kubernetes/pki/etcd/peer.crt"      | --peer-cert-file              | 该etcd实例用于和其它etcd实例通信的TLS证书文件的路径          |
| ETCD_PEER_CLIENT_CERT_AUTH=false                             | --peer-client-cert-auth       | 该etcd实例用于和其它etcd实例通信时是否验证对端的证书         |
| ETCD_PEER_KEY_FILE="/etc/kubernetes/pki/etcd/peer.key"       | --peer-key-file               | 该etcd实例用于和其它etcd实例通信的TLS密钥文件的路径          |
| ETCD_TRUSTED_CA_FILE="/etc/kubernetes/pki/etcd/ca.crt"       | --trusted-ca-file             | 处理HTTP/HTTPS API请求时使用的CA文件路径                     |
| ETCD_PEER_TRUSTED_CA_FILE="/etc/kubernetes/pki/etcd/ca.crt"  | --peer-trusted-ca-file        | 和其它etcd实例通信时使用的CA文件路径                         |
| ETCD_SNAPSHOT_COUNT=10000                                    | --snapshot-count              | 触发数据快照的提交事务数量。etcd处理指定的次数的事务提交后，触发数据快照。 |

