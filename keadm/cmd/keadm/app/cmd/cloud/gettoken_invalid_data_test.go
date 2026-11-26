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

package cloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

// expectedSecretPath is the exact Kubernetes API path that queryToken must use.
const expectedSecretPath = "/api/v1/namespaces/kubeedge/secrets/tokensecret"

// writeTestKubeconfig creates a minimal kubeconfig file pointing at the given
// server URL and returns its path. The file is placed in a test-scoped
// temporary directory so cleanup is automatic.
func writeTestKubeconfig(t *testing.T, serverURL string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")
	content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: ` + serverURL + `
    insecure-skip-tls-verify: true
  name: test
contexts:
- context:
    cluster: test
    user: test
  name: test
current-context: test
users:
- name: test
  user:
    token: unused
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}
	return path
}

// newTokenTestServer creates an httptest.Server that returns a pre-marshalled
// JSON payload for the expected Secret GET path. The response is marshalled
// before the server starts so that t.Fatalf is called from the test goroutine,
// not the HTTP handler goroutine.
func newTokenTestServer(
	t *testing.T,
	statusCode int,
	response any,
) *httptest.Server {
	t.Helper()

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal test response: %v", err)
	}

	return httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf(
					"unexpected HTTP method: got %s, want %s",
					r.Method,
					http.MethodGet,
				)
			}

			if r.URL.Path != expectedSecretPath {
				t.Errorf(
					"unexpected request path: got %s, want %s",
					r.URL.Path,
					expectedSecretPath,
				)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)

			if _, err := w.Write(payload); err != nil {
				t.Errorf("failed to write test response: %v", err)
			}
		},
	))
}

func TestQueryTokenReturnsNonEmptyTokenData(t *testing.T) {
	wantToken := []byte("valid-token-bytes-1234567890")

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.TokenSecretName,
			Namespace: "kubeedge",
		},
		Data: map[string][]byte{
			common.TokenDataName: wantToken,
		},
	}

	srv := newTokenTestServer(t, http.StatusOK, secret)
	defer srv.Close()

	kubeconfig := writeTestKubeconfig(t, srv.URL)
	got, err := queryToken("kubeedge", common.TokenSecretName, kubeconfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(wantToken) {
		t.Errorf("token mismatch: got %d bytes, want %d bytes", len(got), len(wantToken))
	}
}

func TestQueryTokenRejectsSecretWithoutTokenData(t *testing.T) {
	// Secret data map lacks the required "tokendata" key entirely.
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.TokenSecretName,
			Namespace: "kubeedge",
		},
		Data: map[string][]byte{
			"other-key": []byte("irrelevant"),
		},
	}

	srv := newTokenTestServer(t, http.StatusOK, secret)
	defer srv.Close()

	kubeconfig := writeTestKubeconfig(t, srv.URL)
	got, err := queryToken("kubeedge", common.TokenSecretName, kubeconfig)
	if err == nil {
		t.Fatal("expected invalid token-data error, got nil")
	}
	for _, expected := range []string{
		"kubeedge",
		common.TokenSecretName,
		common.TokenDataName,
		"non-empty",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error %q does not contain %q", err, expected)
		}
	}
	if len(got) != 0 {
		t.Errorf("expected no token on error, got %d bytes", len(got))
	}
}

func TestQueryTokenRejectsEmptyTokenData(t *testing.T) {
	// Secret contains the "tokendata" key but with zero-length bytes.
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.TokenSecretName,
			Namespace: "kubeedge",
		},
		Data: map[string][]byte{
			common.TokenDataName: {},
		},
	}

	srv := newTokenTestServer(t, http.StatusOK, secret)
	defer srv.Close()

	kubeconfig := writeTestKubeconfig(t, srv.URL)
	got, err := queryToken("kubeedge", common.TokenSecretName, kubeconfig)
	if err == nil {
		t.Fatal("expected invalid token-data error, got nil")
	}
	for _, expected := range []string{
		"kubeedge",
		common.TokenSecretName,
		common.TokenDataName,
		"non-empty",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error %q does not contain %q", err, expected)
		}
	}
	if len(got) != 0 {
		t.Errorf("expected no token on error, got %d bytes", len(got))
	}
}

func TestQueryTokenPreservesSecretAPIError(t *testing.T) {
	// Server returns a Kubernetes API 404 Status response.
	apiStatus := &metav1.Status{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Status",
		},
		Status:  "Failure",
		Message: "secrets \"tokensecret\" not found",
		Reason:  metav1.StatusReasonNotFound,
		Code:    http.StatusNotFound,
	}

	srv := newTokenTestServer(t, http.StatusNotFound, apiStatus)
	defer srv.Close()

	kubeconfig := writeTestKubeconfig(t, srv.URL)
	got, err := queryToken("kubeedge", common.TokenSecretName, kubeconfig)
	if err == nil {
		t.Fatal("expected API error, got nil")
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected Kubernetes NotFound error, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no token on API error, got %d bytes", len(got))
	}
}
