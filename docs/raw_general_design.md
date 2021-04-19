# Eggo概要设计

## 整体方案

```mermaid
flowchart LR
	A[command client] --> tasksList
	subgraph tasksList
	BA[infrastructure task]
	BA --> BB[first control plane task]
	BB --> BC[worker or other control node task]
	BC -->|current| BD(worker1)
	BC -->|current| BE(worker...)
	BC -->|current| BF(workerN)
	end
	tasksList --> C[[eggo deploy module]]
	subgraph controllers
	DA[infrastructure]
	DB[control plane]
	DC[bootstrap]
	end
	controllers --> C
	gitops --> E(operator)
	E --> controllers
	F[cluster api] --> controllers
	
```

Eggo支持三种部署方式：

- 命令行模式：适用于物理机拉起小规模的K8S集群，用于测试开发使用；
- gitops模式：适用于通过gitops管理集群配置的场景，管理大量K8S集群的场景；
- cluster api模式：适用于对接cluster api开源接口；

## 部署组件

```mermaid
graph TD
	A[(configs)] --> B{{EggoDeploy}}
	B --> C(Infrastructure)
	B --> D(etcd cluster)
	B --> E(control plane)
	B --> F(bootstrap)
	C --- CA>负责待部署节点的基础设施准备]
	D --> DA(intreface)
	DA --> DB[pod方式]
	DA --> DC[二进制方式]
	DB & DC --> DD>负责etcd集群的独立部署]
	E --> EA(intreface)
	EA --> EB[pod方式]
	EA --> EC[二进制方式]
	EB & EC --> ED>负责第一个控制面的部署]
	F --> FA(interface)
	FA --> FB[pod方式]
	FA --> FC[二进制方式]
	FB & FC --> FD>负责worker或者其他控制面节点加入K8S集群的部署]
	style B fill:#fa3
	classDef interfacestyle fill:#b1f,stroke:#f66,stroke-width:1px,color:#fff,stroke-dasharray:5 5
	class DA,EA,FA interfacestyle
```

部署组件负责集群的实际部署工作，主要包括如下部分：

- 基础设施：负责集群节点（物理机、虚拟机等）的准备工作，系统安装、节点间网络互通部署、磁盘安装、依赖组件安装等；
- 独立ETCD集群部署：ETCD集群独立部署，不在控制面或者工作节点以保证数据的安全；
- 集群的首个控制面：部署第一个控制面，需要负责证书生成、kubeconfig生成以及组件服务部署等；
- bootstrap：负责worker节点或者其他控制面节点加入K8S集群的部署工作，证书书拷贝、组件服务部署；

