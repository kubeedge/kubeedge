//go:build !windows

package constants

// Resources
const (
	// Certificates
	DefaultConfigDir = "/etc/kubeedge/config/"
	DefaultCAFile    = "/etc/kubeedge/ca/rootCA.crt"
	DefaultCAKeyFile = "/etc/kubeedge/ca/rootCA.key"
	DefaultCertFile  = "/etc/kubeedge/certs/server.crt"
	DefaultKeyFile   = "/etc/kubeedge/certs/server.key"

	DefaultStreamCAFile   = "/etc/kubeedge/ca/streamCA.crt"
	DefaultStreamCertFile = "/etc/kubeedge/certs/stream.crt"
	DefaultStreamKeyFile  = "/etc/kubeedge/certs/stream.key"

	DefaultMqttCAFile   = "/etc/kubeedge/ca/rootCA.crt"
	DefaultMqttCertFile = "/etc/kubeedge/certs/server.crt"
	DefaultMqttKeyFile  = "/etc/kubeedge/certs/server.key"

	// Bootstrap file, contains token used by edgecore to apply for ca/cert
	BootstrapFile = "/etc/kubeedge/bootstrap-edgecore.conf"

	// Edged
	DefaultRootDir               = "/var/lib/kubelet"
	DefaultRemoteRuntimeEndpoint = "unix:///run/containerd/containerd.sock"
	DefaultRemoteImageEndpoint   = "unix:///run/containerd/containerd.sock"
	DefaultCNIConfDir            = "/etc/cni/net.d"
	DefaultCNIBinDir             = "/opt/cni/bin"
	DefaultCNICacheDir           = "/var/lib/cni/cache"
	DefaultVolumePluginDir       = "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/"

	// DefaultManifestsDir edge node default static pod path
	DefaultManifestsDir = "/etc/kubeedge/manifests"
)
