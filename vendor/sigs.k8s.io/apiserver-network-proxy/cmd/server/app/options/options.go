package options

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/apiserver-network-proxy/pkg/server"
	"sigs.k8s.io/apiserver-network-proxy/pkg/util"
)

type ProxyRunOptions struct {
	// Certificate setup for securing communication to the "client" i.e. the Kube API Server.
	ServerCert   string
	ServerKey    string
	ServerCaCert string
	// Certificate setup for securing communication to the "agent" i.e. the managed cluster.
	ClusterCert   string
	ClusterKey    string
	ClusterCaCert string
	// Flag to switch between gRPC and HTTP Connect
	Mode string
	// Location for use by the "unix" network. Setting enables UDS for server connections.
	UdsName string
	// If file UdsName already exists, delete the file before listen on that UDS file.
	DeleteUDSFile bool
	// Port we listen for server connections on.
	ServerPort uint
	// Port we listen for agent connections on.
	AgentPort uint
	// Port we listen for admin connections on.
	AdminPort uint
	// Port we listen for health connections on.
	HealthPort uint
	// After a duration of this time if the server doesn't see any activity it
	// pings the client to see if the transport is still alive.
	KeepaliveTime         time.Duration
	FrontendKeepaliveTime time.Duration
	// Enables pprof at host:AdminPort/debug/pprof.
	EnableProfiling bool
	// If EnableProfiling is true, this enables the lock contention
	// profiling at host:AdminPort/debug/pprof/block.
	EnableContentionProfiling bool

	// ID of this proxy server.
	ServerID string
	// Number of proxy server instances, should be 1 unless it is a HA proxy server.
	ServerCount uint
	// Agent pod's namespace for token-based agent authentication
	AgentNamespace string
	// Agent pod's service account for token-based agent authentication
	AgentServiceAccount string
	// Token's audience for token-based agent authentication
	AuthenticationAudience string
	// Path to kubeconfig (used by kubernetes client)
	KubeconfigPath string
	// Client maximum QPS.
	KubeconfigQPS float32
	// Client maximum burst for throttle.
	KubeconfigBurst int

	// Proxy strategies used by the server.
	// NOTE the order of the strategies matters. e.g., for list
	// "destHost,destCIDR", the server will try to find a backend associating
	// to the destination host first, if not found, it will try to find a
	// backend within the destCIDR. if it still can't find any backend,
	// it will use the default backend manager to choose a random backend.
	ProxyStrategies string

	// This controls if we attempt to push onto a "full" transfer channel.
	// However checking that the transfer channel is full is not safe.
	// It violates our race condition checking. Adding locks around a potentially
	// blocking call has its own problems, so it cannot easily be made race condition safe.
	// The check is an "unlocked" read but is still use at your own peril.
	WarnOnChannelLimit bool

	// Cipher suites used by the server.
	// If empty, the default suite will be used from tls.CipherSuites(),
	// also checks if given comma separated list contains cipher from tls.InsecureCipherSuites().
	// NOTE that cipher suites are not configurable for TLS1.3,
	// see: https://pkg.go.dev/crypto/tls#Config, so in that case, this option won't have any effect.
	CipherSuites string
}

