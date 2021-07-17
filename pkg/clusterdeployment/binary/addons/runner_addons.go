package addons

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/dependency"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
)

type SetupAddonsTask struct {
	yaml       []*api.PackageConfig
	srcPath    string
	kubeconfig string
}

func (ct *SetupAddonsTask) Name() string {
	return "AddonsTask"
}

func (ct *SetupAddonsTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	logrus.Info("do apply addons...")

	yamlDep := dependency.NewDependencyYaml(ct.srcPath, ct.kubeconfig, ct.yaml)
	if err := yamlDep.Install(r); err != nil {
		return err
	}

	logrus.Info("apply addons success")
	return nil
}

func getAddons(cluster *api.ClusterConfig) []*api.PackageConfig {
	yaml := []*api.PackageConfig{}
	for _, s := range cluster.RoleInfra[api.Master].Softwares {
		if s.Type == "yaml" {
			yaml = append(yaml, s)
		}
	}

	return yaml
}

func setupAddons(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}

	yaml := getAddons(cluster)
	if len(yaml) == 0 {
		logrus.Warn("no addons load")
		return nil
	}

	yamlPath := filepath.Join(cluster.PackageSrc.GetPkgDstPath(), constants.DefaultFilePath)
	kubeconfig := filepath.Join(cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	t := task.NewTaskInstance(&SetupAddonsTask{
		yaml:       yaml,
		srcPath:    yamlPath,
		kubeconfig: kubeconfig,
	})
	var masters []string
	for _, n := range cluster.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
		}
	}

	useMaster, err := nodemanager.RunTaskOnOneNode(t, masters)
	if err != nil {
		return err
	}
	err = nodemanager.WaitNodesFinish([]string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}

	logrus.Infof("[cluster] apply addons success")
	return nil
}

type CleanupAddonsTask struct {
	yaml       []*api.PackageConfig
	srcPath    string
	kubeconfig string
}

func (ct *CleanupAddonsTask) Name() string {
	return "CleanupAddonsTask"
}

func (ct *CleanupAddonsTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	logrus.Info("do remove addons...")

	yamlDep := dependency.NewDependencyYaml(ct.srcPath, ct.kubeconfig, ct.yaml)
	if err := yamlDep.Remove(r); err != nil {
		return err
	}

	logrus.Info("remove addons success")
	return nil
}

func cleanupAddons(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}

	yaml := getAddons(cluster)
	if len(yaml) == 0 {
		logrus.Warn("no addons load")
		return nil
	}

	yamlPath := filepath.Join(cluster.PackageSrc.GetPkgDstPath(), constants.DefaultFilePath)
	kubeconfig := filepath.Join(cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	t := task.NewTaskInstance(&CleanupAddonsTask{
		yaml:       yaml,
		srcPath:    yamlPath,
		kubeconfig: kubeconfig,
	})
	var masters []string
	for _, n := range cluster.Nodes {
		if (n.Type & api.Master) != 0 {
			masters = append(masters, n.Address)
		}
	}

	task.SetIgnoreErrorFlag(t)
	useMaster, err := nodemanager.RunTaskOnOneNode(t, masters)
	if err != nil {
		return err
	}
	err = nodemanager.WaitNodesFinishWithProgress([]string{useMaster}, time.Minute)
	if err != nil {
		return err
	}
	logrus.Infof("[cluster] cleanup addons success")
	return nil
}
