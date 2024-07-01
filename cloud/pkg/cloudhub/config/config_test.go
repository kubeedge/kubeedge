package config

import (
	"reflect"
	"testing"
)

func TestUpdateConfig(t *testing.T) {
	Config.UpdateCA([]byte("ca"), nil)
	if !reflect.DeepEqual(Config.Ca, []byte("ca")) {
		t.Errorf("UpdateCA(): got %v, want %v", Config.Ca, []byte("ca"))
	}
	Config.UpdateCA(nil, []byte("caKey"))
	if !reflect.DeepEqual(Config.CaKey, []byte("caKey")) {
		t.Errorf("UpdateCA(): got %v, want %v", Config.CaKey, []byte("caKey"))
	}
	Config.UpdateCerts([]byte("cert"), nil)
	if !reflect.DeepEqual(Config.Cert, []byte("cert")) {
		t.Errorf("UpdateCerts(): got %v, want %v", Config.Cert, []byte("cert"))
	}
	Config.UpdateCerts(nil, []byte("key"))
	if !reflect.DeepEqual(Config.Key, []byte("key")) {
		t.Errorf("UpdateCerts(): got %v, want %v", Config.Key, []byte("key"))
	}
}