func (o *ProxyRunOptions) Flags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("proxy-server", pflag.ContinueOnError)
	flags.StringVar(&o.ServerCert, "server-cert", o.ServerCert, "If non-empty secure communication with this cert.")
	flags.StringVar(&o.ServerKey, "server-key", o.ServerKey, "If non-empty secure communication with this key.")
	flags.StringVar(&o.ServerCaCert, "server-ca-cert", o.ServerCaCert, "If non-empty the CA we use to validate KAS clients.")
	flags.StringVar(&o.ClusterCert, "cluster-cert", o.ClusterCert, "If non-empty secure communication with this cert.")
	flags.StringVar(&o.ClusterKey, "cluster-key", o.ClusterKey, "If non-empty secure communication with this key.")
	flags.StringVar(&o.ClusterCaCert, "cluster-ca-cert", o.ClusterCaCert, "If non-empty the CA we use to validate Agent clients.")
	flags.StringVar(&o.Mode, "mode", o.Mode, "mode can be either 'grpc' or 'http-connect'.")
	flags.StringVar(&o.UdsName, "uds-name", o.UdsName, "uds-name should be empty for TCP traffic. For UDS set to its name.")
	flags.BoolVar(&o.DeleteUDSFile, "delete-existing-uds-file", o.DeleteUDSFile, "If true and if file UdsName already exists, delete the file before listen on that UDS file")
	flags.UintVar(&o.ServerPort, "server-port", o.ServerPort, "Port we listen for server connections on. Set to 0 for UDS.")
	flags.UintVar(&o.AgentPort, "agent-port", o.AgentPort, "Port we listen for agent connections on.")
	flags.UintVar(&o.AdminPort, "admin-port", o.AdminPort, "Port we listen for admin connections on.")
	flags.UintVar(&o.HealthPort, "health-port", o.HealthPort, "Port we listen for health connections on.")
	flags.DurationVar(&o.KeepaliveTime, "keepalive-time", o.KeepaliveTime, "Time for gRPC agent server keepalive.")
	flags.DurationVar(&o.FrontendKeepaliveTime, "frontend-keepalive-time", o.FrontendKeepaliveTime, "Time for gRPC frontend server keepalive.")
	flags.BoolVar(&o.EnableProfiling, "enable-profiling", o.EnableProfiling, "enable pprof at host:admin-port/debug/pprof")
	flags.BoolVar(&o.EnableContentionProfiling, "enable-contention-profiling", o.EnableContentionProfiling, "enable contention profiling at host:admin-port/debug/pprof/block. \"--enable-profiling\" must also be set.")
	flags.StringVar(&o.ServerID, "server-id", o.ServerID, "The unique ID of this server.")
	flags.UintVar(&o.ServerCount, "server-count", o.ServerCount, "The number of proxy server instances, should be 1 unless it is an HA server.")
	flags.StringVar(&o.AgentNamespace, "agent-namespace", o.AgentNamespace, "Expected agent's namespace during agent authentication (used with agent-service-account, authentication-audience, kubeconfig).")
	flags.StringVar(&o.AgentServiceAccount, "agent-service-account", o.AgentServiceAccount, "Expected agent's service account during agent authentication (used with agent-namespace, authentication-audience, kubeconfig).")
	flags.StringVar(&o.KubeconfigPath, "kubeconfig", o.KubeconfigPath, "absolute path to the kubeconfig file (used with agent-namespace, agent-service-account, authentication-audience).")
	flags.Float32Var(&o.KubeconfigQPS, "kubeconfig-qps", o.KubeconfigQPS, "Maximum client QPS (proxy server uses this client to authenticate agent tokens).")
	flags.IntVar(&o.KubeconfigBurst, "kubeconfig-burst", o.KubeconfigBurst, "Maximum client burst (proxy server uses this client to authenticate agent tokens).")
	flags.StringVar(&o.AuthenticationAudience, "authentication-audience", o.AuthenticationAudience, "Expected agent's token authentication audience (used with agent-namespace, agent-service-account, kubeconfig).")
	flags.StringVar(&o.ProxyStrategies, "proxy-strategies", o.ProxyStrategies, "The list of proxy strategies used by the server to pick a backend/tunnel, available strategies are: default, destHost.")
	flags.BoolVar(&o.WarnOnChannelLimit, "warn-on-channel-limit", o.WarnOnChannelLimit, "Turns on a warning if the system is going to push to a full channel. The check involves an unsafe read.")
	flags.StringVar(&o.CipherSuites, "cipher-suites", o.CipherSuites, "The comma separated list of allowed cipher suites. Has no effect on TLS1.3. Empty means allow default list.")
	return flags
}

