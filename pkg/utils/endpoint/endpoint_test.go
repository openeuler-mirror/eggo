package endpoint

import (
	"testing"

	"gitee.com/openeuler/eggo/pkg/api"
)

func TestGetAPIServerEndpoint(t *testing.T) {
	ret, err := GetAPIServerEndpoint("192.168.0.1:6443", api.APIEndpoint{
		AdvertiseAddress: "127.0.0.1",
		BindPort:         6443,
	})
	if err != nil {
		t.Fatalf("invalid endpoint: %v", err)
	}
	if ret != "https://192.168.0.1:6443" {
		t.Fatalf("expect https://192.168.0.1:6443, get %s", ret)
	}

	ret, err = GetAPIServerEndpoint("https://192.168.0.1:6443", api.APIEndpoint{
		AdvertiseAddress: "127.0.0.1",
		BindPort:         6443,
	})
	if err != nil {
		t.Fatalf("invalid endpoint: %v", err)
	}
	if ret != "https://127.0.0.1:6443" {
		t.Fatalf("expect https://127.0.0.1:6443, get %s", ret)
	}
	t.Logf("test GetAPIServerEndpoint success")
}
