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

func (c *CentOS) SetK8SVersion(version string) {
	c.KubernetesVersion = version
}

func (c *CentOS) SetKubeEdgeVersion(version string) {
	c.KubeEdgeVersion = version
}

func (c *CentOS) IsDockerInstalled(string) string {

	return "18.06"
}

func (c *CentOS) InstallDocker() error {
	fmt.Println("InstallDocker called")
	return nil
}

func (c *CentOS) IsDockerVerInRepo(version string) (bool, error) {
	fmt.Println("IsDockerVerInRepo called")
	return false, nil
}

func (c *CentOS) InstallMQTT() error {
	fmt.Println("InstallMQTT called")
	return nil
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
