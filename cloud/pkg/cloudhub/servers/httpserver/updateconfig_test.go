package httpserver

import (
	"reflect"
	"testing"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

func TestUpdateConfig(t *testing.T) {
	UpdateConfig([]byte("ca"), nil, nil, nil)
	if !reflect.DeepEqual(config.Config.Ca, []byte("ca")) {
		t.Errorf("UpdateConfig(): got %v, want %v", config.Config.Ca, []byte("ca"))
	}
	UpdateConfig(nil, []byte("caKey"), nil, nil)
	if !reflect.DeepEqual(config.Config.CaKey, []byte("caKey")) {
		t.Errorf("UpdateConfig(): got %v, want %v", config.Config.CaKey, []byte("caKey"))
	}
	UpdateConfig(nil, nil, []byte("cert"), nil)
	if !reflect.DeepEqual(config.Config.Cert, []byte("cert")) {
		t.Errorf("UpdateConfig(): got %v, want %v", config.Config.Cert, []byte("cert"))
	}
	UpdateConfig(nil, nil, nil, []byte("key"))
	if !reflect.DeepEqual(config.Config.Key, []byte("key")) {
		t.Errorf("UpdateConfig(): got %v, want %v", config.Config.Key, []byte("key"))
	}
}
