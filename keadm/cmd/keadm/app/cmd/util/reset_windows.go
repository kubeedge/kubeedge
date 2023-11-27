//go:build windows

package util

import "fmt"

const (
	dockerShimRootDir = "/var/lib/dockershim"
	kubernetesRunDir  = "/var/run/kubernetes"
	cniDir            = "/var/lib/cni"
)

var kubeEdgeDirs = []string{
	dockerShimRootDir,
	kubernetesRunDir,
	cniDir,
}

// TearDownKubeEdge will bring down edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(_ bool, _ string) error {
	// 1.1 stop check if running now, stop it if running
	if IsNSSMServiceRunning(KubeEdgeBinaryName) {
		fmt.Println("Egdecore service is running, stop...")
		if _err := StopNSSMService(KubeEdgeBinaryName); _err != nil {
			return _err
		}
		fmt.Println("Egdecore service stop success.")
	}

	// 1.2 remove nssm service
	fmt.Println("Start removing egdecore service using nssm")
	_err := UninstallNSSMService(KubeEdgeBinaryName)
	if _err != nil {
		return _err
	}
	fmt.Println("Egdecore service remove complete")
	return nil
}
