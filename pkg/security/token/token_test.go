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
package token

import (
	"encoding/pem"
	"strings"
	"testing"
)

const (
	testCA = `-----BEGIN CERTIFICATE-----
MIIDEDCCAfigAwIBAgIIHmr3g3dw7rYwDQYJKoZIhvcNAQELBQAwJjEOMAwGA1UE
BhMFQ2hpbmExCTAHBgNVBAoTADEJMAcGA1UECxMAMB4XDTI0MDQwODA5NTY1MloX
DTM0MDQwNjA5NTY1MlowJjEOMAwGA1UEBhMFQ2hpbmExCTAHBgNVBAoTADEJMAcG
A1UECxMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAm2td7Yn3tTv0
g1d6MxQBqESl/flEvt7G1gFWoXHHzSN9+jh75Y1meHkuLu6LeYYuQMdFiHzra/jM
mN78RJToOW96yH97x9F+YstCStKdMh3D04vmiXqwdkzIFXvbcFol1mXP8r72R8z+
odjPr/EwDNI0KSzTtZfoKIalwCDzqX+WPOgRKaCyTHs01dNHSQhdyhG9oTdeDtIL
e6HNqxA966jMF6p/giHSUrcec41XxxZPfHZ5sppaSIMxabBS/M/lMlav2ZMfr6+y
szP33/CRnbn45d767wyH9P0kbWrdU9IPN9vGD7QKfNfcoN2FLHgkkoXOJl/AXJfF
BftXWs0qoQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUwAwEB
/zAdBgNVHQ4EFgQUegaSgp7zhR9AwLcVBKjraccqbkMwDQYJKoZIhvcNAQELBQAD
ggEBAI/I2Ln//zxUhMY9JwM57sDDQ7Vimc+uWSgrtqhiGOGMzhBFREr1dS5UE1a2
dMMh566lBuQAT7hyOC9EqL+zbHAcGZGUyIqByIKv9W2HMNnTOGZ3XbPJNV6DH/wX
66Jv9dvNf+EVj0PhJvRmn6QslbVrOmAtmylllTXDJnoULX2+ZAgHNS2+p3rnXCas
Nh52RfmjaH1sH7e1zvVKvOpTCKbuArzjspSdJ1ssnWYnrLtkAvz7PZEDL88fFmre
uhDkSogDJI/yC8m+6lnvYdLWuDAkVREP39XZ/7KtJFLEeBikRhsRK2BOnVidPFDM
rFqlS7gD0cPmIEo2wgkh3pKaxNE=
-----END CERTIFICATE-----`

	testCAKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAm2td7Yn3tTv0g1d6MxQBqESl/flEvt7G1gFWoXHHzSN9+jh7
5Y1meHkuLu6LeYYuQMdFiHzra/jMmN78RJToOW96yH97x9F+YstCStKdMh3D04vm
iXqwdkzIFXvbcFol1mXP8r72R8z+odjPr/EwDNI0KSzTtZfoKIalwCDzqX+WPOgR
KaCyTHs01dNHSQhdyhG9oTdeDtILe6HNqxA966jMF6p/giHSUrcec41XxxZPfHZ5
sppaSIMxabBS/M/lMlav2ZMfr6+yszP33/CRnbn45d767wyH9P0kbWrdU9IPN9vG
D7QKfNfcoN2FLHgkkoXOJl/AXJfFBftXWs0qoQIDAQABAoIBAQCUnP8M48+cWj89
1EkCTJAlIbeD+nY0+XsyKcd3yv/d9aFBwf8fCq3AZ0e1Et8FjjvuL14a3DCVZyvk
xdx9i9HfEe1biSOId5cdyvSR7YDo6jNVtsH0FgBkrpjoii3T6i+iKmrE2LtQ/wFB
K7u0prFmoR3FfZrXWvFgxxf5dsjn+p3nJQdDZbWjAcZJdf7T78EtnbQ4uzCAQRTW
hsmfI8OPTzyf/FLTkscjNzP6GVMWP9x017TfKgucqwPt8FeKqc9Si+fEZ3GDQiSi
KZHSXGIjO3MxCPDX4XW4sfkP5+iB8OEZHUHN5SHBAnTu2sR5hnwnPj2y1jcL7T3s
Fv3vnTWRAoGBAMW0+L7PZ3foEVi8IL9pkB24+3u6CIIrc98a7YKbLfWmDSmBQis/
5BRSvuf1l99EZDbVHE6xkhecSOP8cmIrZ1NjfUgXloaDkHg4hLgtQ+ZIPnSOHifP
M+DO6re+yCkngxgklofUSxX4STviPJiewZayVTwWjpbK74fv8KrraooNAoGBAMk+
goOH8aUf2QeyCHrROQAb8QXF3hRbyimFES/7eSdlFIy/bROnT9cdcSviyzrvBzwY
3SIEcp7g3ZbJb+dFaUObE/nY6EJP+moCCOurBKjzJeRQ1zlFzb08c4lCAU7/l28z
iHSQJavQof/mvcR5Pi81h5VFuFeFgchBYoHChCHlAoGBAJ9Om8DkzrLHxHKD5L9Y
CFBq5flkhcadzNhRkmBTOk1eZ+yxwuemq9nUcw/lzWKScU3dmtmuK9HqlLFgkaqY
3sFKwYB9wUTSbm7w28CseLHuNKUmfxYE2AClumwkxpSiyfeCQ+lfHsGtNxWRztIL
2mHbgOLSKkNHcotOw9Z1q3thAoGAGA/dUxTCE9hG/uCOmwDBK/4rR2FtOEnxVh2O
/Im45rjzSBDrXdo3daUTjwfC/PzvhIQEjLiza8O/OvRC6QgnmenE7a69tpARhPNR
VbxRBlJsSWxRD4wFGYdM2TCHL4bn+GfU/PrvRiff9tUEA6XrhYGFAJghfnV8GxGW
UaWMXvECgYEAlhQUBLZ/ZTPXQWb7VA5Mur/s0ptrs4CAP+KBc9xyBfKrPEmjMcxJ
2788MXYbnGzwpY0Nk/ruUNJ+PFPy4GgGWnZp7Fqi5qGIrFkuazQaulRQJwTZkIzJ
x0KJ+pjtT+89L1r7murZAJcPL+TyRYeg295NTfcjAfSdcBfIjzURYVg=
-----END RSA PRIVATE KEY-----`

	testAltCA = `-----BEGIN CERTIFICATE-----
MIIDEDCCAfigAwIBAgIIMmFz7zU7iyQwDQYJKoZIhvcNAQELBQAwJjEOMAwGA1UE
BhMFQ2hpbmExCTAHBgNVBAoTADEJMAcGA1UECxMAMB4XDTI0MDQwODEwMDczN1oX
DTM0MDQwNjEwMDczN1owJjEOMAwGA1UEBhMFQ2hpbmExCTAHBgNVBAoTADEJMAcG
A1UECxMAMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA5LnmwXmba7lH
mllpyNQNxr3P5VDz98TGS9G9qRnGY9HZSRg4OZTGTM4ueJoqY8WpfU7KpBIQbxlJ
RHa2wpUCb0Kw39WpLEDZRnl14AFBCUQin85YZa4LOkj9EcQzQrTnNUbGFt45Y9WX
cg+2ouOBrFM7dEwgBNZ+KVZA19j5qG2sWVmX+V9iucA6ZHMFl5q7MyPQY/MJDisj
+YKe6zRMpawJ/VfR3huYLplFKtPHtlgqQFmNpIt9FKz0CZTtOJwJ/NjO5aoLCZ9d
/F5nZ0jIAtqmm8CmHuDx6z15fNgHs0/+Z+7CtbHGx0GWx2zFUj/UWQ9VwJ/nQIGO
eKfZ6kW0cQIDAQABo0IwQDAOBgNVHQ8BAf8EBAMCAqQwDwYDVR0TAQH/BAUwAwEB
/zAdBgNVHQ4EFgQUR+IUW6h9DNK4YX9l65GQK7bxVFUwDQYJKoZIhvcNAQELBQAD
ggEBAKpG+hGmNb4TP2y/aFJbD9g8GYdWsXpNvRTp9SixZUz9xrn6r9GbsiyLmEcW
Wjn3wZfYQXTR5cCsIQO4T5K3H7Z0e6z+IQtTqxq0Jch3kBaGlBcDGGm6FhPu/3qV
zLY6XrTR5chgPnCgbAXUbPOD7xWvJYGZ3sIFUNZcEi/XsXyHs6I6qxcYfXhNnG/3
7kX4TKDFuQrlKV5oaJZ3/Chs7x8EzQ82XQJrjTJJL9Q+nPJMvHx1M/uHzTpXVbRL
GIdbAKp9HzWuWZ0GZo2+3rLdKQkYHQXEcUQvPxyYqXA6Y5g6MvdwrC/AUr58n5kp
QEJCjtBMVHWyXDLk4KliPHpfq64=
-----END CERTIFICATE-----`
)

func TestToken(t *testing.T) {
	var token string

	caBlock, _ := pem.Decode([]byte(testCA))
	if caBlock == nil {
		t.Fatal("failed to decode CA certificate PEM")
	}

	caKeyBlock, _ := pem.Decode([]byte(testCAKey))
	if caKeyBlock == nil {
		t.Fatal("failed to decode CA key PEM")
	}

	t.Run("test Create", func(t *testing.T) {
		var err error
		token, err = Create(caBlock.Bytes, caKeyBlock.Bytes, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(token) == 0 {
			t.Fatal("failed to get token")
		}

		parts := strings.Split(token, ".")
		if len(parts) != 4 {
			t.Fatalf("token should have 4 parts, got %d", len(parts))
		}

		caHash := hashCA(caBlock.Bytes)
		if parts[0] != caHash {
			t.Fatalf("first part should be CA hash, expected %s, got %s", caHash, parts[0])
		}
	})

	t.Run("test VerifyCAAndGetRealToken", func(t *testing.T) {
		var err error
		jwtToken, err := VerifyCAAndGetRealToken(token, caBlock.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(jwtToken) == 0 {
			t.Fatal("failed to get JWT token")
		}

		parts := strings.Split(jwtToken, ".")
		if len(parts) != 3 {
			t.Fatalf("JWT token should have 3 parts, got %d", len(parts))
		}
	})

	t.Run("test Verify", func(t *testing.T) {
		jwtToken, err := VerifyCAAndGetRealToken(token, caBlock.Bytes)
		if err != nil {
			t.Fatal(err)
		}

		b, err := Verify(jwtToken, caKeyBlock.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if !b {
			t.Fatalf("invalid token %s", jwtToken)
		}
	})

	t.Run("test Create with invalid interval", func(t *testing.T) {
		_, err := Create(caBlock.Bytes, caKeyBlock.Bytes, 0)
		if err != nil {
			t.Fatal("Create should work with 0 interval, but got error:", err)
		}

		_, err = Create(caBlock.Bytes, caKeyBlock.Bytes, -1)
		if err != nil {
			t.Fatal("Create should work with negative interval, but got error:", err)
		}
	})

	t.Run("test VerifyCAAndGetRealToken with invalid token format", func(t *testing.T) {
		invalidTokenFormats := []string{
			"invalid",
			"single.part",
			"too.many.parts.1.2.3",
			"",
		}

		for _, invalidToken := range invalidTokenFormats {
			_, err := VerifyCAAndGetRealToken(invalidToken, caBlock.Bytes)
			if err == nil {
				t.Fatalf("VerifyCAAndGetRealToken should fail with invalid token format: %s", invalidToken)
			}
		}
	})

	t.Run("test VerifyCAAndGetRealToken with mismatched CA", func(t *testing.T) {
		// Decode the alternative CA
		altCABlock, _ := pem.Decode([]byte(testAltCA))
		if altCABlock == nil {
			t.Fatal("failed to decode alternative CA certificate PEM")
		}

		// The token was created with testCA, verify with altCA (should fail)
		_, err := VerifyCAAndGetRealToken(token, altCABlock.Bytes)
		if err == nil {
			t.Fatal("VerifyCAAndGetRealToken should fail with mismatched CA")
		}
	})

	t.Run("test Verify with invalid token", func(t *testing.T) {
		invalidTokens := []string{
			"invalid",                          // Completely invalid
			"header.payload.invalid_signature", // Invalid JWT format
		}

		for _, invalidToken := range invalidTokens {
			_, err := Verify(invalidToken, caKeyBlock.Bytes)
			if err == nil {
				t.Fatalf("Verify should fail with invalid token: %s", invalidToken)
			}
		}
	})
}
