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

package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewLeases(t *testing.T) {
	assert := assert.New(t)

	namespace := "test-namespace"
	s := newSend()

	leases := newLeases(namespace, s)

	assert.NotNil(leases)
	assert.Equal(namespace, leases.namespace)
	assert.IsType(&send{}, leases.send)
}

func TestHandleLeaseResp(t *testing.T) {
	assert := assert.New(t)

	t.Run("Valid lease response", func(t *testing.T) {
		lease := &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-lease",
			},
		}
		leaseResp := LeaseResp{
			Object: lease,
		}
		content, _ := json.Marshal(leaseResp)

		result, err := handleLeaseResp(content)

		assert.NoError(err)
		assert.Equal(lease, result)
	})

	t.Run("Response with error", func(t *testing.T) {
		statusErr := apierrors.StatusError{
			ErrStatus: metav1.Status{
				Message: "Test error",
				Reason:  metav1.StatusReasonNotFound,
				Code:    404,
			},
		}
		leaseResp := LeaseResp{
			Err: statusErr,
		}
		content, _ := json.Marshal(leaseResp)

		result, err := handleLeaseResp(content)

		assert.Error(err)
		assert.Nil(result)
		assert.Equal(&statusErr, err)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		content := []byte(`{"invalid": json}`)

		result, err := handleLeaseResp(content)

		assert.Error(err)
		assert.Nil(result)
		assert.Contains(err.Error(), "unmarshal message to lease failed")
	})
}
