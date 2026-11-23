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
	"strings"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

const logicalTokenKey = "logical-token-key"

var serviceAccountTestDBPath string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "kubeedge-serviceaccount-test-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create service account test directory: %v\n", err)
		os.Exit(1)
	}
	serviceAccountTestDBPath = filepath.Join(tempDir, "meta.db")

	exitCode := m.Run()
	if err := os.RemoveAll(tempDir); err != nil && exitCode == 0 {
		fmt.Fprintf(os.Stderr, "failed to remove service account test directory: %v\n", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}

func TestServiceAccountTokenKeyFunc(t *testing.T) {
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

	wantBase := fmt.Sprintf("%q/%q/%#v/%#v/%#v",
		"test-sa", "test-ns", request.Spec.Audiences, expirationSeconds, *request.Spec.BoundObjectRef)
	if got := baseKey("test-sa", "test-ns", request); got != wantBase {
		t.Fatalf("baseKey() = %q, want %q", got, wantBase)
	}
	if got := KeyFunc("test-sa", "test-ns", request); got != wantBase {
		t.Fatalf("KeyFunc() for request = %q, want legacy base key %q", got, wantBase)
	}

	first := request.DeepCopy()
	first.Status = authenticationv1.TokenRequestStatus{
		Token:               "first-secret-token",
		ExpirationTimestamp: metav1.NewTime(time.Unix(1_700_000_000, 123_456_789)),
	}
	wantFirst := fmt.Sprintf("%s/%d", wantBase, first.Status.ExpirationTimestamp.UnixNano())
	firstKey := KeyFunc("test-sa", "test-ns", first)
	if firstKey != wantFirst {
		t.Fatalf("KeyFunc() for first generation = %q, want %q", firstKey, wantFirst)
	}
	if strings.Contains(firstKey, first.Status.Token) {
		t.Fatal("KeyFunc() exposed token data in storage key")
	}
	if got := baseKey("test-sa", "test-ns", first); got != wantBase {
		t.Fatalf("baseKey() changed with token status: got %q, want %q", got, wantBase)
	}

	second := first.DeepCopy()
	second.Status.Token = "second-secret-token"
	second.Status.ExpirationTimestamp = metav1.NewTime(first.Status.ExpirationTimestamp.Add(time.Hour))
	secondKey := KeyFunc("test-sa", "test-ns", second)
	if secondKey == firstKey {
		t.Fatalf("different token generations produced the same key %q", firstKey)
	}
}

func TestNewestUnexpiredToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	key := logicalTokenKey
	entries := []models.Meta{
		{Key: key + "/invalid", Value: "not-json"},
		{Key: key, Value: marshalTokenRequest(t, "expired", now.Add(-time.Minute))},
		{Key: key + "/older", Value: marshalTokenRequest(t, "older", now.Add(time.Hour))},
		{Key: key + "/newest", Value: marshalTokenRequest(t, "newest", now.Add(2*time.Hour))},
		{Key: key + "-other/newest", Value: marshalTokenRequest(t, "other", now.Add(3*time.Hour))},
	}

	got, err := newestUnexpiredToken(key, entries, now)
	if err != nil {
		t.Fatalf("newestUnexpiredToken() returned error: %v", err)
	}
	if got.Status.Token != "newest" {
		t.Fatalf("newestUnexpiredToken() token = %q, want %q", got.Status.Token, "newest")
	}
}

func TestNewestUnexpiredTokenLegacyKey(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	key := logicalTokenKey
	entries := []models.Meta{
		{Key: key, Value: marshalTokenRequest(t, "legacy", now.Add(time.Hour))},
	}

	got, err := newestUnexpiredToken(key, entries, now)
	if err != nil {
		t.Fatalf("newestUnexpiredToken() returned error: %v", err)
	}
	if got.Status.Token != "legacy" {
		t.Fatalf("newestUnexpiredToken() token = %q, want %q", got.Status.Token, "legacy")
	}
}

func TestNewestUnexpiredTokenWithoutUsableEntry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	key := logicalTokenKey
	tests := []struct {
		name    string
		entries []models.Meta
	}{
		{name: "empty"},
		{
			name: "different logical key",
			entries: []models.Meta{
				{Key: key + "-other/generation", Value: marshalTokenRequest(t, "other", now.Add(time.Hour))},
			},
		},
		{
			name: "expired",
			entries: []models.Meta{
				{Key: key + "/generation", Value: marshalTokenRequest(t, "expired", now)},
			},
		},
		{
			name: "zero expiration",
			entries: []models.Meta{
				{Key: key + "/generation", Value: marshalTokenRequest(t, "zero", time.Time{})},
			},
		},
		{
			name: "invalid JSON",
			entries: []models.Meta{
				{Key: key + "/generation", Value: "not-json"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newestUnexpiredToken(key, test.entries, now); err == nil {
				t.Fatal("newestUnexpiredToken() returned no error")
			}
		})
	}
}

