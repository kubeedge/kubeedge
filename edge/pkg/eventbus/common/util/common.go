package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
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
			klog.Errorf("key: %s not found", v)
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

	klog.V(4).Infof("Start to set TLS configuration for MQTT client")
	tlsConfig := &tls.Config{}
	if eventconfig.Config.TLS.Enable {
		cert, err := tls.LoadX509KeyPair(eventconfig.Config.TLS.TLSMqttCertFile, eventconfig.Config.TLS.TLSMqttPrivateKeyFile)
		if err != nil {
			klog.Errorf("Failed to load x509 key pair: %v", err)
			return nil
		}

		caCert, err := ioutil.ReadFile(eventconfig.Config.TLS.TLSMqttCAFile)
		if err != nil {
			klog.Errorf("Failed to read TLSMqttCAFile")
			return nil
		}

		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caCert); !ok {
			klog.Errorf("Cannot parse the certificates")
			return nil
		}

		tlsConfig = &tls.Config{
			RootCAs:            pool,
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: false,
		}
	} else {
		tlsConfig = &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	}
	opts.SetTLSConfig(tlsConfig)
	klog.V(4).Infof("set TLS configuration for MQTT client successfully")

	return opts
}

// LoopConnect connect to mqtt server
func LoopConnect(clientID string, client MQTT.Client) {
	for {
		klog.Infof("start connect to mqtt server with client id: %s", clientID)
		token := client.Connect()
		klog.Infof("client %s isconnected: %v", clientID, client.IsConnected())
		if rs, err := CheckClientToken(token); !rs {
			klog.Errorf("connect error: %v", err)
		} else {
			return
		}
		time.Sleep(5 * time.Second)
	}
}
