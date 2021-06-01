package manager

import (
	"fmt"
	"sync"

	"gitee.com/openeuler/eggo/pkg/api"
)

type ClusterDeploymentCreator func(*api.ClusterConfig) (api.ClusterDeploymentAPI, error)

type clusterDeploymentFactory struct {
	registry map[string]ClusterDeploymentCreator
	m        sync.Mutex
}

func (df *clusterDeploymentFactory) register(name string, c ClusterDeploymentCreator) error {
	df.m.Lock()
	defer df.m.Unlock()
	if _, ok := df.registry[name]; ok {
		return fmt.Errorf("driver %s is already registered", name)
	}
	df.registry[name] = c
	return nil
}

func (df *clusterDeploymentFactory) get(name string) (ClusterDeploymentCreator, error) {
	df.m.Lock()
	defer df.m.Unlock()
	c, ok := df.registry[name]
	if ok {
		return c, nil
	}
	return nil, fmt.Errorf("driver %s cannot be found", name)
}

// global factory instance
var factory = &clusterDeploymentFactory{registry: make(map[string]ClusterDeploymentCreator)}

func RegisterClusterDeploymentDriver(name string, c ClusterDeploymentCreator) error {
	if c == nil {
		return fmt.Errorf("creator is nil")
	}
	return factory.register(name, c)
}

func GetClusterDeploymentDriver(name string) (ClusterDeploymentCreator, error) {
	return factory.get(name)
}
