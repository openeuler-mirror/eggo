# eggo

### Description
The Eggo project was designed to automate the deployment of K8S clusters in mass production environments, track deployment processes, and provide a high degree of flexibility. By combining GitOps management and deployment configuration tracking, cluster deployment is implemented in cloud native mode, enabling cluster management.

- Support multi-release version of Linux: such as openEuler/CentOS/Ubuntu；
- Support multi-architecture (amd64/arm64) deployment: a cluster supports nodes of multiple architectures;
- Support for multiple deployments: binary and KUbeadm (to be implemented);
- Support offline and online deployment;

Currently, eggo implements the deployment using the command. The following are three deployment modes that are supported by eggo:


- Online deployment. Only need to write the `yaml` configuration file for the deployment. The required rpm package/binary file/plug-in/docker image are downloaded during the installation and deployment phase according to the internet. Online deployment Currently, plug-ins cannot be downloaded and installed online. Plug-ins will be deployed online in the future. Details see [eggo operation manual](/docs/manual.md).
- Offline deployment. Package all rpm packages/binary files/plug-in/docker images into a `tar.gz` file in a certain format. Then write the corresponding `yaml` configuration file (details see [eggo operation manual](/docs/manual.md)), the cluster will be deployed by executing commands.  
- Using cluster deploy new cluster by Gitops (to be implemented).

### Software Architecture
detailed [Software architecture description](./docs/general_design.md)

### Detailed usage
detailed [eggo operation manual](https://docs.openeuler.org/zh/docs/21.09/docs/Kubernetes/eggo%E8%87%AA%E5%8A%A8%E5%8C%96%E9%83%A8%E7%BD%B2.html)


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
