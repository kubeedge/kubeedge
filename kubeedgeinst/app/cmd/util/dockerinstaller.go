package util

import "fmt"

type DockerInstTool struct {
	Common
	DefaultToolVer string
}

func (d *DockerInstTool) InstallTools() error {
	d.SetOSInterface(GetOSInterface())
	d.SetDockerVersion(d.ToolVersion)

	action, err := d.IsDockerInstalled(d.DefaultToolVer)
	if err != nil {
		return err
	}
	switch action {
	case VersionNAInRepo:
		return fmt.Errorf("Expected Docker version is not available in OS repo")
	case AlreadySameVersionExist:
		return fmt.Errorf("Same version of docker already installed in this host")
	case DefVerInstallRequired:
		fmt.Println("Installing default", d.DefaultToolVer, "version of docker")
		d.SetDockerVersion(d.DefaultToolVer)
		fallthrough
	case NewInstallRequired:
		err := d.InstallDocker()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error in getting the docker version from host")
	}

	return nil
}
