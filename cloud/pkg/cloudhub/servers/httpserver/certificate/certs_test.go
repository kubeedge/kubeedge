package certificate

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/common/types"
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

func TestEdgeCoreClientCertCSRSubject(t *testing.T) {
	cahandler := certs.GetCAHandler(certs.CAHandlerTypeX509)
	caKey, err := cahandler.GenPrivateKey()
	require.NoError(t, err)
	caPem, err := cahandler.NewSelfSigned(caKey)
	require.NoError(t, err)

	origConfig := hubconfig.Config
	t.Cleanup(func() { hubconfig.Config = origConfig })
	hubconfig.Config.Ca = caPem.Bytes
	hubconfig.Config.CaKey = caKey.DER()
	hubconfig.Config.CloudHub.EdgeCertSigningDuration = 1

	certshandler := certs.GetHandler(certs.HandlerTypeX509)
	nodeKey, err := certshandler.GenPrivateKey()
	require.NoError(t, err)
	newCSR := func(sub pkix.Name) []byte {
		csrPem, err := certshandler.CreateCSR(sub, nodeKey, nil)
		require.NoError(t, err)
		return csrPem.Bytes
	}
	nodeSubject := func(nodeName string) pkix.Name {
		return pkix.Name{
			Organization: []string{"system:nodes"},
			CommonName:   "system:node:" + nodeName,
		}
	}

	// the certificate node-a already holds, used for certificate rotation
	nodeACertPem, err := certshandler.SignCerts(certs.SignCertsOptionsWithCSR(
		newCSR(nodeSubject("node-a")), caPem.Bytes, caKey.DER(),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, time.Hour))
	require.NoError(t, err)
	nodeACert, err := x509.ParseCertificate(nodeACertPem.Bytes)
	require.NoError(t, err)

	bootstrapToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}).SignedString(caKey.DER())
	require.NoError(t, err)

	cases := []struct {
		name     string
		peerCert *x509.Certificate
		token    string
		csr      []byte
		wantCode int
	}{
		{
			name:     "rotation with csr for own node",
			peerCert: nodeACert,
			csr:      newCSR(nodeSubject("node-a")),
			wantCode: http.StatusOK,
		},
		{
			name:     "rotation with csr for another node",
			peerCert: nodeACert,
			csr:      newCSR(nodeSubject("node-b")),
			wantCode: http.StatusForbidden,
		},
		{
			name:     "rotation with csr for another organization",
			peerCert: nodeACert,
			csr: newCSR(pkix.Name{
				Organization: []string{"system:nodes", "system:masters"},
				CommonName:   "system:node:node-a",
			}),
			wantCode: http.StatusForbidden,
		},
		{
			name:     "token with csr for the requested node",
			token:    "Bearer " + bootstrapToken,
			csr:      newCSR(nodeSubject("node-a")),
			wantCode: http.StatusOK,
		},
		{
			name:     "token with csr for another node",
			token:    "Bearer " + bootstrapToken,
			csr:      newCSR(nodeSubject("node-b")),
			wantCode: http.StatusForbidden,
		},
		{
			name:     "token with csr without node organization",
			token:    "Bearer " + bootstrapToken,
			csr:      newCSR(pkix.Name{CommonName: "system:node:node-a"}),
			wantCode: http.StatusForbidden,
		},
		{
			name:     "invalid csr",
			token:    "Bearer " + bootstrapToken,
			csr:      []byte("not a csr"),
			wantCode: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			httpReq := httptest.NewRequest(http.MethodGet, "/edge.crt", bytes.NewReader(c.csr))
			httpReq.Header.Set(types.HeaderNodeName, "node-a")
			if c.peerCert != nil {
				httpReq.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{c.peerCert}}
			} else {
				httpReq.TLS = &tls.ConnectionState{}
				httpReq.Header.Set(types.HeaderAuthorization, c.token)
			}
			recorder := httptest.NewRecorder()

			EdgeCoreClientCert(restful.NewRequest(httpReq), restful.NewResponse(recorder))

			require.Equal(t, c.wantCode, recorder.Code)
			if c.wantCode != http.StatusOK {
				return
			}
			signed, err := x509.ParseCertificate(recorder.Body.Bytes())
			require.NoError(t, err)
			require.Equal(t, "system:node:node-a", signed.Subject.CommonName)
			require.Equal(t, []string{"system:nodes"}, signed.Subject.Organization)
		})
	}
}
