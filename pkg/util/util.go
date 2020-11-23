/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
)

func GetLocalIP(hostName string) (string, error) {
	var ipAddr net.IP
	var err error
	addrs, _ := net.LookupIP(hostName)
	for _, addr := range addrs {
		if err := ValidateNodeIP(addr); err == nil {
			if addr.To4() != nil {
				ipAddr = addr
				break
			}
			if addr.To16() != nil && ipAddr == nil {
				ipAddr = addr
			}
		}
	}
	if ipAddr == nil {
		ipAddr, err = utilnet.ChooseHostInterface()
	}

	if err != nil {
		return "", err
	}

	return ipAddr.String(), nil
}

// ValidateNodeIP validates given node IP belongs to the current host
func ValidateNodeIP(nodeIP net.IP) error {
	// Honor IP limitations set in setNodeStatus()
	if nodeIP.To4() == nil && nodeIP.To16() == nil {
		return fmt.Errorf("nodeIP must be a valid IP address")
	}
	if nodeIP.IsLoopback() {
		return fmt.Errorf("nodeIP can't be loopback address")
	}
	if nodeIP.IsMulticast() {
		return fmt.Errorf("nodeIP can't be a multicast address")
	}
	if nodeIP.IsLinkLocalUnicast() {
		return fmt.Errorf("nodeIP can't be a link-local unicast address")
	}
	if nodeIP.IsUnspecified() {
		return fmt.Errorf("nodeIP can't be an all zeros address")
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil && ip.Equal(nodeIP) {
			return nil
		}
	}
	return fmt.Errorf("Node IP: %q not found in the host's network interfaces", nodeIP.String())
}

//Command executes command and returns output
func Command(name string, arg []string) (string, error) {
	cmd := exec.Command(name, arg...)
	ret, err := cmd.Output()
	if err != nil {
		klog.Errorf("exec command failed: %v", err)
		return string(ret), err
	}
	return strings.Trim(string(ret), "\n"), nil
}

//GetCurPath returns filepath
func GetCurPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	rst := filepath.Dir(path)
	return rst
}

func SpliceErrors(errors []error) string {
	if len(errors) == 0 {
		return ""
	}
	var stb strings.Builder
	stb.WriteString("[\n")
	for _, err := range errors {
		stb.WriteString(fmt.Sprintf("  %s\n", err.Error()))
	}
	stb.WriteString("]\n")
	return stb.String()
}

// GetPodSandboxImage return snadbox image name based on arch, default image is for amd64.
func GetPodSandboxImage() string {
	switch runtime.GOARCH {
	case "arm":
		return constants.DefaultArmPodSandboxImage
	case "arm64":
		return constants.DefaultArm64PodSandboxImage
	default:
		return constants.DefaultPodSandboxImage
	}
}
