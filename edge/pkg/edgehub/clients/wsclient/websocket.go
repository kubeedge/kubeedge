package wsclient

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
)

const (
	retryCount       = 5
	cloudAccessSleep = 60 * time.Second
)

//WebSocketClient defines websocket client object
type WebSocketClient struct {
	webConn  *websocket.Conn
	sendLock sync.Mutex
	config   *WebSocketConfig
}

//WebSocketConfig defines configuration object
type WebSocketConfig struct {
	URL              string
	CertFilePath     string
	KeyFilePath      string
	HandshakeTimeout time.Duration
	ReadDeadline     time.Duration
	WriteDeadline    time.Duration
	ExtendHeader     http.Header
}

//NewWebSocketClient returns a new web socket client object with its configuration
func NewWebSocketClient(conf *WebSocketConfig) *WebSocketClient {
	return &WebSocketClient{config: conf}
}

// Init initializes websocket client
func (wcc *WebSocketClient) Init() error {
	log.LOGGER.Infof("Start to connect Access")
	cert, err := tls.LoadX509KeyPair(wcc.config.CertFilePath, wcc.config.KeyFilePath)
	if err != nil {
		log.LOGGER.Errorf("Failed to load x509 key pair: %v", err)
		return fmt.Errorf("failed to load x509 key pair, error: %v", err)
	}

	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		},
		HandshakeTimeout: wcc.config.HandshakeTimeout,
	}

	for i := 0; i < retryCount; i++ {
		conn, resp, err := dialer.Dial(wcc.config.URL, wcc.config.ExtendHeader)
		if err != nil {
			var respMsg string
			if resp != nil {
				body, rErr := ioutil.ReadAll(resp.Body)
				if rErr == nil {
					respMsg = fmt.Sprintf(", response code: %d, response body: %s", resp.StatusCode, string(body))
				} else {
					respMsg = fmt.Sprintf(", response code: %d", resp.StatusCode)
				}
				resp.Body.Close()
			}
			log.LOGGER.Errorf("Error when init websocket connection%s: %v", respMsg, err)
		} else {
			log.LOGGER.Infof("Success to connect Access")
			wcc.webConn = conn
			return nil
		}
		time.Sleep(cloudAccessSleep)
	}
	return errors.New("max retry count to connect Access")
}

//Uninit closes the web socket connection
func (wcc *WebSocketClient) Uninit() {
	wcc.webConn.Close()
}

//Send sends the message as binary message through the connection
func (wcc *WebSocketClient) Send(message model.Message) error {
	deadline := time.Now().Add(wcc.config.WriteDeadline)
	wcc.sendLock.Lock()
	defer wcc.sendLock.Unlock()
	wcc.webConn.SetWriteDeadline(deadline)

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("websocket write msg failed with marshal failed. error %s", err.Error())
	}

	return wcc.webConn.WriteMessage(websocket.BinaryMessage, data)
}

//Receive reads the binary message through the connection
func (wcc *WebSocketClient) Receive() (model.Message, error) {
	var message model.Message

	_, buf, err := wcc.webConn.ReadMessage()
	if err != nil {
		return model.Message{}, err
	}

	err = json.Unmarshal(buf, &message)
	if err != nil {
		log.LOGGER.Errorf("Failed to read json: %v", err)
		return model.Message{}, err
	}

	return message, nil
}

//Notify logs info
func (wcc *WebSocketClient) Notify(authInfo map[string]string) {
	log.LOGGER.Infof("don't care")
}
