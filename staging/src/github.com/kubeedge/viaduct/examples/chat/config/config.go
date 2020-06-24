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

package config

import "flag"

type Config struct {
	Addr     string
	CmdType  string
	KeyFile  string
	CertFile string
	CaFile   string
	Type     string
}

func InitConfig() *Config {
	config := Config{}

	flag.StringVar(&config.Addr, "addr", "127.0.0.1:9890", "the addr of server or client")
	flag.StringVar(&config.CmdType, "cmd-type", "server", "client or server")
	flag.StringVar(&config.KeyFile, "key", "", "the path of the key file")
	flag.StringVar(&config.CertFile, "cert", "", "the path of the cert file")
	flag.StringVar(&config.CaFile, "ca", "", "the path of the ca file")
	flag.StringVar(&config.Type, "type", "quic", "protocol type, websocket|quic")
	flag.Parse()

	return &config
}