func (o *ProxyRunOptions) Print() {
	klog.V(1).Infof("ServerCert set to %q.\n", o.ServerCert)
	klog.V(1).Infof("ServerKey set to %q.\n", o.ServerKey)
	klog.V(1).Infof("ServerCACert set to %q.\n", o.ServerCaCert)
	klog.V(1).Infof("ClusterCert set to %q.\n", o.ClusterCert)
	klog.V(1).Infof("ClusterKey set to %q.\n", o.ClusterKey)
	klog.V(1).Infof("ClusterCACert set to %q.\n", o.ClusterCaCert)
	klog.V(1).Infof("Mode set to %q.\n", o.Mode)
	klog.V(1).Infof("UDSName set to %q.\n", o.UdsName)
	klog.V(1).Infof("DeleteUDSFile set to %v.\n", o.DeleteUDSFile)
	klog.V(1).Infof("Server port set to %d.\n", o.ServerPort)
	klog.V(1).Infof("Agent port set to %d.\n", o.AgentPort)
	klog.V(1).Infof("Admin port set to %d.\n", o.AdminPort)
	klog.V(1).Infof("Health port set to %d.\n", o.HealthPort)
	klog.V(1).Infof("Keepalive time set to %v.\n", o.KeepaliveTime)
	klog.V(1).Infof("Frontend keepalive time set to %v.\n", o.FrontendKeepaliveTime)
	klog.V(1).Infof("EnableProfiling set to %v.\n", o.EnableProfiling)
	klog.V(1).Infof("EnableContentionProfiling set to %v.\n", o.EnableContentionProfiling)
	klog.V(1).Infof("ServerID set to %s.\n", o.ServerID)
	klog.V(1).Infof("ServerCount set to %d.\n", o.ServerCount)
	klog.V(1).Infof("AgentNamespace set to %q.\n", o.AgentNamespace)
	klog.V(1).Infof("AgentServiceAccount set to %q.\n", o.AgentServiceAccount)
	klog.V(1).Infof("AuthenticationAudience set to %q.\n", o.AuthenticationAudience)
	klog.V(1).Infof("KubeconfigPath set to %q.\n", o.KubeconfigPath)
	klog.V(1).Infof("KubeconfigQPS set to %f.\n", o.KubeconfigQPS)
	klog.V(1).Infof("KubeconfigBurst set to %d.\n", o.KubeconfigBurst)
	klog.V(1).Infof("ProxyStrategies set to %q.\n", o.ProxyStrategies)
	klog.V(1).Infof("WarnOnChannelLimit set to %t.\n", o.WarnOnChannelLimit)
	klog.V(1).Infof("CipherSuites set to %q.\n", o.CipherSuites)
}

