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

package metamanager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

var metaManagerTestDBPath string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "kubeedge-metamanager-test-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create MetaManager test directory: %v\n", err)
		os.Exit(1)
	}
	metaManagerTestDBPath = filepath.Join(tempDir, "meta.db")

	exitCode := m.Run()
	if err := os.RemoveAll(tempDir); err != nil && exitCode == 0 {
		fmt.Fprintf(os.Stderr, "failed to remove MetaManager test directory: %v\n", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func TestHandleServiceAccountTokenResponseKeepsGenerations(t *testing.T) {
	dao.Init(metaManagerTestDBPath, &edgecorev1alpha2.MetaManager{Enable: true})
	manager := newMetaManager(true)
	expirationSeconds := int64(3600)
	request := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{"https://kubernetes.default.svc"},
			ExpirationSeconds: &expirationSeconds,
			BoundObjectRef: &authenticationv1.BoundObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Name:       "test-pod",
				UID:        types.UID("test-pod-uid"),
			},
		},
	}

	first := request.DeepCopy()
	first.Status = authenticationv1.TokenRequestStatus{
		Token:               "first-token",
		ExpirationTimestamp: metav1.NewTime(time.Unix(1_700_000_000, 0)),
	}
	second := request.DeepCopy()
	second.Status = authenticationv1.TokenRequestStatus{
		Token:               "second-token",
		ExpirationTimestamp: metav1.NewTime(time.Unix(1_700_003_600, 0)),
	}

	resource := "test-ns/" + model.ResourceTypeServiceAccountToken + "/test-sa"
	for _, response := range []*authenticationv1.TokenRequest{first, second, first} {
		message := model.NewMessage("").
			BuildRouter(CloudControllerModel, GroupResource, resource, model.ResponseOperation).
			FillBody(response)
		if err := manager.handleMessage(message); err != nil {
			t.Fatalf("handleMessage() returned error: %v", err)
		}
	}

	baseKey := client.KeyFunc("test-sa", "test-ns", request)
	metas, err := manager.metaService.QueryAllMetaByKeyPrefix(baseKey)
	if err != nil {
		t.Fatalf("failed to query cached token generations: %v", err)
	}
	if metas == nil || len(*metas) != 2 {
		t.Fatalf("cached token generations = %v, want 2", metas)
	}

	wantTokens := map[string]string{
		client.KeyFunc("test-sa", "test-ns", first):  first.Status.Token,
		client.KeyFunc("test-sa", "test-ns", second): second.Status.Token,
	}
	for _, meta := range *metas {
		if meta.Key == baseKey {
			t.Errorf("token response overwrote legacy base key %q", baseKey)
		}
		var tokenRequest authenticationv1.TokenRequest
		if err := json.Unmarshal([]byte(meta.Value), &tokenRequest); err != nil {
			t.Fatalf("failed to unmarshal cached token %q: %v", meta.Key, err)
		}
		wantToken, ok := wantTokens[meta.Key]
		if !ok {
			t.Errorf("unexpected cached token key %q", meta.Key)
			continue
		}
		if tokenRequest.Status.Token != wantToken {
			t.Errorf("cached token %q = %q, want %q", meta.Key, tokenRequest.Status.Token, wantToken)
		}
	}
}
