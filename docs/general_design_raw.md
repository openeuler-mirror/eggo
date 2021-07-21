# Eggo方案设计

## 概要方案

```mermaid
flowchart LR
	A(client) --> AA[[eggo]]
	subgraph steps
        BA[infrastructure]
        BA --> BB[etcd]
        BB --> BC[control plane]
        BC --> BD[join]
	end
	AA --> steps
	steps --> CA[binary implement]
	steps --> CB[pod implement]
	D(gitops) --> EA[CRD-1]
	F(cluster api) --> EB[CRD-2]
	EA & EB --> H[job]
	H --> AA
	classDef interfacestyle fill:#b1f,stroke:#f66,stroke-width:1px,color:#fff,stroke-dasharray:5 5
	class A,D,F interfacestyle
```

Eggo支持三种部署方式：

- 命令行模式：适用于物理机拉起小规模的K8S集群，用于测试开发使用；
- gitops模式：适用于通过gitops管理集群配置的场景，管理大量K8S集群的场景；
- cluster api模式：适用于对接cluster api开源接口；

### 部署组件

```mermaid
graph TD
	A[(configs)] --> B{{Eggo}}
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

Eggo组件交互关系图

```mermaid
sequenceDiagram
	participant A as client
	participant B as Eggo
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
	A[preflight check] --> B[copy dependences]
	B --> C[install dependences]
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
	O[preflight check] --> A[create encryption]
	A --> B[generate ca]
	B --> C[generate kubeconfigs]
	C --> D[save above files]
	D --> E[copy ca to master]
	E --> F[run services of master k8s]
	F --> G[apply admin role]
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
		D -->>+ B: return
		Note Left of B: update status of nodeA
		B -->>- A: add label to task
		and [push task to nodeB]
		A ->> C: push task
		C ->> E: use connection run command
		E -->>+ C: return
		Note Left of C: update status of nodeB
		C -->>- A: add label to task
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

#### 证书管理设计

集群的创建、新节点的加入都依赖证书；因此Eggo需要创建、存储和分发CA证书，并且协助节点创建其他依赖的证书；证书管理分为两种场景：

- 命令行管理集群的场景，需要创建并且本地存储ca证书和admin.kubeconfig，在集群部署master和node节点时分发相应的ca证书；
- 集群管理集群的场景，需要创建并且根据集群对应的存储ca证书和admin.kubeconfig到元集群的ETCD中，在集群部署master和node节点时分发相应的ca证书；

证书管理流程图如下：

```mermaid
flowchart TD
	A(eggo) --> BA[CMD模式]
	subgraph command
	BA --> BB{ca existed}
	BB -->|NO| BC(create ca)
	BC --> BD[save ca into disk]
	BB -->|YES| BE(read ca from disk)
	BD --> BE
	end
	A --> CA[集群管理模式]
	subgraph cluster
	CA --> CB{ca exist}
	CB -->|NO| CC(create ca)
	CC --> CD[save ca into etcd]
	CB -->|YES| CE(read ca from etcd)
	CD --> CE
	end
	BE --> E
	CE --> E
	subgraph master
	E{is master} -->|YES| F[copy ca]
	F --> FA[create required ceritifaces and kubeconfigs]
	FA --> FB[run k8s services]
	end
	E -->|NO| G[copy ca]
	subgraph worker
	G --> GA[get token]
	GA --> GB{is expired}
	GB -->|YES| GC[create new token]
	GB -->|NO| GD[return token]
	GC --> GD
	GD --> H[generate bootstrap kubeconfig]
	H --> I[join worker into cluster]
	end
```

时序关系如下：

```mermaid
sequenceDiagram
	participant eggo
	participant master
	participant worker
	eggo->eggo: exist ca?
	alt no
		eggo->>eggo: create and save ca
	end
	rect rgba(0,255,0,.1)
	eggo->>master: copy ca to master
	master->>master: create certificates and kubeconfigs
	master->>master: run services for k8s
	master-->>eggo: return
	end
	rect rgba(255,255,0,.1)
	eggo->>worker: copy ca to worker
	eggo->>master: request token
	master->>master: check token, is expired token?
	alt yes
		master->>master: create new token
	end
	master-->>eggo: return token
	eggo->>worker: send token to worker
	worker->>worker: create bootstrap kubeconfig
	worker->>worker: join to cluster
	worker-->>eggo: return
	end
```

