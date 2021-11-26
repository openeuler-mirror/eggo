# eggo支持的hooks设计

为了提供更好的灵活性，eggo支持多种hooks，主要分为如下几个场景：

- 集群生命周期管理的过程中针对特定角色的每个节点的hooks，支持prehook和posthook两种；
    - 生命周期管理包括：集群创建、集群删除、节点加入、节点删除；
    - 节点角色包括：master、worker、etcd和loadbalance；
- 针对整个集群的hooks，集群创建和删除的过程中只会在一个master节点上执行一次，支持prehook和posthook两种；

## 配置hook方式

### 命令行参数方式

| 参数                          | 支持的命令                  | 说明                                                         |
| ----------------------------- | --------------------------- | ------------------------------------------------------------ |
| --cluster-prehook=[dir/file]  | deploy, cleanup             | 设置集群创建/删除之前执行的hooks，可以是一个脚本文件或者目录 |
| --cluster-posthook=[dir/file] | deploy, cleanup             | 设置集群创建/删除之后执行的hooks，可以是一个脚本文件或者目录 |
| --prehook=[dir/file],role     | deploy, cleanup,join,delete | 集群创建/删除，节点加入/删除之前执行的hooks，可以是脚本文件或者目录；role设置执行脚本的节点角色； |
| --posthook=[dir/file],role    | deploy, cleanup,join,delete | 集群创建/删除，节点加入/删除之后执行的hooks，可以是脚本文件或者目录；role设置执行脚本的节点角色； |

说明：

- 脚本目录下的所有脚本都会被执行，而子目录中的脚本不会被执行；
- 每个脚本的超时时间为60s；
- role可以为master,worker,etcd或者loadbalance；

### 配置文件参数方式

在集群配置的addition字段中，可以设置shell类型的文件：

- 支持prejoin、postjoin、precleanup和postcleanup等执行时机；
- 而且通过在master或者worker角色下配置，设置hook执行的节点类型；

示例如下：

```
  addition:                                   // 额外的安装包或二进制文件列表
    master:
    - name: prejoin.sh
      type: shell                             // shell脚本
      schedule: "prejoin"                     // 执行时间master节点加入集群前
      TimeOut:  "30s"                         // 脚本执行时间，超时则被杀死，未配置默认30s
    worker:
    - name: postjoin.sh
      type: shell                             // shell脚本
      schedule: "postjoin"                    // 执行时间worker节点加入集群后
```

## hook规范

eggo会在hook执行时，通过环境变量传递部分信息，用于脚本执行。环境变量如下：

| key                       | value说明                                   |
| ------------------------- | ------------------------------------------- |
| EGGO_CLUSTER_ID           | 集群ID                                      |
| EGGO_CLUSTER_API_ENDPOINT | 集群的API入口                               |
| EGGO_CLUSTER_CONFIG_DIR   | 集群配置存放目录，默认/etc/kubernetes       |
| EGGO_NODE_IP              | hook执行的节点IP                            |
| EGGO_NODE_NAME            | hook执行的节点name                          |
| EGGO_NODE_ARCH            | hook执行的节点架构                          |
| EGGO_NODE_ROLE            | hook执行的节点角色                          |
| EGGO_HOOK_TYPE            | hook的类型，prehook或者posthook             |
| EGGO_OPERATOR             | 当前的操作，deploy，cleanup，join，delete。 |
