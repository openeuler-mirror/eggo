package addons

import "gitee.com/openeuler/eggo/pkg/api"

// TODO: support run apply addons in eggo, not run in master

func SetupAddons(cluster *api.ClusterConfig) error {
	return setupAddons(cluster)
}

func CleanupAddons(cluster *api.ClusterConfig) error {
	return cleanupAddons(cluster)
}
