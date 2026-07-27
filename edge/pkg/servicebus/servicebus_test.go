package servicebus

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	servicebusConfig "github.com/kubeedge/kubeedge/edge/pkg/servicebus/config"
)

func TestStartServerCanRetryAfterStartupFailure(t *testing.T) {
	resetServerState()
	port := freePort(t)
	servicebusConfig.Config = servicebusConfig.Configure{
		ServiceBus: v1alpha2.ServiceBus{
			Enable:            true,
			Server:            "127.0.0.1",
			Port:              port,
			Timeout:           1,
			TLSCertFile:       filepath.Join(t.TempDir(), "missing.crt"),
			TLSPrivateKeyFile: filepath.Join(t.TempDir(), "missing.key"),
		},
	}

	startServer()
	require.Equal(t, int32(0), atomic.LoadInt32(&inited))

	dir := t.TempDir()
	caCert, caKey, caPool := newCertificateAuthority(t)
	certPath, keyPath, _ := writeServerCertificate(t, dir, "127.0.0.1", 1, caCert, caKey)
	servicebusConfig.Config.ServiceBus.TLSCertFile = certPath
	servicebusConfig.Config.ServiceBus.TLSPrivateKeyFile = keyPath

	startServer()
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inited) == 1
	}, 3*time.Second, 50*time.Millisecond)

	resp, err := httpsClient(caPool, "").Get("https://127.0.0.1:" + strconv.Itoa(port))
	require.NoError(t, err)
	_ = resp.Body.Close()

	stopServer()
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inited) == 0
	}, 3*time.Second, 50*time.Millisecond)
}

func TestTLSServerLifecycleAndRotation(t *testing.T) {
	resetServerState()
	dir := t.TempDir()
	caCert, caKey, caPool := newCertificateAuthority(t)
	certPath, keyPath, serial1 := writeServerCertificate(t, dir, "127.0.0.1", 1, caCert, caKey)
	cfg := v1alpha2.ServiceBus{
		Enable:            true,
		Server:            "127.0.0.1",
		Port:              0,
		Timeout:           1,
		TLSCertFile:       certPath,
		TLSPrivateKeyFile: keyPath,
	}
	servicebusConfig.Config = servicebusConfig.Configure{ServiceBus: cfg}

	srv, listener, err := newTLSServer(cfg)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		serveTLS(srv, listener)
	}()
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(stopCtx)
		<-done
	})

	addr := listener.Addr().String()
	resp, err := httpsClient(caPool, "").Get("https://" + addr)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Contains(t, string(body), "invalid params")
	require.Equal(t, serial1.String(), resp.TLS.PeerCertificates[0].SerialNumber.String())

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	require.NoError(t, err)
	_, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: " + addr + "\r\n\r\n"))
	require.NoError(t, err)
	buf := make([]byte, 64)
	n, readErr := conn.Read(buf)
	_ = conn.Close()
	plainResponse := string(buf[:n])
	require.True(t, readErr != nil || !strings.Contains(plainResponse, "invalid params"))

	_, err = httpsClient(x509.NewCertPool(), "").Get("https://" + addr)
	require.Error(t, err)
	_, err = httpsClient(caPool, "wrong-host").Get("https://" + addr)
	require.Error(t, err)

	_, _, serial2 := writeServerCertificate(t, dir, "127.0.0.1", 2, caCert, caKey)
	resp, err = httpsClient(caPool, "").Get("https://" + addr)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, serial2.String(), resp.TLS.PeerCertificates[0].SerialNumber.String())
}

func httpsClient(pool *x509.CertPool, serverName string) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    pool,
			ServerName: serverName,
		},
	}
	return &http.Client{Transport: transport, Timeout: 3 * time.Second}
}

func resetServerState() {
	serverMu.Lock()
	active = nil
	serverMu.Unlock()
	atomic.StoreInt32(&inited, 0)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func newCertificateAuthority(t *testing.T) (*x509.Certificate, *rsa.PrivateKey, *x509.CertPool) {
	t.Helper()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1000),
		Subject:               pkix.Name{CommonName: "servicebus-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	return caCert, caKey, pool
}

func writeServerCertificate(t *testing.T, dir, host string, serial int64, caCert *x509.Certificate, caKey *rsa.PrivateKey) (string, string, *big.Int) {
	t.Helper()
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "servicebus"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		IPAddresses:  []net.IP{net.ParseIP(host)},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	certPath := filepath.Join(dir, "servicebus.crt")
	keyPath := filepath.Join(dir, "servicebus.key")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}), 0o644))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)}), 0o600))
	return certPath, keyPath, serverTemplate.SerialNumber
}
