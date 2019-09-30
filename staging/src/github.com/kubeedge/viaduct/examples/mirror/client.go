package main

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/examples/chat/config"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/client"
	"github.com/kubeedge/viaduct/pkg/conn"
)

func StartClient(cfg *config.Config) error {
	tls := &tls.Config{InsecureSkipVerify: true}

	var exOpts interface{}

	header := make(http.Header)
	header.Add("client_id", "client1")
	switch cfg.Type {
	case api.ProtocolTypeQuic:
		exOpts = api.QuicClientOption{
			Header: header,
		}
	case api.ProtocolTypeWS:
		exOpts = api.WSClientOption{
			Header: header,
		}
	}

	client := client.Client{
		Options: client.Options{
			Type:      cfg.Type,
			Addr:      cfg.Addr,
			TLSConfig: tls,
			AutoRoute: false,
			ConnUse:   api.UseTypeStream,
		},
		ExOpts: exOpts,
	}

	connClient, err := client.Connect()
	if err != nil {
		klog.Errorf("failed to connect peer, error: %+v", err)
		return err
	}
	klog.Info("connect successfully")
	return SendStdin(connClient)
}

func SendStdin(conn conn.Connection) error {
	_, err := io.Copy(conn, os.Stdin)
	return err
}
