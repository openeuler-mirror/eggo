package kubectl

import (
	"fmt"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"github.com/sirupsen/logrus"
)

type Taint struct {
	Key    string
	Value  string
	Effect string
}

func (t Taint) ToString() (string, error) {
	if t.Key == "" {
		return "", fmt.Errorf("empty key")
	}
	if t.Value != "" {
		return fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect), nil
	}
	if t.Effect == "" {
		return t.Key, nil
	}
	return fmt.Sprintf("%s:%s", t.Key, t.Effect), nil
}

func AddNodeTaints(cluster *api.ClusterConfig, r runner.Runner, objectName string, taints []Taint) error {
	for _, t := range taints {
		tstr, err := t.ToString()
		if err != nil {
			logrus.Warnf("invalid taint: %v", err)
			continue
		}
		cmd := fmt.Sprintf("taint nodes %s %s", objectName, tstr)
		err = RunKubectlCmd(r, cmd, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func AddNodeLabels(cluster *api.ClusterConfig, r runner.Runner, objectName string, lables map[string]string) error {
	for k, v := range lables {
		cmd := fmt.Sprintf("label node %s %s=%s", objectName, k, v)
		err := RunKubectlCmd(r, cmd, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
