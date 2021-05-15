# Eggo方案设计

## 概要方案

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
	tasksList --> C[[EggoDeploy module]]
	subgraph manager-cluster
	DA[infrastructure controller]
	DB[control-plane controller]
	DC[bootstrap controller]
	end
	manager-cluster --> C
	gitops --> E(operator)
	E --> manager-cluster
	F[cluster api] --> manager-cluster
	
```

Eggo支持三种部署方式：

- 命令行模式：适用于物理机拉起小规模的K8S集群，用于测试开发使用；
- gitops模式：适用于通过gitops管理集群配置的场景，管理大量K8S集群的场景；
- cluster api模式：适用于对接cluster api开源接口；

### 部署组件

```mermaid
graph TD
	A[(configs)] --> B{{EggoDeploy}}
	B --> C(intreface)
	C --> D[pod方式]
	C --> E[二进制方式]
	D --> DA(Infrastructure)
	D --> DB(etcd cluster)
	D --> DC(control plane)
	D --> DD(bootstrap)
	D --> DE(upgrade/clean)
	E --> EA(Infrastructure)
	E --> EB(etcd cluster)
	E --> EC(control plane)
	E --> ED(bootstrap)
	E --> EE(upgrade/clean)
	style B fill:#fa3
	classDef interfacestyle fill:#b1f,stroke:#f66,stroke-width:1px,color:#fff,stroke-dasharray:5 5
	class C interfacestyle
```

部署组件负责集群的实际部署工作，主要包括如下部分：

- 基础设施：负责集群节点（物理机、虚拟机等）的准备工作，系统安装、节点间网络互通部署、磁盘安装、依赖组件安装等；
- 独立ETCD集群部署：ETCD集群独立部署，不在控制面或者工作节点以保证数据的安全；
- 集群的首个控制面：部署第一个控制面，需要负责证书生成、kubeconfig生成以及组件服务部署等；
- bootstrap：负责worker节点或者其他控制面节点加入K8S集群的部署工作，证书书拷贝、组件服务部署；
- upgrade/clean：集群组件升级或者清理；

## 详细设计

### 部署组件

EggoDeploy组件交互关系图

```mermaid
sequenceDiagram
	participant A as client
	participant B as EggoDeploy
	participant C as kubeadm
	participant D as binaryadm
	A ->>+ B: call (infra/etcd/control/bootstrap..)
	B ->> B: parse config
	alt driver is kubeadm
		B ->> C: do implements of interfaces
		C -->> B: response
	else driver is binaray
		B ->> D: do implements of interfaces
		D -->> B: response
	end
	B -->>- A: response
```

#### infrastructure流程

```mermaid
graph TD
	A[preflight check] --> B[download dependences]
	B --> C[install dependences]
	C --> D[config environment]
```

#### etcd集群流程

```mermaid
graph TD
	A[preflight check] --> B[download dependences/certs]
	B --> C[install dependences]
	C --> D[run etcd services]
	D --> E[check etcd cluster]
```

#### control plane流程

```mermaid
graph TD
	O[preflight check] --> A[download dependces]
	A --> B[install dependces]
	B --> C[generate certs]
	C --> D[generate kubeconfigs]
	D --> E[upload certs and kubeconfigs]
	E --> F[run services of K8S]
	F --> G[apply addons]
```

#### bootstrap流程

```mermaid
graph TD
	A[preflight check] --> B[download dependence/certs]
	B --> C[install dependences]
	C --> D[run services of k8s]
```

#### clean流程

```mermaid
graph TD
	A[preflight check] --> B[quit from cluster]
	B --> C[quit from etcd cluster]
	C --> D[stop services]
	D --> E[remove install modules]
	E --> F[remove dependences/certs and so on]
```

#### node-task任务管理机制

旨在统一管理node上执行task，为节点部署集群，提供统一的命令和文件拷贝接口。

模块时序如下：

```mermaid
sequenceDiagram
	participant A as node-manager
	participant B as nodeA
	participant D as 节点A
	participant C as nodeB
	participant E as 节点B
	rect rgba(0, 255, 0, .1)
        A ->> B: register nodeA
        B ->> D: connect节点
        D --> B: return connection
        B ->> B: goroutine wait task
        A ->> C: register nodeB
        C ->> E: connect节点
        E --> C: return connection
        C ->> C: goroutine wait task
    end
	rect rgba(255, 0, 255, .1)
		par [push task to nodeA]
		A ->> B: push task
		B ->> D: use connection run command
		D -->> B: return
		B -->> A: add label to task
		and [push task to nodeB]
		A ->> C: push task
		C ->> E: use connection run command
		E -->> C: return
		C -->> A: add label to task
		end
		A -->> A: wait task on nodes success
	end
    
```

类和接口关系如下：

```mermaid
classDiagram
	class NodeManager
	NodeManager : -nodes map
	NodeManager : -lock RWMutex
	NodeManager : +RegisterNode()
	NodeManager : +UnRegisterNode()
	NodeManager : +UnRegisterAllNodes()
	NodeManager : +WaitTaskOnNodesFinished()
	NodeManager : +WaitTaskOnAllFinished()
	NodeManager : +RunTaskOnNodes()
	NodeManager : +RunTaskOnAll()
	class Node
	Node : -host HostConfig
	Node : -r Runner
	Node : -stop chan
	Node : -queue chan
	Node : +PushTask()
	Node : +Finish()
	class Task{
		<<interface>>
		Name() string
		Run(runner.Runner) error
		AddLabels(key, lable string)
		GetLable(key string) string
	}
	class Runner{
		<<interface>>
		Copy(src, dst string) error
		RunCommand(cmd string) error
		Close()
	}
	class SSHRunner{
		Host *kkv1alpha1.HostCfg
		Conn ssh.Connection
	}
	Runner <|-- SSHRunner : implements
	NodeManager "1" -- "n" Node: contains
	Task "1" -- "1" Runner: call
	Node "n" .. "n" Task: bind
	
	
```

