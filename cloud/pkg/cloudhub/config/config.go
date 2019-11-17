package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
)

var c Configure
var once sync.Once

type Configure struct {
	ProtocolWebsocket  bool
	ProtocolQuic       bool
	ProtocolUDS        bool
	MaxIncomingStreams int
	Address            string
	Port               int
	QuicPort           int
	UDSAddress         string
	KeepaliveInterval  int
	Ca                 []byte
	Cert               []byte
	Key                []byte
	WriteTimeout       int
	NodeLimit          int
}

func InitConfigure() {
	once.Do(func() {
		var errs []error
		defer func() {
			if len(errs) != 0 {
				klog.Error("init cloudhub config error")
				for _, e := range errs {
					klog.Errorf("%v", e)
				}
				os.Exit(1)
			}
		}()
		protocolWebsocket, err := config.CONFIG.GetValue("cloudhub.protocol_websocket").ToBool()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.protocol_websocket configuration key error %v", err))
			return
		}
		protocolQuic, err := config.CONFIG.GetValue("cloudhub.protocol_quic").ToBool()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.protocol_quic configuration key error %v", err))
		}
		protocolUDS, err := config.CONFIG.GetValue("cloudhub.enable_uds").ToBool()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.enable_uds configuration key error %v", err))
		}
		address, err := config.CONFIG.GetValue("cloudhub.address").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.address configuration key error %v", err))
		}
		port, err := config.CONFIG.GetValue("cloudhub.port").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.port configuration key error %v", err))
		}
		quicPort, err := config.CONFIG.GetValue("cloudhub.quic_port").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.quic_port configuration key error %v", err))
		}
		maxIncomingStreams, err := config.CONFIG.GetValue("cloudhub.max_incomingstreams").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.max_incomingstreams configuration key error %v", err))
		}
		udsAddress, err := config.CONFIG.GetValue("cloudhub.uds_address").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.uds_address configuration key error %v", err))
		}
		keepaliveInterval, err := config.CONFIG.GetValue("cloudhub.keepalive-interval").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.keepalive-interval configuration key error %v", err))
		}
		writeTimeout, err := config.CONFIG.GetValue("cloudhub.write-timeout").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.write-timeout configuration key error %v", err))
		}
		nodeLimit, err := config.CONFIG.GetValue("cloudhub.node-limit").ToInt()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.node-limit configuration key error %v", err))
		}
		cafile, err := config.CONFIG.GetValue("cloudhub.ca").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.ca configuration key error %v", err))
		}
		certfile, err := config.CONFIG.GetValue("cloudhub.cert").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.cert configuration key error %v", err))
		}
		keyfile, err := config.CONFIG.GetValue("cloudhub.key").ToString()
		if err != nil {
			errs = append(errs, fmt.Errorf("get cloudhub.key configuration key error %v", err))
		}
		ca, err := ioutil.ReadFile(cafile)
		if err != nil {
			errs = append(errs, fmt.Errorf("read ca file %v error %v", cafile, err))
		}
		cert, err := ioutil.ReadFile(certfile)
		if err != nil {
			errs = append(errs, fmt.Errorf("read cert file %v error %v", certfile, err))
		}
		key, err := ioutil.ReadFile(keyfile)
		if err != nil {
			errs = append(errs, fmt.Errorf("read key file %v error %v", keyfile, err))
		}
		c = Configure{
			ProtocolWebsocket:  protocolWebsocket,
			ProtocolQuic:       protocolQuic,
			ProtocolUDS:        protocolUDS,
			Address:            address,
			Port:               port,
			QuicPort:           quicPort,
			MaxIncomingStreams: maxIncomingStreams,
			UDSAddress:         udsAddress,
			KeepaliveInterval:  keepaliveInterval,
			WriteTimeout:       writeTimeout,
			NodeLimit:          nodeLimit,
			Ca:                 ca,
			Cert:               cert,
			Key:                key,
		}
		if !c.ProtocolWebsocket && !c.ProtocolQuic {
			c.ProtocolWebsocket = true
		}
	})
}
func Get() Configure {
	return c
}