func TestServiceAccountTokenCacheLifecycle(t *testing.T) {
	dao.Init(serviceAccountTestDBPath, &edgecorev1alpha2.MetaManager{Enable: true})
	metaService := dbclient.NewMetaService()
	now := time.Now()
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
	for _, podUID := range []types.UID{"test-pod-uid", "refresh-pod-uid", "deleted-pod-uid"} {
		newServiceAccountToken(nil).DeleteServiceAccountToken(podUID)
	}

	older := request.DeepCopy()
	older.Status = authenticationv1.TokenRequestStatus{
		Token:               "older-token",
		ExpirationTimestamp: metav1.NewTime(now.Add(30 * time.Minute)),
	}
	newest := request.DeepCopy()
	newest.Status = authenticationv1.TokenRequestStatus{
		Token:               "newest-token",
		ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour)),
	}
	olderKey := insertCachedToken(t, metaService, "test-sa", "test-ns", older)
	newestKey := insertCachedToken(t, metaService, "test-sa", "test-ns", newest)

	got, err := getTokenLocally("test-sa", "test-ns", request)
	if err != nil {
		t.Fatalf("getTokenLocally() returned error: %v", err)
	}
	if got.Status.Token != newest.Status.Token {
		t.Fatalf("getTokenLocally() token = %q, want %q", got.Status.Token, newest.Status.Token)
	}
	for _, token := range []string{older.Status.Token, newest.Status.Token} {
		if !CheckTokenExist(token) {
			t.Errorf("CheckTokenExist(%q) = false, want true during generation overlap", token)
		}
	}

	refreshRequest := request.DeepCopy()
	refreshRequest.Spec.BoundObjectRef.UID = types.UID("refresh-pod-uid")
	refreshCandidate := refreshRequest.DeepCopy()
	refreshCandidate.Status = authenticationv1.TokenRequestStatus{
		Token:               "refresh-token",
		ExpirationTimestamp: metav1.NewTime(now.Add(5 * time.Minute)),
	}
	refreshKey := insertCachedToken(t, metaService, "test-sa", "test-ns", refreshCandidate)

	if _, err := getTokenLocally("test-sa", "test-ns", refreshRequest); err == nil || !strings.Contains(err.Error(), "token requires refresh") {
		t.Fatalf("getTokenLocally() error = %v, want token requires refresh", err)
	}
	rows, err := metaService.QueryMeta("key", refreshKey)
	if err != nil {
		t.Fatalf("failed to query refresh candidate: %v", err)
	}
	if rows == nil || len(*rows) != 1 {
		t.Fatalf("refresh candidate rows = %v, want one retained row", rows)
	}

	gcExpiredTokensOnce(metaService, now.Add(31*time.Minute))
	assertCachedTokenCount(t, metaService, olderKey, 0)
	assertCachedTokenCount(t, metaService, newestKey, 1)
	assertCachedTokenCount(t, metaService, refreshKey, 0)
	if CheckTokenExist(older.Status.Token) {
		t.Errorf("CheckTokenExist(%q) = true after old generation expired", older.Status.Token)
	}
	if !CheckTokenExist(newest.Status.Token) {
		t.Errorf("CheckTokenExist(%q) = false after old generation GC", newest.Status.Token)
	}

	deleteRequest := request.DeepCopy()
	deleteRequest.Spec.BoundObjectRef.UID = types.UID("deleted-pod-uid")
	deleteOlder := deleteRequest.DeepCopy()
	deleteOlder.Status = authenticationv1.TokenRequestStatus{
		Token:               "delete-older-token",
		ExpirationTimestamp: metav1.NewTime(now.Add(2 * time.Hour)),
	}
	deleteNewest := deleteRequest.DeepCopy()
	deleteNewest.Status = authenticationv1.TokenRequestStatus{
		Token:               "delete-newest-token",
		ExpirationTimestamp: metav1.NewTime(now.Add(3 * time.Hour)),
	}
	deleteOlderKey := insertCachedToken(t, metaService, "test-sa", "test-ns", deleteOlder)
	deleteNewestKey := insertCachedToken(t, metaService, "test-sa", "test-ns", deleteNewest)

	newServiceAccountToken(nil).DeleteServiceAccountToken(types.UID("deleted-pod-uid"))
	for _, key := range []string{deleteOlderKey, deleteNewestKey} {
		assertCachedTokenCount(t, metaService, key, 0)
	}
	for _, token := range []string{deleteOlder.Status.Token, deleteNewest.Status.Token} {
		if CheckTokenExist(token) {
			t.Errorf("CheckTokenExist(%q) = true after pod deletion", token)
		}
	}

	newServiceAccountToken(nil).DeleteServiceAccountToken(types.UID("test-pod-uid"))
}

