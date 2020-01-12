package quicclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
	qclient "github.com/kubeedge/viaduct/pkg/client"
	"github.com/kubeedge/viaduct/pkg/conn"
)

// QuicClient a quic client
type QuicClient struct {
	config *QuicConfig
	client conn.Connection
}

// QuicConfig config for quic
type QuicConfig struct {
	Addr             string
	CaFilePath       string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
	NodeID           string
	ProjectID        string
}

// NewQuicClient initializes a new quic client instance
func NewQuicClient(conf *QuicConfig) *QuicClient {
	return &QuicClient{config: conf}
}

// Init initializes quic client
func (qcc *QuicClient) Init() error {
	klog.Infof("Quic start to connect Access")
	cert, err := tls.LoadX509KeyPair(qcc.config.CertFilePath, qcc.config.KeyFilePath)
	if err != nil {
		klog.Errorf("Failed to load x509 key pair: %v", err)
		return fmt.Errorf("failed to load x509 key pair, error: %v", err)
	}
	caCrt, err := ioutil.ReadFile(qcc.config.CaFilePath)
	if err != nil {
		klog.Errorf("Failed to load ca file: %s", err.Error())
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
	exOpts := api.QuicClientOption{Header: make(http.Header)}
	exOpts.Header.Set("node_id", qcc.config.NodeID)
	exOpts.Header.Set("project_id", qcc.config.ProjectID)
	client := qclient.NewQuicClient(option, exOpts)
	connection, err := client.Connect()
	if err != nil {
		klog.Errorf("Init quic connection failed %s", err.Error())
		return err
	}
	qcc.client = connection
	klog.Infof("Quic connect to cloud access successful")

	return nil
}

//Uninit closes the quic connection
func (qcc *QuicClient) Uninit() {
	qcc.client.Close()
}

//Send sends the message as JSON object through the connection
func (qcc *QuicClient) Send(message model.Message) error {
	return qcc.client.WriteMessageAsync(&message)
}

//Receive reads the binary message through the connection
func (qcc *QuicClient) Receive() (model.Message, error) {
	message := model.Message{}
	qcc.client.ReadMessage(&message)
	return message, nil
}

//Notify logs info
func (qcc *QuicClient) Notify(authInfo map[string]string) {
	klog.Infof("Don not care")
}
