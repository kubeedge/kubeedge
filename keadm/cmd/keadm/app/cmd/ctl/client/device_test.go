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

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

// patchVersionedClient points metaclient.VersionedKubeClient at a test server that answers
// the device update with the given status code.
func patchVersionedClient(t *testing.T, code int) *gomonkey.Patches {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if code == http.StatusOK {
			_ = json.NewEncoder(w).Encode(&v1beta1.Device{ObjectMeta: metav1.ObjectMeta{Name: "dev", Namespace: "default"}})
			return
		}
		_ = json.NewEncoder(w).Encode(&metav1.Status{
			TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
			Status:   metav1.StatusFailure,
			Message:  "device dev is invalid",
			Reason:   metav1.StatusReasonInvalid,
			Code:     int32(code),
		})
	}))
	t.Cleanup(server.Close)

	client, err := versioned.NewForConfig(&rest.Config{Host: server.URL})
	assert.NoError(t, err)
	return gomonkey.ApplyFunc(metaclient.VersionedKubeClient, func() (versioned.Interface, error) {
		return client, nil
	})
}

func TestUpdateDeviceReturnsTheServerError(t *testing.T) {
	patches := patchVersionedClient(t, http.StatusUnprocessableEntity)
	defer patches.Reset()

	request := &DeviceRequest{Namespace: "default", DeviceName: "dev"}
	device := &v1beta1.Device{ObjectMeta: metav1.ObjectMeta{Name: "dev", Namespace: "default"}}
	res, err := request.UpdateDevice(context.Background(), device)

	assert.Nil(t, res)
	assert.ErrorContains(t, err, "failed to update device default/dev")
	assert.ErrorContains(t, err, "device dev is invalid")
}

func TestUpdateDeviceSucceeds(t *testing.T) {
	patches := patchVersionedClient(t, http.StatusOK)
	defer patches.Reset()

	request := &DeviceRequest{Namespace: "default", DeviceName: "dev"}
	device := &v1beta1.Device{ObjectMeta: metav1.ObjectMeta{Name: "dev", Namespace: "default"}}
	res, err := request.UpdateDevice(context.Background(), device)

	assert.NoError(t, err)
	assert.NotNil(t, res)
}
