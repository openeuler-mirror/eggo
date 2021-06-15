# eggo

### 介绍
Eggo项目旨在解决大规模生产环境K8S集群自动化部署问题、部署流程跟踪以及提供高度的灵活性。通过结合GitOps管理、跟踪部署配置，通过云原生的方式实现集群的部署，实现集群部署集群的能力。

### 软件架构

[软件架构说明](./docs/design.md)

### 编译安装

```bash
# 使能go mod
$ go env -w GO111MODULE=on
# 设置goproxy为国内代理，也可以设置为其他公司的代理
$ go env -w GOPROXY=https://goproxy.cn,direct
# 下载依赖库
$ go mod tidy
# 编译
$ make
# 使用vendor本地编译，前提需要之前下载过依赖的go库
$ go mod vendor
$ make local
# 安装
$ make install
```

### 运行测试

```bash
$ make test
```

### 基本用法

```bash
# 生成默认配置模板
$ eggo template -f test.yaml
# 生成指定master节点IP列表的模板
$ eggo template  --masters=192.168.0.1  --masters=192.168.0.2 -f test.yaml
# template当前支持多个参数覆盖默认值
$ ./eggo template --help
      --etcds stringArray          set etcd node ips
  -l, --loadbalancer stringArray   set loadbalancer node (default [192.168.0.1])
      --masters stringArray        set master ips (default [192.168.0.2])
  -n, --name string                set cluster name (default "k8s-cluster")
      --nodes stringArray          set worker ips (default [192.168.0.3,192.168.0.4])
  -p, --password string            password to login all node (default "123456")
  -u, --user string                user to login all node (default "root")

# 使用上面template命令生成的配置文件，创建集群
$ eggo deploy -f test.yaml

# 使用上面的配置清理集群
$ eggo cleanup -f test.yaml
```

详细的用法见 https://gitee.com/openeuler/eggo/blob/master/docs/manual.md

### 感谢

本项目受[Kubekey](https://github.com/kubesphere/kubekey)的启发，感谢Kubekey的伟大工作。

### 参与贡献

1.  Fork 本仓库
2.  新建 Feat_xxx 分支
3.  提交代码
4.  新建 Pull Request
