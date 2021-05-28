# eggo集群清理流程

eggo集群清理时需要使用集群部署时的配置，以确保能准确完整地进行集群清理。清理过程会先删除calico插件等系统资源，删除workers节点，删除etcd集群。集群清理过程不会删除容器引擎以及iptables，需要用户自行清理。下图是eggo集群清理的流程：

~~~mermaid
sequenceDiagram
    participant eggo
    participant masters
    participant workers
    participant etcds
    participant loadbalances
    participant coredns
    eggo ->> masters : delete system resources
    eggo ->> masters : remove workers from cluster
    eggo ->> etcds : remove etcd members
    eggo ->> workers : stop service，uninstall pkgs，restore firewall，remove files and dirs
    eggo ->> masters : stop service，uninstall pkgs，restore firewall，remove files and dirs
    eggo ->> etcds : stop service，uninstall pkgs，restore firewall，remove files and dirs
    eggo ->> loadbalances : stop service，uninstall pkgs，restore firewall，remove files and dirs
    eggo ->> coredns : stop service，uninstall pkgs，restore firewall，remove files and dirs
~~~

1. 使用第1个master节点删除calico插件等系统资源
2. 使用第1个master节点删除所有的workers节点
3. 使用第1个etcd节点删除除了该节点外的所有etcd members节点
4. 登录到所有节点进行清理操作，所有安装有master/worker/etcd/loadbalance/coredns的任何一个的节点都会登录上去进行清理操作，清理操作主要是调用infrastructure提供的接口进行清理，并删除一些在部署过程中进行过改动的配置/service文件/证书文件等。