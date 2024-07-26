package certificate

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/require"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

func TestVerifyCert(t *testing.T) {
	capem, _ := pem.Decode([]byte(ca))
	hubconfig.Config.Ca = capem.Bytes

	certpem, _ := pem.Decode([]byte(cert))
	certs, err := x509.ParseCertificate(certpem.Bytes)
	require.NoError(t, err)

	err = verifyCert(certs, "hw-test1ht6hcsru")
	require.NoError(t, err)
}

func TestVerifyAuthorization(t *testing.T) {
	cakeyDer, err := base64.StdEncoding.DecodeString(cakey)
	require.NoError(t, err)
	hubconfig.Config.CaKey = cakeyDer

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: time.Now().Add(-1 * time.Minute).Unix(),
	})
	expiredToken, err := token.SignedString(cakeyDer)
	require.NoError(t, err)

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
	})
	passedToken, err := token.SignedString(cakeyDer)
	require.NoError(t, err)

	cases := []struct {
		name          string
		token         string
		wantCode      int
		containsError string
	}{
		{
			name:          "token empty",
			token:         "",
			wantCode:      http.StatusUnauthorized,
			containsError: "token validation failure, token is empty",
		},
		{
			name:          "not splited token",
			token:         "xxxx",
			wantCode:      http.StatusUnauthorized,
			containsError: "token validation failure, token cannot be splited",
		},
		{
			name:          "invalid token",
			token:         "Bearer xxxx",
			wantCode:      http.StatusUnauthorized,
			containsError: "token validation failure, err:",
		},
		{
			name:          "expired token",
			token:         "Bearer " + expiredToken,
			wantCode:      http.StatusUnauthorized,
			containsError: "token validation failure, err: Token is expired",
		},
		{
			name:     "passed token",
			token:    "Bearer " + passedToken,
			wantCode: http.StatusOK,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, err := verifyAuthorization(c.token)
			require.Equal(t, c.wantCode, code)
			if c.containsError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, c.containsError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

const (
	ca = `-----BEGIN CERTIFICATE-----
MIIBejCCAR+gAwIBAgICBAAwCgYIKoZIzj0EAwIwEzERMA8GA1UEAxMIS3ViZUVk
Z2UwIBcNMjQwNTA5MDczNzU2WhgPMjEyNDAxMDYwNzM3NTZaMBMxETAPBgNVBAMT
CEt1YmVFZGdlMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEq4Rd11aJ/FXEYBE2
YCUMjRZVpqytxDBq2anuzokPculGaTrSDiRy1IKukPhlg34bq7J6wqkF0cmFUvcT
jtReq6NhMF8wDgYDVR0PAQH/BAQDAgKkMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggr
BgEFBQcDAjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTvK3f704DC7OOiVbmO
PyKwJAUwQjAKBggqhkjOPQQDAgNJADBGAiEAkOgvZtFy+aYSqsfxIVMXxScsGilA
P1Iiy/r5PerqODcCIQCH+qeEuxIgZzUAD/Wm6xameEmyn/mO4su/4UE6APNZFQ==
-----END CERTIFICATE-----`

	cakey = `MHcCAQEEIJQgy45Hw91mXm3pRXwxwDg4BgR4DY1UvHlzm/JXr9K6oAoGCCqGSM49AwEHoUQDQgAEq4Rd11aJ/FXEYBE2YCUMjRZVpqytxDBq2anuzokPculGaTrSDiRy1IKukPhlg34bq7J6wqkF0cmFUvcTjtReqw==`

	cert = `-----BEGIN CERTIFICATE-----
MIIBjjCCATWgAwIBAgIIWugtLvecOyUwCgYIKoZIzj0EAwIwEzERMA8GA1UEAxMI
S3ViZUVkZ2UwHhcNMjQwNTA5MDc0MjQ4WhcNMjUwNTA5MDc0MjQ4WjA+MRUwEwYD
VQQKEwxzeXN0ZW06bm9kZXMxJTAjBgNVBAMTHHN5c3RlbTpub2RlOmh3LXRlc3Qx
aHQ2aGNzcnUwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARH0RepNVRX5U5lXcTJ
XJM/3PgoaDbxEeiS4RZq2Vz86fESc1KmFL+dUyvaqQ4BBIq3FOfirkwUYuhtiSXr
UyiRo0gwRjAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYIKwYBBQUHAwIwHwYD
VR0jBBgwFoAU7yt3+9OAwuzjolW5jj8isCQFMEIwCgYIKoZIzj0EAwIDRwAwRAIg
fswApb3FlJZXRw5aSIvls+uqR1ryfczy4fuzL/Y2i4MCIFyyV0t9Ts9uMHHx8R2+
6oFBzFcJvH65edh9/eH8rUy8
-----END CERTIFICATE-----`

	certKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIJOhw4Hxmkq6E9YTwj/y+L+bl7nUgxNKcz1dcgiPlm4WoAoGCCqGSM49
AwEHoUQDQgAER9EXqTVUV+VOZV3EyVyTP9z4KGg28RHokuEWatlc/OnxEnNSphS/
nVMr2qkOAQSKtxTn4q5MFGLobYkl61MokQ==
-----END EC PRIVATE KEY-----`
)
