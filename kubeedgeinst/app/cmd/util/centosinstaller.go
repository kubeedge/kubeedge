package util

import "fmt"

type CentOS struct {
	DockerVersion     string
	KubernetesVersion string
	KubeEdgeVersion   string
	IsEdgeNode        bool //True - Edgenode False - Cloudnode
}

func (c *CentOS) SetDockerVersion(version string) {
	c.DockerVersion = version
}

func (c *CentOS) SetK8SVersionAndIsNodeFlag(version string, flag bool) {
	c.KubernetesVersion = version
	c.IsEdgeNode = flag
}

func (c *CentOS) SetKubeEdgeVersion(version string) {
	c.KubeEdgeVersion = version
}

func (c *CentOS) IsDockerInstalled(string) (InstallState, error) {

	return VersionNAInRepo, nil
}

func (c *CentOS) InstallDocker() error {
	fmt.Println("InstallDocker called")
	return nil
}

func (c *CentOS) IsToolVerInRepo(toolName, version string) (bool, error) {
	fmt.Println("IsToolVerInRepo called")
	return false, nil
}

func (c *CentOS) InstallMQTT() error {
	fmt.Println("InstallMQTT called")
	return nil
}

func (c *CentOS) IsK8SComponentInstalled(component, defVersion string) (InstallState, error) {
	return VersionNAInRepo, nil
}

func (c *CentOS) InstallK8S() error {
	fmt.Println("InstallK8S called")
	return nil
}

func (c *CentOS) InstallKubeEdge() error {
	fmt.Println("InstallKubeEdge called")
	return nil
}

func (c *CentOS) RunEdgeCore() error {
	return nil
}
