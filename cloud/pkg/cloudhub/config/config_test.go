package config

import (
	"encoding/pem"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

// Helper function to create valid PEM files
func createValidPEMFile(t *testing.T, filepath string, blockType string) {
	block := &pem.Block{
		Type:  blockType,
		Bytes: []byte("test data"),
	}
	pemData := pem.EncodeToMemory(block)
	if err := os.WriteFile(filepath, pemData, 0644); err != nil {
		t.Fatalf("Failed to write PEM file: %v", err)
	}
}

func TestUpdateConfig(t *testing.T) {
	originalConfig := Configure{}

	t.Run("Update CA", func(t *testing.T) {
		Config = originalConfig
		Config.UpdateCA([]byte("ca"), nil)
		if !reflect.DeepEqual(Config.Ca, []byte("ca")) {
			t.Errorf("UpdateCA(): got %v, want %v", Config.Ca, []byte("ca"))
		}
		Config.UpdateCA(nil, []byte("caKey"))
		if !reflect.DeepEqual(Config.CaKey, []byte("caKey")) {
			t.Errorf("UpdateCA(): got %v, want %v", Config.CaKey, []byte("caKey"))
		}
	})

	t.Run("Update Certs", func(t *testing.T) {
		Config = originalConfig
		Config.UpdateCerts([]byte("cert"), nil)
		if !reflect.DeepEqual(Config.Cert, []byte("cert")) {
			t.Errorf("UpdateCerts(): got %v, want %v", Config.Cert, []byte("cert"))
		}
		Config.UpdateCerts(nil, []byte("key"))
		if !reflect.DeepEqual(Config.Key, []byte("key")) {
			t.Errorf("UpdateCerts(): got %v, want %v", Config.Key, []byte("key"))
		}
	})
}

func TestInitConfigureBasic(t *testing.T) {
	// Reset global state
	Config = Configure{}
	once = sync.Once{}

	hub := &v1alpha1.CloudHub{
		AdvertiseAddress: []string{"127.0.0.1"},
	}

	// Test basic initialization
	InitConfigure(hub)

	if len(Config.AdvertiseAddress) != 1 || Config.AdvertiseAddress[0] != "127.0.0.1" {
		t.Errorf("Basic initialization failed, got address %v, want [127.0.0.1]", Config.AdvertiseAddress)
	}
}

func TestInitConfigureWithValidCerts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid PEM files
	caFile := tmpDir + "/ca.crt"
	caKeyFile := tmpDir + "/ca.key"
	certFile := tmpDir + "/server.crt"
	keyFile := tmpDir + "/server.key"

	createValidPEMFile(t, caFile, "CERTIFICATE")
	createValidPEMFile(t, caKeyFile, "RSA PRIVATE KEY")
	createValidPEMFile(t, certFile, "CERTIFICATE")
	createValidPEMFile(t, keyFile, "RSA PRIVATE KEY")

	// Reset global state
	Config = Configure{}
	once = sync.Once{}

	hub := &v1alpha1.CloudHub{
		AdvertiseAddress:  []string{"127.0.0.1"},
		TLSCAFile:         caFile,
		TLSCAKeyFile:      caKeyFile,
		TLSCertFile:       certFile,
		TLSPrivateKeyFile: keyFile,
	}

	InitConfigure(hub)

	// Verify that certificates were loaded
	if Config.Ca == nil || Config.CaKey == nil || Config.Cert == nil || Config.Key == nil {
		t.Error("Failed to load certificates")
	}
}

func TestInitConfigureWithInvalidPaths(_ *testing.T) {
	// Reset global state
	Config = Configure{}
	once = sync.Once{}

	hub := &v1alpha1.CloudHub{
		AdvertiseAddress: []string{"127.0.0.1"},
		TLSCAFile:        "/nonexistent/ca.crt",
		TLSCertFile:      "/nonexistent/cert.crt",
	}

	// This should not panic, only log warnings
	InitConfigure(hub)
}

func TestConcurrentAccess(t *testing.T) {
	// Reset global state
	Config = Configure{}
	once = sync.Once{}

	hub := &v1alpha1.CloudHub{
		AdvertiseAddress: []string{"127.0.0.1"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			InitConfigure(hub)
		}()
	}
	wg.Wait()

	if len(Config.AdvertiseAddress) != 1 || Config.AdvertiseAddress[0] != "127.0.0.1" {
		t.Error("Concurrent initialization failed")
	}
}
