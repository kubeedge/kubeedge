//go:build windows

package constants

// Resources
const (
	KubeEdgePath       = "C:\\etc\\kubeedge\\"
	KubeEdgeUsrBinPath = "C:\\usr\\local\\bin"

	DefaultConfigDir   = "C:\\etc\\kubeedge\\config\\"
	EdgecoreConfigPath = "C:\\etc\\kubeedge\\config\\edgecore.yaml"

	KubeEdgeLogPath = "C:\\var\\log\\kubeedge\\"

	// Certificates
	// DefaultCertPath is the default certificate path in edge node
	DefaultCertPath  = "C:\\etc\\kubeedge\\certs"
	DefaultCAFile    = "C:\\etc\\kubeedge\\ca\\rootCA.crt"
	DefaultCAKeyFile = "C:\\etc\\kubeedge\\ca\\rootCA.key"
	DefaultCertFile  = "C:\\etc\\kubeedge\\certs\\server.crt"
	DefaultKeyFile   = "C:\\etc\\kubeedge\\certs\\server.key"

	DefaultStreamCAFile   = "C:\\etc\\kubeedge\\ca\\streamCA.crt"
	DefaultStreamCertFile = "C:\\etc\\kubeedge\\certs\\stream.crt"
	DefaultStreamKeyFile  = "C:\\etc\\kubeedge\\certs\\stream.key"

	DefaultMqttCAFile   = "C:\\etc\\kubeedge\\ca\\rootCA.crt"
	DefaultMqttCertFile = "C:\\etc\\kubeedge\\certs\\server.crt"
	DefaultMqttKeyFile  = "C:\\etc\\kubeedge\\certs\\server.key"

	// Bootstrap file, contains token used by edgecore to apply for ca/cert
	BootstrapFile = "C:\\etc\\kubeedge\\bootstrap-edgecore.conf"

	// Edged
	DefaultRootDir               = "C:\\var\\lib\\kubelet"
	DefaultRemoteRuntimeEndpoint = "npipe://./pipe/containerd-containerd"
	DefaultRemoteImageEndpoint   = "npipe://./pipe/containerd-containerd"
	DefaultCNIConfDir            = "C:\\etc\\cni\\net.d"
	DefaultCNIBinDir             = "C:\\opt\\cni\\bin"
	DefaultCNICacheDir           = "C:\\var\\lib\\cni\\cache"
	DefaultVolumePluginDir       = "C:\\usr\\libexec\\kubernetes\\kubelet-plugins\\volume\\exec\\"

	// DefaultManifestsDir edge node default static pod path
	DefaultManifestsDir = "C:\\etc\\kubeedge\\manifests\\"
)
