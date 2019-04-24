package util

import "fmt"

type DockerInstTool struct {
	Common
	DefaultToolVer string
}

func (d *DockerInstTool) InstallTools() error {
	d.SetOSInterface(GetOSInterface())
	d.SetDockerVersion(d.ToolVersion)

	switch d.IsDockerInstalled(d.DefaultToolVer) {
	case "Unavailable":
		return fmt.Errorf("Expected Docker versions are not available in OS repo")
	case "Same Version Docker":
		return fmt.Errorf("Same version docker already installed in this host")
	case "Install Default Docker":
		d.SetDockerVersion(d.ToolVersion)
		fallthrough
	case "Install Required Docker":
		err := d.InstallDocker()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Error in getting the docker version from host")
	}

	return nil
}
