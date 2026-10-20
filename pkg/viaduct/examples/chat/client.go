package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/examples/chat/config"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/client"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/mux"
)

func handleClient(container *mux.MessageContainer, writer mux.ResponseWriter) {
	klog.Infof("receive message: %s", container.Message.GetContent())
	if container.Message.IsSync() {
		writer.WriteResponse(container.Message, "ack")
	}
}

func initClientEntries() {
	mux.Entry(mux.NewPattern("*").Op("*"), handleClient)
}

func StartClient(cfg *config.Config) error {
	tlsConfig, err := GetTlsConfig(cfg)
	if err != nil {
		return err
	}
	// Ensure the CA pool is used for server verification
	tlsConfig.RootCAs = tlsConfig.ClientCAs

	initClientEntries()

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
	default:
		return fmt.Errorf("unsupported protocol type: %v", cfg.Type)
	}

	c := client.Client{
		Options: client.Options{
			Type:      cfg.Type,
			Addr:      cfg.Addr,
			TLSConfig: tlsConfig,
			AutoRoute: true,
			ConnUse:   api.UseTypeMessage,
		},
		ExOpts: exOpts,
	}

	connClient, err := c.Connect()
	if err != nil {
		return err
	}
	stat := connClient.ConnectionState()
	klog.Infof("connect stat:%+v", stat)

	return SendStdin([]conn.Connection{connClient}, "client")
}

func SendStdin(conns []conn.Connection, source string) error {
	input := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("send message: ")
		inputData, err := input.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				klog.Info("stdin closed, exiting")
				for _, c := range conns {
					if err := c.Close(); err != nil {
						klog.Errorf("failed to close connection, error: %+v", err)
					}
				}
				return nil
			}
			klog.Errorf("failed to read input, error: %+v", err)
			for _, c := range conns {
				if err := c.Close(); err != nil {
					klog.Errorf("failed to close connection, error: %+v", err)
				}
			}
			return err
		}
		message := model.NewMessage("").
			BuildRouter(source, "", "viaduct_message", "update").
			FillBody(inputData)

		for _, c := range conns {
			err = c.WriteMessageAsync(message)
			if err != nil {
				klog.Errorf("failed to write message async, error:%+v", err)
			}
		}
	}
}
