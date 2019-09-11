package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/examples/chat/config"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/cmgr"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/server"
)

// just for testing
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}

type loggerWriter struct{}

func (w loggerWriter) Write(b []byte) (int, error) {
	fmt.Print(string(b))
	return len(b), nil
}

func ConnNotify(conn conn.Connection) {
	go func() {
		_, err := io.Copy(loggerWriter{}, conn)
		klog.Infof("error: %+v", err)
	}()
}

func StartServer(cfg *config.Config) error {
	tls := generateTLSConfig()

	connMgr := cmgr.NewManager(nil)

	var exOpts interface{}
	switch cfg.Type {
	case api.ProtocolTypeQuic:
		exOpts = api.QuicServerOption{}
	case api.ProtocolTypeWS:
		exOpts = api.WSServerOption{
			Path: "/test",
		}
	}

	server := server.Server{
		Type:       cfg.Type,
		Addr:       cfg.Addr,
		TLSConfig:  tls,
		AutoRoute:  false,
		ConnMgr:    connMgr,
		ConnNotify: ConnNotify,
		ExOpts:     exOpts,
	}

	return server.ListenAndServeTLS("", "")
}
