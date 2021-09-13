# eggo configuration

The config directory stores multiple cluster config templates, and help users configure their own clusters.

## config templates

| name | OS | arch | masterNum | workerNum | loadbalance | Runtime | Install |
| --- | --- | --- | --- | --- | --- | --- | --- |
| centos.config | CentOS7 | amd64 | 1 | 2 | No | docker | bin |
| openEuler.config | openEuler 21.03 | arm64 | 1 | 2 | No | iSulad | repo + rpm |

## Instructions

### Modify config
Users select the appropriate config template and make modifications. The following modifications will be considered:
- ssh login username/password
- domain name, ip, architecture name/ip/arch of each node
- apiserver-endpoint, if you set the loadbalance, it is the IP of `loadbalance: bind-port`, else it is the IP of the first `master: 6443`
- install software `packages` and compressed packages `packages-src`

For other detailed configuration of config, please refer to the eggo operation manual

### Prepare compressed packages

Users prepare the offline installation package according to the packages configuration. Take `openEuler.config `as an example, the compressed package include:

```
$ tree /root/packages
/root/packages
├── file
│   └── calico.yaml
├── image
│   └── images.tar
└── pkg
    └── coredns-1.7.0-1.0.oe1.aarch64.rpm

3 directories, 3 files
```

The images.tar is the image used in the cluster deployment, including:
```
REPOSITORY                  TAG       IMAGE ID       CREATED         SIZE
calico/node                 v3.19.1   c4d75af7e098   5 weeks ago     168MB
calico/pod2daemon-flexvol   v3.19.1   5660150975fb   5 weeks ago     21.7MB
calico/cni                  v3.19.1   5749e8b276f9   5 weeks ago     146MB
calico/kube-controllers     v3.19.1   5d3d5ddc8605   5 weeks ago     60.6MB
k8s.gcr.io/pause            3.2       80d28bedfe5d   16 months ago   683kB
```
Users can also not prepare images.tar, and the container engine will pull the image.

Package compression
```
$ cd /root/packages
$ tar -zcvf packages-arm.tar.gz ./*
```
If there are X86 architecture nodes, you need to prepare a package of X86 architecture software

### Prepare notes documentation (optional)

In addition to the config file and the compressed package, a description file is also needed. It records the source of the files in the compressed package, and make them traceability. Taking `centos.config` as an example, the corresponding notes file is as follows:

```bash
1.  ETCD
    - etcd,etcdctl
    - Architecture：x86
    - Version：3.5.0
    - Download: https://github.com/etcd-io/etcd/releases/download/v3.5.0/etcd-v3.5.0-linux-amd64.tar.gz

2. Docker Engine
    - containerd,containerd-shim,ctr,docker,dockerd,docker-init,docker-proxy,runc
    - Architecture：x86
    - Version：19.03.0
    - Download: https://download.docker.com/linux/static/stable/x86_64/docker-19.03.0.tgz

3. Kubernetes
    - kube-apiserver,kube-controller-manager,kube-scheduler,kubectl,kubelet,kube-proy
    - Architecture：x86
    - Version：1.21.1
    - Download: https://www.downloadkubernetes.com/

4. network
    - bandwidth,dhcp,flannel,host-local,loopback,portmap,sbr,tuning,vrf,bridge,firewall,host-device,ipvlan,macvlan,ptp,static,vlan
    - Architecture：x86
    - Version：v0.9.1
    - Download: https://github.com/containernetworking/plugins/releases/download/v0.9.1/cni-plugins-linux-amd64-v0.9.1.tgz

```

### Summarize

Eggo's offline deployment package should include three parts. take `kubernetes-1.21.tar.gz` as an example:

```bash
$ tar -tvf kubernetes-1.21.tar.gz
centos.config
packages/packages-x86.tar.gz
packages/packages-arm.tar.gz
notes
```

# eggops Configuration

eggops_cluster.yaml stores some resource configurations for cluster deployment by eggops. It help users configure their own eggops_cluster.yaml.

## Instructions

### Modify eggops_cluster.yaml

Users modify eggops_cluster.yaml according to actual requirements. The following modifications will be considered:
- Machine, configure a Machine resource for each available machine, which include IP, architecture, ssh login port, etc.
- Secret, configure the username/password required for machine ssh login
- PV and PVC, mount the package to the container through the shared data volume
- Infrastructure, the infrastructure configuration of the cluster, including package PVC, exposed ports, installation packages, etc.
- Cluster, the number of masters, workers, login keys, infrastructure and other information required by the cluster

For detailed configuration of eggops_cluster.yaml, please refer to docs/eggops.md

### Prepare compressed packages

The user prepares the offline installation package. The steps are the same as the steps for preparing compressed package in the eggo configuration. Don't repeat them here.
