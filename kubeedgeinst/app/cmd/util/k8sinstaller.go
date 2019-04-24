package util

import "fmt"

type K8SInstTool struct {
	Common
	IsEdgeNode     bool //True - Edgenode False - Cloudnode
	DefaultToolVer string
}

func (ks *K8SInstTool) InstallTools() error {
	ks.SetOSInterface(GetOSInterface())
	ks.SetK8SVersionAndIsNodeFlag(ks.ToolVersion, ks.IsEdgeNode)

	component := "kubeadm"
	if ks.IsEdgeNode == true {
		component = "kubectl"
	}
	action, err := ks.IsK8SComponentInstalled(component, ks.DefaultToolVer)
	if err != nil {
		return err
	}
	switch action {
	case VersionNAInRepo:
		return fmt.Errorf("Expected %s version is not available in OS repo", component)
	case AlreadySameVersionExist:
		return fmt.Errorf("Same version of %s already installed in this host", component)
	case DefVerInstallRequired:
		fmt.Println("Installing default", ks.DefaultToolVer, "version of", component)
		ks.SetK8SVersionAndIsNodeFlag(ks.DefaultToolVer, ks.IsEdgeNode)
		fallthrough
	case NewInstallRequired:
		err := ks.InstallK8S()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error in getting the %s version from host", component)
	}
	return nil
}
