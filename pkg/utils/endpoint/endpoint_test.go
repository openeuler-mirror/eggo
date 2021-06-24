package endpoint

import (
	"testing"

	"isula.org/eggo/pkg/api"
)

func TestGetAPIServerEndpoint(t *testing.T) {
	ret, err := GetAPIServerEndpoint(&api.ClusterConfig{
		APIEndpoint: api.APIEndpoint{
			AdvertiseAddress: "192.168.0.1",
			BindPort:         6443,
		},
	})
	if err != nil {
		t.Fatalf("invalid endpoint: %v", err)
	}
	if ret != "https://192.168.0.1:6443" {
		t.Fatalf("expect https://192.168.0.1:6443, get %s", ret)
	}

	t.Logf("test GetAPIServerEndpoint success")
}
