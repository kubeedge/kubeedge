/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
