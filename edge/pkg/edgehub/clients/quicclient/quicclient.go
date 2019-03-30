package quicclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/kubeedge/viaduct/pkg/api"
	"io/ioutil"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	qclient "github.com/kubeedge/viaduct/pkg/client"
	"github.com/kubeedge/viaduct/pkg/conn"
)

const Default_MaxIncomingStreams = 10000

type QuicClient struct {
	config *QuicConfig
	client conn.Connection
}

type QuicConfig struct {
	Addr             string
	CaFilePath       string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
}

func NewQuicClient(conf *QuicConfig) *QuicClient {
	return &QuicClient{config: conf}
}

func (qcc *QuicClient) Init() error {
	log.LOGGER.Infof("quic start to connect Access")
	cert, err := tls.LoadX509KeyPair(qcc.config.CertFilePath, qcc.config.KeyFilePath)
	if err != nil {
		log.LOGGER.Errorf("failed to load x509 key pair: %v", err)
		return fmt.Errorf("failed to load x509 key pair, error: %v", err)
	}
	caCrt, err := ioutil.ReadFile(qcc.config.CaFilePath)
	if err != nil {
		log.LOGGER.Errorf("failed to load ca file: %s", err.Error())
		return fmt.Errorf("failed to load ca file: %s", err.Error())
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCrt)

	tlsConfig := &tls.Config{
		RootCAs:            pool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	option := qclient.Options{
		HandshakeTimeout: qcc.config.HandshakeTimeout,
		TLSConfig:        tlsConfig,
		Type:             api.ProtocolTypeQuic,
		Addr:             qcc.config.Addr,
	}
	exOpts := api.QuicClientOption{}
	client := qclient.NewQuicClient(option, exOpts)
	connection, err := client.Connect()
	if err != nil {
		log.LOGGER.Errorf("init quic connection failed %s", err.Error())
		return err
	}
	qcc.client = connection
	log.LOGGER.Infof("quic connect to cloud access successful")

	return nil
}

func (qcc *QuicClient) Uninit() {
	qcc.client.Close()
}

func (qcc *QuicClient) Send(message model.Message) error {
	return qcc.client.WriteMessageAsync(&message)
}

func (qcc *QuicClient) Receive() (model.Message, error) {
	message := model.Message{}
	qcc.client.ReadMessage(&message)
	return message, nil
}

func (qcc *QuicClient) Notify(authInfo map[string]string) {
	log.LOGGER.Infof("don not care")
}
