package wsclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/viaduct/pkg/api"
	wsclient "github.com/kubeedge/viaduct/pkg/client"
	"github.com/kubeedge/viaduct/pkg/conn"
)

const (
	retryCount       = 5
	cloudAccessSleep = 5 * time.Second
)

// WebSocketClient a websocket client
type WebSocketClient struct {
	config     *WebSocketConfig
	connection conn.Connection
}

// WebSocketConfig config for websocket
type WebSocketConfig struct {
	URL              string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
	NodeID           string
	ProjectID        string
}

// NewWebSocketClient initializes a new websocket client instance
func NewWebSocketClient(conf *WebSocketConfig) *WebSocketClient {
	return &WebSocketClient{config: conf}
}

// Init initializes websocket client
func (wsc *WebSocketClient) Init() error {
	klog.Infof("Websocket start to connect Access")
	cert, err := tls.LoadX509KeyPair(wsc.config.CertFilePath, wsc.config.KeyFilePath)
	if err != nil {
		klog.Errorf("Failed to load x509 key pair: %v", err)
		return fmt.Errorf("failed to load x509 key pair, error: %v", err)
	}

	caCert, err := ioutil.ReadFile(config.Config.TLSCAFile)
	if err != nil {
		return err
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		return fmt.Errorf("cannot parse the certificates")
	}

	tlsConfig := &tls.Config{
		RootCAs:            pool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}

	option := wsclient.Options{
		HandshakeTimeout: wsc.config.HandshakeTimeout,
		TLSConfig:        tlsConfig,
		Type:             api.ProtocolTypeWS,
		Addr:             wsc.config.URL,
		AutoRoute:        false,
		ConnUse:          api.UseTypeMessage,
	}
	exOpts := api.WSClientOption{Header: make(http.Header)}
	exOpts.Header.Set("node_id", wsc.config.NodeID)
	exOpts.Header.Set("project_id", wsc.config.ProjectID)
	client := &wsclient.Client{Options: option, ExOpts: exOpts}

	for i := 0; i < retryCount; i++ {
		connection, err := client.Connect()
		if err != nil {
			klog.Errorf("Init websocket connection failed %s", err.Error())
		} else {
			wsc.connection = connection
			klog.Infof("Websocket connect to cloud access successful")
			return nil
		}
		time.Sleep(cloudAccessSleep)
	}
	return errors.New("max retry count reached when connecting to cloud")
}

//Uninit closes the websocket connection
func (wsc *WebSocketClient) Uninit() {
	wsc.connection.Close()
}

//Send sends the message as JSON object through the connection
func (wsc *WebSocketClient) Send(message model.Message) error {
	return wsc.connection.WriteMessageAsync(&message)
}

//Receive reads the binary message through the connection
func (wsc *WebSocketClient) Receive() (model.Message, error) {
	message := model.Message{}
	wsc.connection.ReadMessage(&message)
	return message, nil
}

//Notify logs info
func (wsc *WebSocketClient) Notify(authInfo map[string]string) {
	klog.Infof("no op")
}
