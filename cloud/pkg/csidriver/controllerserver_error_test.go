/*
Copyright 2026 The KubeEdge Authors.

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
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestControllerServerPreservesResponseErrors(t *testing.T) {
	responseErr := "edge response failed"

	tests := []struct {
		name string
		call func(context.Context, *controllerServer) error
	}{
		{
			name: "DeleteVolume",
			call: func(ctx context.Context, cs *controllerServer) error {
				_, err := cs.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: "volume-id"})
				return err
			},
		},
		{
			name: "ControllerPublishVolume",
			call: func(ctx context.Context, cs *controllerServer) error {
				_, err := cs.ControllerPublishVolume(ctx, &csi.ControllerPublishVolumeRequest{
					VolumeId: "volume-id",
					NodeId:   "node-id",
				})
				return err
			},
		},
		{
			name: "ControllerUnpublishVolume",
			call: func(ctx context.Context, cs *controllerServer) error {
				_, err := cs.ControllerUnpublishVolume(ctx, &csi.ControllerUnpublishVolumeRequest{
					VolumeId: "volume-id",
					NodeId:   "node-id",
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &controllerServer{
				nodeID:           "test-node",
				kubeEdgeEndpoint: startResponseErrorServer(t, responseErr),
			}

			err := tt.call(context.Background(), cs)
			if err == nil {
				t.Fatalf("expected error")
			}
			if err.Error() != responseErr {
				t.Fatalf("expected %q, got %q", responseErr, err.Error())
			}
		})
	}
}

func startResponseErrorServer(t *testing.T, responseErr string) string {
	t.Helper()

	socketPath := filepath.Join(t.TempDir(), "csidriver.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on socket: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveResponseError(listener, responseErr)
	}()

	t.Cleanup(func() {
		listener.Close()
		if err := <-errCh; err != nil {
			t.Errorf("response error server failed: %v", err)
		}
	})

	return "unix://" + socketPath
}

func serveResponseError(listener net.Listener, responseErr string) error {
	conn, err := listener.Accept()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		return err
	}
	defer conn.Close()

	request := new(model.Message)
	if err := json.NewDecoder(conn).Decode(request); err != nil {
		return err
	}
	response := model.NewErrorMessage(request, responseErr).
		SetRoute(DefaultReceiveModuleName, request.GetGroup())
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}
