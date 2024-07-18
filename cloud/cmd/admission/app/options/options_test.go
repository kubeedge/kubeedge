/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAdmissionOptions(t *testing.T) {
	assert := assert.New(t)

	opt := NewAdmissionOptions()
	assert.NotNil(opt, "Expected NewAdmissionOptions to return a non-nil value")
	assert.Equal(int32(0), opt.Port, "Expected Port to be 0 by default")
	assert.False(opt.PrintVersion, "Expected PrintVersion to be false by default")
	assert.Equal("", opt.Master)
	assert.Equal("", opt.Kubeconfig)
	assert.Equal("", opt.CertFile)
	assert.Equal("", opt.KeyFile)
	assert.Equal("", opt.CaCertFile)
	assert.Equal("", opt.AdmissionServiceName)
	assert.Equal("", opt.AdmissionServiceNamespace)
	assert.Equal("", opt.SchedulerName)
}

func TestAdmissionOptions_Flags(t *testing.T) {
	assert := assert.New(t)

	opt := NewAdmissionOptions()
	fss := opt.Flags()
	fs := fss.FlagSet("admission")

	assert.NotNil(fs, "Expected Flags to return a non-nil FlagSet")

	flags := []struct {
		name       string
		defaultVal string
		usage      string
	}{
		{
			name:       "master",
			defaultVal: "",
			usage:      "The address of the Kubernetes API server (overrides any value in kubeconfig)",
		},
		{
			name:       "kubeconfig",
			defaultVal: "",
			usage:      "Path to kubeconfig file with authorization and master location information.",
		},
		{
			name:       "tls-cert-file",
			defaultVal: "",
			usage:      "File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert).",
		},
		{
			name:       "tls-private-key-file",
			defaultVal: "",
			usage:      "File containing the default x509 private key matching --tls-cert-file.",
		},
		{
			name:       "ca-cert-file",
			defaultVal: "",
			usage:      "File containing the x509 Certificate for HTTPS.",
		},
		{
			name:       "port",
			defaultVal: "443",
			usage:      "the port used by admission-controller-server.",
		},
		{
			name:       "webhook-namespace",
			defaultVal: "kubeedge",
			usage:      "The namespace of this webhook",
		},
		{
			name:       "webhook-service-name",
			defaultVal: "kubeedge-admission-service",
			usage:      "The name of this admission service",
		},
	}

	for _, f := range flags {
		flag := fs.Lookup(f.name)
		assert.NotNil(flag, "Expected '%s' flag to be present in the FlagSet", f.name)
		assert.Equal(f.name, flag.Name)
		assert.Equal(f.defaultVal, flag.DefValue)
		assert.Equal(f.usage, flag.Usage)
	}

	err := fs.Parse([]string{
		"--master=http://localhost:8080",
		"--kubeconfig=/path/to/kubeconfig",
		"--tls-cert-file=/path/to/cert",
		"--tls-private-key-file=/path/to/key",
		"--ca-cert-file=/path/to/ca",
		"--port=8443",
		"--webhook-namespace=test-namespace",
		"--webhook-service-name=test-service",
	})
	assert.NoError(err)

	assert.Equal("http://localhost:8080", opt.Master)
	assert.Equal("/path/to/kubeconfig", opt.Kubeconfig)
	assert.Equal("/path/to/cert", opt.CertFile)
	assert.Equal("/path/to/key", opt.KeyFile)
	assert.Equal("/path/to/ca", opt.CaCertFile)
	assert.Equal(int32(8443), opt.Port)
	assert.Equal("test-namespace", opt.AdmissionServiceNamespace)
	assert.Equal("test-service", opt.AdmissionServiceName)
	assert.Equal(false, opt.PrintVersion, "Expected PrintVersion to be false by default")
	assert.Equal("", opt.SchedulerName, "Expected SchedulerName to be an empty string by default")
}
