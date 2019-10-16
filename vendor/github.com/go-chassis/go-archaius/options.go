package archaius

import (
	"crypto/tls"
	"github.com/go-chassis/go-archaius/source/util"

	"github.com/go-chassis/go-chassis-config"
)

// RemoteInfo has attribute for config center source initialization
type RemoteInfo struct {
	//required.
	//Key value can be in different namespace, we call it dimension.
	//although key is same but in different dimension, the value is different.
	//you must specify the service,app and version, so that the remote source will pull key value
	DefaultDimension map[string]string
	//archaius config center source support 2 types of refresh mechanism:
	//0: Web-Socket Based -  client makes an web socket connection with
	//the config server and keeps getting an events whenever any data changes.
	//1: Pull Configuration interval- In this type client keeps polling the configuration from
	//the config server at regular intervals.
	RefreshMode int

	//Pull Configuration interval, unit is second
	RefreshInterval int

	//currentConfig for config client implementation
	//if you already create a client, don't need to set those config
	URL           string
	TenantName    string
	EnableSSL     bool
	TLSConfig     *tls.Config
	AutoDiscovery bool
	ClientType    string
	APIVersion    string
	RefreshPort   string
}

//Options hold options
type Options struct {
	RequiredFiles []string
	OptionalFiles []string
	FileHandler   util.FileHandler
	RemoteInfo    *RemoteInfo
	ConfigClient  config.Client
	UseCLISource  bool
	UseENVSource  bool
	UseMemSource  bool
}

//Option is a func
type Option func(options *Options)

//WithRequiredFiles tell archaius to manage files, if not exist will return error
func WithRequiredFiles(f []string) Option {
	return func(options *Options) {
		options.RequiredFiles = f
	}
}

//WithOptionalFiles tell archaius to manage files, if not exist will NOT return error
func WithOptionalFiles(f []string) Option {
	return func(options *Options) {
		options.OptionalFiles = f
	}
}

//WithDefaultFileHandler let user custom handler
//you can decide how to convert file into kv pairs
func WithDefaultFileHandler(handler util.FileHandler) Option {
	return func(options *Options) {
		options.FileHandler = handler
	}
}

//WithRemoteSource accept the information for initiating a config center source,
//RemoteInfo is required if you want to use config center source
//client is optional,if client is nil, archaius will create one based on RemoteInfo
//config client will be injected into config source as a client to interact with a config server
func WithRemoteSource(ri *RemoteInfo, c config.Client) Option {
	return func(options *Options) {
		options.RemoteInfo = ri
		options.ConfigClient = c
	}
}

//WithCommandLineSource enable cmd line source
//archaius will read command line params as key value
func WithCommandLineSource() Option {
	return func(options *Options) {
		options.UseCLISource = true
	}
}

//WithENVSource enable env source
//archaius will read ENV as key value
func WithENVSource() Option {
	return func(options *Options) {
		options.UseENVSource = true
	}
}

//WithMemorySource accept the information for initiating a Memory source
func WithMemorySource() Option {
	return func(options *Options) {
		options.UseMemSource = true
	}
}

//FileOptions for AddFile func
type FileOptions struct {
	Handler util.FileHandler
}

//FileOption is a func
type FileOption func(options *FileOptions)

//WithFileHandler use custom handler
func WithFileHandler(h util.FileHandler) FileOption {
	return func(options *FileOptions) {
		options.Handler = h
	}

}
