package kubectl

import (
	"context"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	k8scorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Taint struct {
	Key    string
	Value  string
	Effect string
}

func NodeTaintAndLabel(cluster string, objectName string, labels map[string]string, taints []Taint) error {
	path := filepath.Join(api.GetClusterHomePath(cluster), constants.KubeConfigFileNameAdmin)
	cs, err := GetKubeClient(path)
	if err != nil {
		return err
	}

	n, err := cs.CoreV1().Nodes().Get(context.TODO(), objectName, v1.GetOptions{})
	if err != nil {
		return err
	}
	var ktaints []k8scorev1.Taint

	for _, taint := range taints {
		t := k8scorev1.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: k8scorev1.TaintEffect(taint.Effect),
		}
		flag := false
		for _, tt := range n.Spec.Taints {
			if tt == t {
				flag = true
				break
			}
		}
		if flag {
			continue
		}
		ktaints = append(ktaints, t)
	}
	n.Spec.Taints = append(n.Spec.Taints, ktaints...)
	for k, v := range labels {
		n.Labels[k] = v
	}

	rs, err := cs.CoreV1().Nodes().Update(context.TODO(), n, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	logrus.Infof("taint and labels node: %s success", rs.GetName())

	return nil
}
