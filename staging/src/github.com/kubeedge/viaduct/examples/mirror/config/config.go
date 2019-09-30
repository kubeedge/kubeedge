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
	flag.StringVar(&config.Type, "type", "quic", "protocol type, websocket|quic")
	flag.Parse()

	return &config
}
