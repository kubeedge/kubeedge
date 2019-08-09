package util

// HubConfig is the config for entire CloudHub
var HubConfig *Config

// Config represents configuration options for http access
type Config struct {
	ProtocolWebsocket  bool
	ProtocolQuic       bool
	MaxIncomingStreams int
	Address            string
	Port               int
	QuicPort           int
	KeepaliveInterval  int
	Ca                 []byte
	Cert               []byte
	Key                []byte
	WriteTimeout       int
	NodeLimit          int
}
