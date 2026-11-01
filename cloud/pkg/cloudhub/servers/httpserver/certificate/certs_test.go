package certificate

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
)

func TestVerifyCert(t *testing.T) {
	cahandler := certs.GetCAHandler(certs.CAHandlerTypeX509)
	pk, err := cahandler.GenPrivateKey()
	require.NoError(t, err)

	caPem, err := cahandler.NewSelfSigned(pk)
	require.NoError(t, err)

	certshandler := certs.GetHandler(certs.HandlerTypeX509)
	csrPem, err := certshandler.CreateCSR(pkix.Name{
		Country:      []string{"CN"},
		Organization: []string{"system:nodes"},
		Locality:     []string{"Hangzhou"},
		Province:     []string{"Zhejiang"},
		CommonName:   "system:node:testnode",
	}, pk, nil)
	require.NoError(t, err)

	certPrm, err := certshandler.SignCerts(certs.SignCertsOptionsWithCSR(
		csrPem.Bytes,
		caPem.Bytes,
		pk.DER(),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		time.Hour,
	))
	require.NoError(t, err)

	hubconfig.Config.Ca = caPem.Bytes
	certs, err := x509.ParseCertificate(certPrm.Bytes)
	require.NoError(t, err)

	err = verifyCert(certs, "testnode")
	require.NoError(t, err)
}

func TestVerifyCert_ExpiredCertificate(t *testing.T) {
	cahandler := certs.GetCAHandler(certs.CAHandlerTypeX509)
	pk, err := cahandler.GenPrivateKey()
	require.NoError(t, err)

	caPem, err := cahandler.NewSelfSigned(pk)
	require.NoError(t, err)

	certshandler := certs.GetHandler(certs.HandlerTypeX509)
	csrPem, err := certshandler.CreateCSR(pkix.Name{
		Country:      []string{"CN"},
		Organization: []string{"system:nodes"},
		Locality:     []string{"Hangzhou"},
		Province:     []string{"Zhejiang"},
		CommonName:   "system:node:testnode",
	}, pk, nil)
	require.NoError(t, err)

	// Create an expired certificate by using negative duration
	certPrm, err := certshandler.SignCerts(certs.SignCertsOptionsWithCSR(
		csrPem.Bytes,
		caPem.Bytes,
		pk.DER(),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		-time.Hour, // Negative duration to create expired certificate
	))
	require.NoError(t, err)

	hubconfig.Config.Ca = caPem.Bytes
	certs, err := x509.ParseCertificate(certPrm.Bytes)
	require.NoError(t, err)

	err = verifyCert(certs, "testnode")
	require.Error(t, err, "Expected error for expired certificate")
}

func TestVerifyAuthorization(t *testing.T) {
	const cakey = `MHcCAQEEIJQgy45Hw91mXm3pRXwxwDg4BgR4DY1UvHlzm/JXr9K6oAoGCCqGSM49AwEHoUQDQgAEq4Rd11aJ/FXEYBE2YCUMjRZVpqytxDBq2anuzokPculGaTrSDiRy1IKukPhlg34bq7J6wqkF0cmFUvcTjtReqw==`
	cakeyDer, err := base64.StdEncoding.DecodeString(cakey)
	require.NoError(t, err)
	hubconfig.Config.CaKey = cakeyDer

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
	})
	expiredToken, err := token.SignedString(cakeyDer)
	require.NoError(t, err)

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Minute)),
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
			name:          "not split token",
			token:         "xxxx",
			wantCode:      http.StatusUnauthorized,
			containsError: "token validation failure, token cannot be split",
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
			containsError: "token validation failure, err: token has invalid claims: token is expire",
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