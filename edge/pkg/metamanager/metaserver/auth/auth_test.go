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

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/client-go/tools/cache"
)

// buildTestToken constructs a syntactically valid JWT-like token
// (header.payload.signature) whose payload contains the given issuer.
// The token is NOT cryptographically signed; it is only used to test
// parsing logic that does not verify signatures.
func buildTestToken(issuer string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]interface{}{"iss": issuer})
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encodedPayload + ".fakesig"
}

// TestHasCorrectIssuer tests the private hasCorrectIssuer method via the
// same package (white-box test).
func TestHasCorrectIssuer(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	authenticatorObj := JWTTokenAuthenticator[privateClaims](
		indexer,
		[]string{"https://valid-issuer.example.com"},
		nil,
		nil,
		nil,
	)
	j := authenticatorObj.(*jwtTokenAuthenticator[privateClaims])

	tests := []struct {
		name      string
		token     string
		wantMatch bool
	}{
		{
			name:      "matching issuer",
			token:     buildTestToken("https://valid-issuer.example.com"),
			wantMatch: true,
		},
		{
			name:      "non-matching issuer",
			token:     buildTestToken("https://other-issuer.example.com"),
			wantMatch: false,
		},
		{
			name:      "empty issuer",
			token:     buildTestToken(""),
			wantMatch: false,
		},
		{
			name:      "token with wrong number of parts",
			token:     "onlyone",
			wantMatch: false,
		},
		{
			name:      "token with invalid base64 payload",
			token:     "header.!!!invalid!!!.sig",
			wantMatch: false,
		},
		{
			name:      "token with non-JSON payload",
			token:     "header." + base64.RawURLEncoding.EncodeToString([]byte("not-json")) + ".sig",
			wantMatch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := j.hasCorrectIssuer(tt.token)
			if got != tt.wantMatch {
				t.Errorf("hasCorrectIssuer(%q) = %v, want %v", tt.token, got, tt.wantMatch)
			}
		})
	}
}

// TestJWTTokenAuthenticatorWrongIssuer verifies that AuthenticateToken returns
// (nil, false, nil) for a token whose issuer is not in the allowed set.
func TestJWTTokenAuthenticatorWrongIssuer(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	auth := JWTTokenAuthenticator[privateClaims](
		indexer,
		[]string{"https://allowed.example.com"},
		nil,
		authenticator.Audiences{"aud"},
		nil,
	)

	token := buildTestToken("https://not-allowed.example.com")
	resp, ok, err := auth.AuthenticateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for wrong issuer")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}
}

// TestJWTTokenAuthenticatorMalformedTokenStructuralFastPath verifies that
// AuthenticateToken returns (nil, false, nil) for a token whose structure is
// not a valid three-segment JWT (header.payload.signature).  The token
// "not.a.real.jwt.token" splits into five segments; hasCorrectIssuer rejects
// it immediately at the len(parts) != 3 guard — before any base64 decoding,
// issuer comparison, jose parsing, or database access.
//
// Note: this test exercises the malformed-structure fast path in
// hasCorrectIssuer, not the issuer-allowlist filter.  The wrong-issuer case
// (a structurally valid three-part token whose issuer is not in the allowed
// set) is covered by TestJWTTokenAuthenticatorWrongIssuer.
func TestJWTTokenAuthenticatorMalformedTokenStructuralFastPath(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	auth := JWTTokenAuthenticator[privateClaims](
		indexer,
		[]string{"https://valid-issuer.example.com"},
		nil,
		authenticator.Audiences{"aud"},
		nil,
	)

	// "not.a.real.jwt.token" has five dot-separated segments, so
	// hasCorrectIssuer returns false at the len(parts) != 3 check before
	// reaching any issuer comparison, jose parsing, or GORM access.
	resp, ok, err := auth.AuthenticateToken(context.Background(), "not.a.real.jwt.token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false for token with wrong number of parts")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}
}

// TestJWTTokenAuthenticatorMultipleIssuers ensures that any of the configured
// issuers is accepted by the hasCorrectIssuer helper.  This test calls
// hasCorrectIssuer directly and does not exercise the full AuthenticateToken
// path.
func TestJWTTokenAuthenticatorMultipleIssuers(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	auth := JWTTokenAuthenticator[privateClaims](
		indexer,
		[]string{"https://issuer-a.example.com", "https://issuer-b.example.com"},
		nil,
		nil,
		nil,
	)
	j := auth.(*jwtTokenAuthenticator[privateClaims])

	if !j.hasCorrectIssuer(buildTestToken("https://issuer-a.example.com")) {
		t.Error("issuer-a should be accepted")
	}
	if !j.hasCorrectIssuer(buildTestToken("https://issuer-b.example.com")) {
		t.Error("issuer-b should be accepted")
	}
	if j.hasCorrectIssuer(buildTestToken("https://issuer-c.example.com")) {
		t.Error("issuer-c should not be accepted")
	}
}

// TestJWTTokenAuthenticatorEmptyIssuers verifies that when no issuers are
// configured, no token passes the hasCorrectIssuer check.  This test calls
// hasCorrectIssuer directly and does not exercise the full AuthenticateToken
// path.
func TestJWTTokenAuthenticatorEmptyIssuers(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	auth := JWTTokenAuthenticator[privateClaims](
		indexer,
		[]string{},
		nil,
		nil,
		nil,
	)
	j := auth.(*jwtTokenAuthenticator[privateClaims])

	if j.hasCorrectIssuer(buildTestToken("https://any-issuer.example.com")) {
		t.Error("no issuer should be accepted when the allowed set is empty")
	}
}
