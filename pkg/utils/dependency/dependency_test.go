package dependency

import (
	"testing"

	"github.com/sirupsen/logrus"

	"isula.org/eggo/pkg/api"
)

type MockRunner struct {
}

func (m *MockRunner) Copy(src, dst string) error {
	logrus.Infof("copy %s to %s", src, dst)
	return nil
}

func (m *MockRunner) RunCommand(cmd string) (string, error) {
	logrus.Infof("run command: %s", cmd)
	return "", nil
}

func (m *MockRunner) RunShell(shell string, name string) (string, error) {
	logrus.Infof("run shell: %s, name: %s", shell, name)
	return "", nil
}

func (m *MockRunner) Reconnect() error {
	logrus.Infof("reconnect")
	return nil
}

func (m *MockRunner) Close() {
	logrus.Infof("close")
}

func TestNewDependencyShell(t *testing.T) {
	var mr MockRunner

	shell := &api.PackageConfig{
		Name:     "test.sh",
		Type:     "shell",
		Dst:      "/root",
		Schedule: api.SchedulePreJoin,
		TimeOut:  "30s",
	}

	dp := NewDependencyShell("/tmp", []*api.PackageConfig{shell})
	if err := dp.Install(&mr); err != nil {
		t.Fatalf("run test failed: %v", err)
	}
}
