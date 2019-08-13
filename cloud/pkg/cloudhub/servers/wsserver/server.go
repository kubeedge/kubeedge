package wsserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/server"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/channelq"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
)

// the api path
var (
	pathEvent = fmt.Sprintf("/{%s}/{%s}/events", model.ProjectID, model.NodeID)
)

// StartCloudHub starts the cloud hub service
func StartCloudHub(config *util.Config, eventq *channelq.ChannelEventQueue, c *context.Context) error {
	// init certificate
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(config.Ca)
	if !ok {
		return fmt.Errorf("fail to load ca content")
	}
	cert, err := tls.X509KeyPair(config.Cert, config.Key)
	if err != nil {
		return err
	}
	tlsConfig := tls.Config{
		ClientCAs:    pool,
		ClientAuth:   tls.RequestClientCert,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}

	handler.InitHandler(config, eventq, c)

	s := server.Server{
		Type:       api.ProtocolTypeWS,
		Addr:       fmt.Sprintf("%s:%d", config.Address, config.Port),
		TLSConfig:  &tlsConfig,
		AutoRoute:  true,
		ConnNotify: handler.CloudhubHandler.OnRegister,
		ExOpts:     api.WSServerOption{Path: "/"},
	}

	klog.Info("Start cloud hub websocket server")
	go s.ListenAndServeTLS("", "")

	return nil
}
