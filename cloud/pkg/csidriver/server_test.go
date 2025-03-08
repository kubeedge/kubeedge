/*
Copyright 2025 The KubeEdge Authors.

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

package csidriver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/cloud/cmd/csidriver/app/options"
)

func TestNewCSIDriver(t *testing.T) {
	tests := []struct {
		name    string
		opts    *options.CSIDriverOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			opts: &options.CSIDriverOptions{
				Endpoint:         "unix:///var/lib/kubelet/plugins/csi-hostpath/csi.sock",
				DriverName:       "hostpath.csi.k8s.io",
				NodeID:           "node1",
				KubeEdgeEndpoint: "http://localhost:10000",
				Version:          "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			opts: &options.CSIDriverOptions{
				DriverName:       "hostpath.csi.k8s.io",
				NodeID:           "node1",
				KubeEdgeEndpoint: "http://localhost:10000",
				Version:          "1.0.0",
			},
			wantErr: true,
			errMsg:  "no driver endpoint provided",
		},
		{
			name: "missing driver name",
			opts: &options.CSIDriverOptions{
				Endpoint:         "unix:///var/lib/kubelet/plugins/csi-hostpath/csi.sock",
				NodeID:           "node1",
				KubeEdgeEndpoint: "http://localhost:10000",
				Version:          "1.0.0",
			},
			wantErr: true,
			errMsg:  "no driver name provided",
		},
		{
			name: "missing node ID",
			opts: &options.CSIDriverOptions{
				Endpoint:         "unix:///var/lib/kubelet/plugins/csi-hostpath/csi.sock",
				DriverName:       "hostpath.csi.k8s.io",
				KubeEdgeEndpoint: "http://localhost:10000",
				Version:          "1.0.0",
			},
			wantErr: true,
			errMsg:  "no node id provided",
		},
		{
			name: "missing kubeedge endpoint",
			opts: &options.CSIDriverOptions{
				Endpoint:   "unix:///var/lib/kubelet/plugins/csi-hostpath/csi.sock",
				DriverName: "hostpath.csi.k8s.io",
				NodeID:     "node1",
				Version:    "1.0.0",
			},
			wantErr: true,
			errMsg:  "no kubeedge endpoint provided",
		},
		{
			name: "missing version",
			opts: &options.CSIDriverOptions{
				Endpoint:         "unix:///var/lib/kubelet/plugins/csi-hostpath/csi.sock",
				DriverName:       "hostpath.csi.k8s.io",
				NodeID:           "node1",
				KubeEdgeEndpoint: "http://localhost:10000",
			},
			wantErr: true,
			errMsg:  "no version provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, err := NewCSIDriver(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
				assert.Nil(t, driver)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, driver)
				assert.Equal(t, tt.opts.Endpoint, driver.Endpoint)
				assert.Equal(t, tt.opts.DriverName, driver.DriverName)
				assert.Equal(t, tt.opts.NodeID, driver.NodeID)
				assert.Equal(t, tt.opts.KubeEdgeEndpoint, driver.KubeEdgeEndpoint)
				assert.Equal(t, tt.opts.Version, driver.Version)
			}
		})
	}
}

func TestCSIDriverRun(t *testing.T) {
	opts := &options.CSIDriverOptions{
		Endpoint:         "unix:///tmp/test.sock",
		DriverName:       "test.csi.k8s.io",
		NodeID:           "test-node",
		KubeEdgeEndpoint: "http://localhost:10000",
		Version:          "1.0.0",
	}

	driver, err := NewCSIDriver(opts)
	assert.NoError(t, err)
	assert.NotNil(t, driver)

	done := make(chan bool)
	go func() {
		driver.Run()
		done <- true
	}()
}
