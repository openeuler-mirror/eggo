# eggo

### Description
The Eggo project was designed to automate deployment of K8S clusters in mass production environments, track deployment processes, and provide a high degree of flexibility. By combining GitOps management and deployment configuration tracking, cluster deployment is implemented in cloud native mode, enabling cluster management.

- Support for multiple deployments: binary and KUbeadm (to be implemented);
- Supports multi-architecture deployment: a cluster supports nodes of multiple architectures;
- Supports offline and online deployment;

### Software Architecture
[Software architecture description](./docs/design.md)

### Build and install

```bash
# enable go mod
$ export GO111MODULE=on
# set goproxy
$ go env -w GOPROXY=https://goproxy.cn,direct
# download dependences
$ go mod tidy
# compile
$ make
# use vendor to compile, must download dependences go library at first
$ go mod vendor
$ make local
# install
$ make install
```

### Unit test

```bash
$ make test
```

### Usages

```bash
# generate default template for cluster
$ eggo template -f test.yaml
# use special master ips to generate template
$ eggo template  --masters=192.168.0.1  --masters=192.168.0.2 -f test.yaml
# current support arguments for subcommand template
$ ./eggo template --help
      --etcds stringArray          set etcd node ips
  -l, --loadbalancer stringArray   set loadbalancer node (default [192.168.0.1])
      --masters stringArray        set master ips (default [192.168.0.2])
  -n, --name string                set cluster name (default "k8s-cluster")
      --nodes stringArray          set worker ips (default [192.168.0.3,192.168.0.4])
  -p, --password string            password to login all node (default "123456")
  -u, --user string                user to login all node (default "root")

# use generated config to deploy cluster
$ eggo deploy -f test.yaml

# use generated config to cleanup cluster
$ eggo cleanup -f test.yaml
```

see https://gitee.com/openeuler/eggo/blob/master/docs/manual.md for detail usage.

### Releases

```
# Step 1: update file of VERSION, and push pr
$ vi VERSION
# Step 2: get release note by call releasenote.sh
$ ./hack/releasenote.sh
```

### Gratitude

The design of Eggo was inspired by [Kubekey](https://github.com/kubesphere/kubekey), thanks to their great work.

### Contribution

1.  Fork the repository
2.  Create Feat_xxx branch
3.  Commit your code
4.  Create Pull Request
