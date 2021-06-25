package addons

import (
	"fmt"
	"path/filepath"
	"time"

	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	"isula.org/eggo/pkg/utils/nodemanager"
	"isula.org/eggo/pkg/utils/runner"
	"isula.org/eggo/pkg/utils/task"
	"isula.org/eggo/pkg/utils/template"
	"github.com/sirupsen/logrus"
)

func runCmd(name string, r runner.Runner, cluster *api.ClusterConfig, tmpl string, datastore map[string]interface{}) error {
	cmdStr, err := template.TemplateRender(tmpl, datastore)
	if err != nil {
		return err
	}
	output, err := r.RunShell(cmdStr, name)
	if err != nil {
		return err
	}

	logrus.Debugf("apply %s success\noutput: %s", name, output)
	return nil
}

type SetupAddonsTask struct {
	Cluster *api.ClusterConfig
}

func (ct *SetupAddonsTask) Name() string {
	return "AddonsTask"
}

func (ct *SetupAddonsTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	addonsTmpl := `
#!/bin/bash
export KUBECONFIG={{ .KubeConfig }}
{{- range $i, $v := .Addons }}
kubectl apply -f {{ $v }}
if [ $? -ne 0 ]; then
	echo "apply {{ $v }} failed"
	exit 1
fi
{{- end }}
exit 0
`
	datastore := make(map[string]interface{})
	var addons []string
	for _, a := range ct.Cluster.Addons {
		addons = append(addons, filepath.Join(constants.DefaultK8SAddonsDir, a.Filename))
	}
	datastore["Addons"] = addons
	datastore["KubeConfig"] = filepath.Join(ct.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	return runCmd("apply-addons", r, ct.Cluster, addonsTmpl, datastore)
}

func setupAddons(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}
	if cluster.Addons == nil || len(cluster.Addons) == 0 {
		logrus.Debugf("no addons found")
		return nil
	}
	t := task.NewTaskInstance(&SetupAddonsTask{Cluster: cluster})
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
	err = nodemanager.WaitTaskOnNodesFinished(t, []string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}
	logrus.Infof("[cluster] apply addons success")
	return nil
}

type CleanupAddonsTask struct {
	Cluster *api.ClusterConfig
}

func (ct *CleanupAddonsTask) Name() string {
	return "CleanupAddonsTask"
}

func (ct *CleanupAddonsTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	addonsTmpl := `
#!/bin/bash
export KUBECONFIG={{ .KubeConfig }}
{{- range $i, $v := .Addons }}
kubectl delete --force=true -f {{ $v }}
if [ $? -ne 0 ]; then
	echo "apply {{ $v }} failed"
	exit 1
fi
{{- end }}
exit 0
`
	datastore := make(map[string]interface{})
	var addons []string
	for _, a := range ct.Cluster.Addons {
		addons = append(addons, filepath.Join(constants.DefaultK8SAddonsDir, a.Filename))
	}
	datastore["Addons"] = addons
	datastore["KubeConfig"] = filepath.Join(ct.Cluster.GetConfigDir(), constants.KubeConfigFileNameAdmin)

	return runCmd("delete-addons", r, ct.Cluster, addonsTmpl, datastore)
}

func cleanupAddons(cluster *api.ClusterConfig) error {
	if cluster == nil {
		return fmt.Errorf("invalid cluster config")
	}
	if cluster.Addons == nil || len(cluster.Addons) == 0 {
		logrus.Debugf("no addons found")
		return nil
	}
	t := task.NewTaskInstance(&CleanupAddonsTask{Cluster: cluster})
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
	err = nodemanager.WaitTaskOnNodesFinished(t, []string{useMaster}, 5*time.Minute)
	if err != nil {
		return err
	}
	logrus.Infof("[cluster] cleanup addons success")
	return nil
}
