package util

import (
	"crypto/tls"
	"errors"
	"os"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var (
	// TokenWaitTime to wait
	TokenWaitTime = 120 * time.Second
)

// CheckKeyExist check dis info format
func CheckKeyExist(keys []string, disinfo map[string]interface{}) error {
	for _, v := range keys {
		_, ok := disinfo[v]
		if !ok {
			log.LOGGER.Errorf("key: %s not found", v)
			return errors.New("key not found")
		}
	}
	return nil
}

// CheckClientToken checks token is right
func CheckClientToken(token MQTT.Token) (bool, error) {
	if token.Wait() && token.Error() != nil {
		return false, token.Error()
	}
	return true, nil
}

// PathExist check file exists or not
func PathExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// HubClientInit create mqtt client config
func HubClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)
	return opts
}

// LoopConnect connect to mqtt server
func LoopConnect(clientID string, client MQTT.Client) {
	for {
		log.LOGGER.Infof("start connect to mqtt server with client id: %s", clientID)
		token := client.Connect()
		log.LOGGER.Infof("client %s isconnected: %s", clientID, client.IsConnected())
		if rs, err := CheckClientToken(token); !rs {
			log.LOGGER.Errorf("connect error: %v", err)
		} else {
			return
		}
		time.Sleep(5 * time.Second)
	}
}
