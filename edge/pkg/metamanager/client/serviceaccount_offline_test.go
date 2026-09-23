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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// projectedTTL is the lifetime kubelet requests for a projected service
// account token volume.
const projectedTTL int64 = 3607

// These tests drive the real metaDB through dao.Init rather than stubbing the
// data layer, so they exercise the per-generation key scheme end to end.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "kubeedge-sa-offline")
	if err != nil {
		panic(err)
	}
	dao.Init(filepath.Join(dir, "meta.db"), &v1alpha2.MetaManager{Enable: true})
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// offlineSend fails every sync send, which is what an edge node that has lost
// its connection to the cloud sees.
type offlineSend struct{}

func (offlineSend) SendSync(*model.Message) (*model.Message, error) {
	return nil, fmt.Errorf("timeout to get response")
}
func (offlineSend) Send(*model.Message) {}

func specOf(ttl int64) *authenticationv1.TokenRequest {
	exp := ttl
	return &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{ExpirationSeconds: &exp},
	}
}

// cacheGeneration stores one generation of a token, issued age ago, under the
// per-generation key that metamanager would use for it.
func cacheGeneration(t *testing.T, name, namespace string, age time.Duration, token string) *authenticationv1.TokenRequest {
	t.Helper()
	exp := projectedTTL
	issued := time.Now().Add(-age)
	tr := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{ExpirationSeconds: &exp},
		Status: authenticationv1.TokenRequestStatus{
			Token:               token,
			ExpirationTimestamp: metav1.NewTime(issued.Add(time.Duration(projectedTTL) * time.Second)),
		},
	}
	value, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	if err := dbclient.NewMetaService().SaveMeta(&models.Meta{
		Key:   KeyFunc(name, namespace, tr),
		Type:  model.ResourceTypeServiceAccountToken,
		Value: string(value),
	}); err != nil {
		t.Fatalf("save meta: %v", err)
	}
	return tr
}

// A refresh stores the new token under its own key, so the previous generation
// survives the propagation window during which a pod may still present it.
func TestGenerationsCoexistAcrossRefresh(t *testing.T) {
	const name, namespace = "coexist", "default"
	cacheGeneration(t, name, namespace, 50*time.Minute, "tok-old")
	cacheGeneration(t, name, namespace, time.Minute, "tok-new")

	rows, err := dbclient.NewMetaService().QueryMetaByKeyPrefix(baseKey(name, namespace, specOf(projectedTTL)))
	if err != nil {
		t.Fatalf("query by prefix: %v", err)
	}
	if rows == nil || len(*rows) != 2 {
		t.Fatalf("cached generations = %d, want 2", len(*rows))
	}

	// Both remain usable for authentication while they are un-expired.
	if !CheckTokenExist("tok-old") {
		t.Error("the previous generation was rejected while it was still valid")
	}
	if !CheckTokenExist("tok-new") {
		t.Error("the refreshed generation was rejected")
	}

	// The newest generation is the one handed to a caller.
	got, err := newestCachedGeneration(name, namespace, specOf(projectedTTL))
	if err != nil {
		t.Fatalf("newestCachedGeneration: %v", err)
	}
	if got.Status.Token != "tok-new" {
		t.Errorf("token = %q, want %q", got.Status.Token, "tok-new")
	}
}

// With the cloud unreachable and a cached token that is past the refresh
// threshold but not expired, the cached token is served so the pod can start.
func TestGetServiceAccountTokenServesStaleTokenWhenOffline(t *testing.T) {
	const name, namespace = "offline-stale", "default"
	tr := cacheGeneration(t, name, namespace, 50*time.Minute, "tok-stale")

	if !requiresRefresh(tr) {
		t.Fatal("precondition failed: token is not yet due for refresh")
	}
	if expired(tr) {
		t.Fatal("precondition failed: token has already expired")
	}

	c := &serviceAccountToken{send: offlineSend{}}
	got, err := c.GetServiceAccountToken(namespace, name, specOf(projectedTTL))
	if err != nil {
		t.Fatalf("GetServiceAccountToken: %v", err)
	}
	if got.Status.Token != "tok-stale" {
		t.Errorf("token = %q, want %q", got.Status.Token, "tok-stale")
	}
}

// An expired cached generation cannot be served, so the refresh failure
// surfaces instead.
func TestGetServiceAccountTokenRejectsExpiredWhenOffline(t *testing.T) {
	const name, namespace = "offline-expired", "default"
	cacheGeneration(t, name, namespace, 2*time.Hour, "tok-expired")

	c := &serviceAccountToken{send: offlineSend{}}
	if _, err := c.GetServiceAccountToken(namespace, name, specOf(projectedTTL)); err == nil {
		t.Fatal("expected an error for an expired cached token")
	}
}

// With nothing cached and no cloud, there is nothing to serve.
func TestGetServiceAccountTokenFailsWithNoCacheWhenOffline(t *testing.T) {
	c := &serviceAccountToken{send: offlineSend{}}
	if _, err := c.GetServiceAccountToken("default", "absent", specOf(projectedTTL)); err == nil {
		t.Fatal("expected an error when no token is cached")
	}
}

// A fresh cached token is served without consulting the cloud at all.
func TestGetServiceAccountTokenUsesFreshCache(t *testing.T) {
	const name, namespace = "fresh", "default"
	cacheGeneration(t, name, namespace, time.Minute, "tok-fresh")

	c := &serviceAccountToken{send: offlineSend{}}
	got, err := c.GetServiceAccountToken(namespace, name, specOf(projectedTTL))
	if err != nil {
		t.Fatalf("GetServiceAccountToken: %v", err)
	}
	if got.Status.Token != "tok-fresh" {
		t.Errorf("token = %q, want %q", got.Status.Token, "tok-fresh")
	}
}

// Presence in metaDB is not proof a token is still usable. An expired row
// survives until the GC sweep runs, and MetaServer has no other check when no
// public key is available.
func TestCheckTokenExistRejectsExpired(t *testing.T) {
	cacheGeneration(t, "auth-live", "default", time.Minute, "tok-auth-live")
	cacheGeneration(t, "auth-dead", "default", 2*time.Hour, "tok-auth-dead")

	if !CheckTokenExist("tok-auth-live") {
		t.Error("CheckTokenExist rejected a token that has not expired")
	}
	if CheckTokenExist("tok-auth-dead") {
		t.Error("CheckTokenExist accepted an expired token")
	}
}

func TestExpired(t *testing.T) {
	exp := projectedTTL
	cases := []struct {
		name string
		tr   *authenticationv1.TokenRequest
		want bool
	}{
		{
			name: "no expiration timestamp is treated as usable",
			tr:   &authenticationv1.TokenRequest{},
			want: false,
		},
		{
			name: "future expiry",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{ExpirationSeconds: &exp},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(time.Now().Add(time.Minute)),
				},
			},
			want: false,
		},
		{
			name: "past expiry",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{ExpirationSeconds: &exp},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(time.Now().Add(-time.Minute)),
				},
			},
			want: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := expired(c.tr); got != c.want {
				t.Errorf("expired() = %v, want %v", got, c.want)
			}
		})
	}
}
