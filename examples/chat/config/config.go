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
