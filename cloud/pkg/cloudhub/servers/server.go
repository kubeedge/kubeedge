package servers

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/server"
)

// StartCloudHub starts the cloud hub service
func StartCloudHub(messageHandler handler.Handler) {
	// start websocket server
	if hubconfig.Config.WebSocket.Enable {
		go startWebsocketServer(messageHandler)
	}
	// start quic server
	if hubconfig.Config.Quic.Enable {
		go startQuicServer(messageHandler)
	}
}

func createTLSConfig(ca []byte, certFallback, keyFallback []byte, certFile, keyFile string) tls.Config {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: ca}))
	if !ok {
		panic(fmt.Errorf("fail to load ca content"))
	}

	// maintain backwards compatibility with the DER encoded cert read from
	// the kubernetes secret
	var fallback tls.Certificate
	if certFallback != nil && keyFallback != nil {
		certPem := pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: certFallback})
		keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyFallback})
		var err error
		fallback, err = tls.X509KeyPair(certPem, keyPem)
		if err != nil {
			klog.Exitf("Failed to create keypair: %v", err)
		}
	}

	return tls.Config{
		ClientCAs:  pool,
		ClientAuth: tls.RequireAndVerifyClientCert,
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if certFile != "" && keyFile != "" {
				// dynamically read the certificate if files are defined
				certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
				if err != nil {
					klog.Exitf("Failed to load cert %q and key %q: %v", certFile, keyFile, err)
				}
				return &certificate, nil
			}
			return &fallback, nil
		},
		MinVersion: tls.VersionTLS12,
		// has to match cipher used by NewPrivateKey method, currently is ECDSA
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}
}

func startWebsocketServer(messageHandler handler.Handler) {
	tlsConfig := createTLSConfig(hubconfig.Config.Ca, hubconfig.Config.Cert, hubconfig.Config.Key, hubconfig.Config.TLSCertFile, hubconfig.Config.TLSPrivateKeyFile)
	svc := server.Server{
		Type:               api.ProtocolTypeWS,
		TLSConfig:          &tlsConfig,
		AutoRoute:          true,
		ConnNotify:         messageHandler.HandleConnection,
		OnReadTransportErr: messageHandler.OnReadTransportErr,
		Addr:               fmt.Sprintf("%s:%d", hubconfig.Config.WebSocket.Address, hubconfig.Config.WebSocket.Port),
		ExOpts:             api.WSServerOption{Path: "/"},
	}
	klog.Infof("Starting cloudhub %s server", api.ProtocolTypeWS)
	klog.Exit(svc.ListenAndServeTLS("", ""))
}

func startQuicServer(messageHandler handler.Handler) {
	tlsConfig := createTLSConfig(hubconfig.Config.Ca, hubconfig.Config.Cert, hubconfig.Config.Key, hubconfig.Config.TLSCertFile, hubconfig.Config.TLSPrivateKeyFile)
	svc := server.Server{
		Type:               api.ProtocolTypeQuic,
		TLSConfig:          &tlsConfig,
		AutoRoute:          true,
		ConnNotify:         messageHandler.HandleConnection,
		OnReadTransportErr: messageHandler.OnReadTransportErr,
		Addr:               fmt.Sprintf("%s:%d", hubconfig.Config.Quic.Address, hubconfig.Config.Quic.Port),
		ExOpts:             api.QuicServerOption{MaxIncomingStreams: int(hubconfig.Config.Quic.MaxIncomingStreams)},
	}

	klog.Infof("Starting cloudhub %s server", api.ProtocolTypeQuic)
	klog.Exit(svc.ListenAndServeTLS("", ""))
}
