//go:build !windows

package util

import (
	"fmt"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
)

var kubeEdgeDirs = []string{
	KubeEdgeUsrBinPath,
}

// tearDownKubeEdge will bring down either cloud or edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, kubeConfig string) error {
	var ke common.ToolsInstaller
	ke = &helm.KubeCloudHelmInstTool{
		Common: Common{
			KubeConfig: kubeConfig,
		},
	}
	if isEdgeNode {
		ke = &KubeEdgeInstTool{Common: util.Common{}}
	}

	err := ke.TearDown()
	if err != nil {
		return fmt.Errorf("tearDown failed, err:%v", err)
	}
	return nil
}
