package kubectl

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"isula.org/eggo/pkg/api"
	"isula.org/eggo/pkg/constants"
	k8scorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
	oldData, err := json.Marshal(n)
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

	newData, err := json.Marshal(n)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, k8scorev1.Node{})
	if err != nil {
		return err
	}

	rs, err := cs.CoreV1().Nodes().Patch(context.TODO(), n.Name, types.StrategicMergePatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		return err
	}
	logrus.Infof("taint and labels node: %s success", rs.GetName())

	return nil
}