func (o *ProxyRunOptions) Validate() error {
	if o.ServerKey != "" {
		if _, err := os.Stat(o.ServerKey); os.IsNotExist(err) {
			return fmt.Errorf("error checking server key %s, got %v", o.ServerKey, err)
		}
		if o.ServerCert == "" {
			return fmt.Errorf("cannot have server cert empty when server key is set to %q", o.ServerKey)
		}
	}
	if o.ServerCert != "" {
		if _, err := os.Stat(o.ServerCert); os.IsNotExist(err) {
			return fmt.Errorf("error checking server cert %s, got %v", o.ServerCert, err)
		}
		if o.ServerKey == "" {
			return fmt.Errorf("cannot have server key empty when server cert is set to %q", o.ServerCert)
		}
	}
	if o.ServerCaCert != "" {
		if _, err := os.Stat(o.ServerCaCert); os.IsNotExist(err) {
			return fmt.Errorf("error checking server CA cert %s, got %v", o.ServerCaCert, err)
		}
	}
	if o.ClusterKey != "" {
		if _, err := os.Stat(o.ClusterKey); os.IsNotExist(err) {
			return fmt.Errorf("error checking cluster key %s, got %v", o.ClusterKey, err)
		}
		if o.ClusterCert == "" {
			return fmt.Errorf("cannot have cluster cert empty when cluster key is set to %q", o.ClusterKey)
		}
	}
	if o.ClusterCert != "" {
		if _, err := os.Stat(o.ClusterCert); os.IsNotExist(err) {
			return fmt.Errorf("error checking cluster cert %s, got %v", o.ClusterCert, err)
		}
		if o.ClusterKey == "" {
			return fmt.Errorf("cannot have cluster key empty when cluster cert is set to %q", o.ClusterCert)
		}
	}
	if o.ClusterCaCert != "" {
		if _, err := os.Stat(o.ClusterCaCert); os.IsNotExist(err) {
			return fmt.Errorf("error checking cluster CA cert %s, got %v", o.ClusterCaCert, err)
		}
	}
	if o.Mode != "grpc" && o.Mode != "http-connect" {
		return fmt.Errorf("mode must be set to either 'grpc' or 'http-connect' not %q", o.Mode)
	}
	if o.UdsName != "" {
		if o.ServerPort != 0 {
			return fmt.Errorf("server port should be set to 0 not %d for UDS", o.ServerPort)
		}
		if o.ServerKey != "" {
			return fmt.Errorf("server key should not be set for UDS")
		}
		if o.ServerCert != "" {
			return fmt.Errorf("server cert should not be set for UDS")
		}
		if o.ServerCaCert != "" {
			return fmt.Errorf("server ca cert should not be set for UDS")
		}
	}
	if o.ServerPort > 49151 {
		return fmt.Errorf("please do not try to use ephemeral port %d for the server port", o.ServerPort)
	}
	if o.AgentPort > 49151 {
		return fmt.Errorf("please do not try to use ephemeral port %d for the agent port", o.AgentPort)
	}
	if o.AdminPort > 49151 {
		return fmt.Errorf("please do not try to use ephemeral port %d for the admin port", o.AdminPort)
	}
	if o.HealthPort > 49151 {
		return fmt.Errorf("please do not try to use ephemeral port %d for the health port", o.HealthPort)
	}

	if o.ServerPort < 1024 {
		if o.UdsName == "" {
			return fmt.Errorf("please do not try to use reserved port %d for the server port", o.ServerPort)
		}
	}
	if o.AgentPort < 1024 {
		return fmt.Errorf("please do not try to use reserved port %d for the agent port", o.AgentPort)
	}
	if o.AdminPort < 1024 {
		return fmt.Errorf("please do not try to use reserved port %d for the admin port", o.AdminPort)
	}
	if o.HealthPort < 1024 {
		return fmt.Errorf("please do not try to use reserved port %d for the health port", o.HealthPort)
	}
	if o.EnableContentionProfiling && !o.EnableProfiling {
		return fmt.Errorf("if --enable-contention-profiling is set, --enable-profiling must also be set")
	}

	// validate agent authentication params
	// all 4 parameters must be empty or must have value (except KubeconfigPath that might be empty)
	if o.AgentNamespace != "" || o.AgentServiceAccount != "" || o.AuthenticationAudience != "" || o.KubeconfigPath != "" {
		if o.ClusterCaCert != "" {
			return fmt.Errorf("ClusterCaCert can not be used when service account authentication is enabled")
		}
		if o.AgentNamespace == "" {
			return fmt.Errorf("AgentNamespace cannot be empty when agent authentication is enabled")
		}
		if o.AgentServiceAccount == "" {
			return fmt.Errorf("AgentServiceAccount cannot be empty when agent authentication is enabled")
		}
		if o.AuthenticationAudience == "" {
			return fmt.Errorf("AuthenticationAudience cannot be empty when agent authentication is enabled")
		}
		if o.KubeconfigPath != "" {
			if _, err := os.Stat(o.KubeconfigPath); os.IsNotExist(err) {
				return fmt.Errorf("error checking KubeconfigPath %q, got %v", o.KubeconfigPath, err)
			}
		}
	}

	// validate the proxy strategies
	if o.ProxyStrategies != "" {
		pss := strings.Split(o.ProxyStrategies, ",")
		for _, ps := range pss {
			switch ps {
			case string(server.ProxyStrategyDestHost):
			case string(server.ProxyStrategyDefault):
			case string(server.ProxyStrategyDefaultRoute):
			default:
				return fmt.Errorf("unknown proxy strategy: %s, available strategy are: default, destHost, defaultRoute", ps)
			}
		}
	}

	// validate the cipher suites
	if o.CipherSuites != "" {
		acceptedCiphers := util.GetAcceptedCiphers()
		css := strings.Split(o.CipherSuites, ",")
		for _, cipher := range css {
			_, ok := acceptedCiphers[cipher]
			if !ok {
				return fmt.Errorf("cipher suite %s not supported, doesn't exist or considered as insecure", cipher)
			}
		}
	}

	return nil
}

func NewProxyRunOptions() *ProxyRunOptions {
	o := ProxyRunOptions{
		ServerCert:                "",
		ServerKey:                 "",
		ServerCaCert:              "",
		ClusterCert:               "",
		ClusterKey:                "",
		ClusterCaCert:             "",
		Mode:                      "grpc",
		UdsName:                   "",
		DeleteUDSFile:             false,
		ServerPort:                8090,
		AgentPort:                 8091,
		HealthPort:                8092,
		AdminPort:                 8095,
		KeepaliveTime:             1 * time.Hour,
		FrontendKeepaliveTime:     1 * time.Hour,
		EnableProfiling:           false,
		EnableContentionProfiling: false,
		ServerID:                  uuid.New().String(),
		ServerCount:               1,
		AgentNamespace:            "",
		AgentServiceAccount:       "",
		KubeconfigPath:            "",
		KubeconfigQPS:             0,
		KubeconfigBurst:           0,
		AuthenticationAudience:    "",
		ProxyStrategies:           "default",
		WarnOnChannelLimit:        false,
		CipherSuites:              "",
	}
	return &o
}