func TestTokenExistsContinuesPastMalformedRows(t *testing.T) {
	metas := []string{
		"not-json",
		marshalTokenRequest(t, "other-token", time.Now().Add(time.Hour)),
		marshalTokenRequest(t, "target-token", time.Now().Add(time.Hour)),
	}

	if !tokenExists("target-token", metas) {
		t.Fatal("tokenExists() rejected a token stored after a malformed row")
	}
	if tokenExists("missing-token", metas) {
		t.Fatal("tokenExists() accepted a token not present in cache")
	}
	if tokenExists("", metas) {
		t.Fatal("tokenExists() accepted an empty token")
	}
}

func TestGCExpiredTokensOnce(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	metas := []models.Meta{
		{Key: "expired", Value: marshalTokenRequest(t, "expired", now.Add(-time.Second))},
		{Key: "expires-now", Value: marshalTokenRequest(t, "expires-now", now)},
		{Key: "fresh", Value: marshalTokenRequest(t, "fresh", now.Add(time.Second))},
		{Key: "zero", Value: marshalTokenRequest(t, "zero", time.Time{})},
		{Key: "invalid", Value: "not-json"},
	}
	metaStore := &fakeServiceAccountTokenMetaStore{metas: &metas}

	gcExpiredTokensOnce(metaStore, now)

	if got, want := strings.Join(metaStore.deletedKeys, ","), "expired,expires-now"; got != want {
		t.Fatalf("gcExpiredTokensOnce() deleted %q, want %q", got, want)
	}
	if metaStore.queryKey != "type" || metaStore.queryCondition != model.ResourceTypeServiceAccountToken {
		t.Fatalf("gcExpiredTokensOnce() queried %q=%q", metaStore.queryKey, metaStore.queryCondition)
	}
}

func TestGCExpiredTokensOnceContinuesAfterDeleteError(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	metas := []models.Meta{
		{Key: "first", Value: marshalTokenRequest(t, "first", now.Add(-time.Second))},
		{Key: "second", Value: marshalTokenRequest(t, "second", now.Add(-time.Second))},
	}
	metaStore := &fakeServiceAccountTokenMetaStore{
		metas:        &metas,
		deleteErrors: map[string]error{"first": fmt.Errorf("delete failed")},
	}

	gcExpiredTokensOnce(metaStore, now)

	if got, want := strings.Join(metaStore.deletedKeys, ","), "first,second"; got != want {
		t.Fatalf("gcExpiredTokensOnce() attempted deletes %q, want %q", got, want)
	}
}

func TestRunExpiredTokenGCStops(t *testing.T) {
	queried := make(chan struct{})
	metaStore := &fakeServiceAccountTokenMetaStore{querySignal: queried}
	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		runExpiredTokenGC(stopCh, time.Hour, metaStore)
		close(done)
	}()

	select {
	case <-queried:
	case <-time.After(time.Second):
		t.Fatal("runExpiredTokenGC() did not run initial collection")
	}
	close(stopCh)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runExpiredTokenGC() did not stop")
	}
	if metaStore.queryCalls != 1 {
		t.Fatalf("runExpiredTokenGC() query calls = %d, want 1", metaStore.queryCalls)
	}
}

func marshalTokenRequest(t *testing.T, token string, expiresAt time.Time) string {
	t.Helper()
	request := authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token:               token,
			ExpirationTimestamp: metav1.NewTime(expiresAt),
		},
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal token request: %v", err)
	}
	return string(data)
}

func insertCachedToken(t *testing.T, metaService *dbclient.MetaService, name, namespace string, request *authenticationv1.TokenRequest) string {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal cached token: %v", err)
	}
	key := KeyFunc(name, namespace, request)
	if err := metaService.InsertOrUpdate(&models.Meta{
		Key:   key,
		Type:  model.ResourceTypeServiceAccountToken,
		Value: string(data),
	}); err != nil {
		t.Fatalf("failed to insert cached token: %v", err)
	}
	return key
}

func assertCachedTokenCount(t *testing.T, metaService *dbclient.MetaService, key string, want int) {
	t.Helper()
	rows, err := metaService.QueryMeta("key", key)
	if err != nil {
		t.Fatalf("failed to query cached token %q: %v", key, err)
	}
	if rows == nil || len(*rows) != want {
		t.Fatalf("cached token %q count = %v, want %d", key, rows, want)
	}
}

type fakeServiceAccountTokenMetaStore struct {
	metas          *[]models.Meta
	queryKey       string
	queryCondition string
	queryCalls     int
	querySignal    chan struct{}
	deleteErrors   map[string]error
	deletedKeys    []string
}

func (f *fakeServiceAccountTokenMetaStore) QueryAllMeta(key, condition string) (*[]models.Meta, error) {
	f.queryKey = key
	f.queryCondition = condition
	f.queryCalls++
	if f.querySignal != nil {
		close(f.querySignal)
		f.querySignal = nil
	}
	return f.metas, nil
}

func (f *fakeServiceAccountTokenMetaStore) DeleteMetaByKey(key string) error {
	f.deletedKeys = append(f.deletedKeys, key)
	return f.deleteErrors[key]
}
