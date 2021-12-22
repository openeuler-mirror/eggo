package dependency

import (
	"testing"

	"isula.org/eggo/pkg/api"
)

func TestCopyHooks(t *testing.T) {
	var mr MockRunner

	hs := &api.ClusterHookConf{
		Type:       api.PreHookType,
		Operator:   api.HookOpDeploy,
		Target:     api.Master,
		HookSrcDir: "/tmp",
		HookFiles:  []string{"test.sh", "test2.bash"},
	}

	node := &api.HostConfig{}

	ct := &CopyHooksTask{hooks: hs}
	if err := ct.Run(&mr, node); err != nil {
		t.Fatalf("run test failed: %v", err)
	}
}

func TestExecuteCmdHooks(t *testing.T) {
	hooks := &api.ClusterHookConf{
		Target:   api.Master,
		Operator: api.HookOpDeploy,
		Type:     api.PreHookType,
	}
	host := &api.HostConfig{
		Type: api.Master,
	}
	ccfg := &api.ClusterConfig{
		HooksConf: []*api.ClusterHookConf{hooks},
	}
	if err := ExecuteCmdHooks(ccfg, []*api.HostConfig{host}, api.HookOpJoin, api.PostHookType); err != nil {
		t.Fatalf("run test failed: %v", err)
	}
}
