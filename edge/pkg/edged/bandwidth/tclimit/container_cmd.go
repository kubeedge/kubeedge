/*
Copyright 2024 The KubeEdge Authors.

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

package tclimit

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"k8s.io/klog/v2"
)

const (
	MinMatches        = 2
	VethInterfaceName = "ip link show | grep %s: | grep veth | awk '{print $2}' | tr -d ':' | awk -F" +
		"'@' '{print $1}'\n"
)

func GetNetlinkDeviceName(containerID string) (string, error) {
	ifaceName, err := getInterfaceName(containerID)
	if err != nil {
		klog.Warningf("getInterfaceName Error: %v", err)
		return "", err
	}

	ifaceNumber, err := extractInterfaceNumber(ifaceName)
	if err != nil {
		klog.Warningf("extractInterfaceNumber Error: %v", err)
		return "", err
	}

	hostVethName, err := getHostVethInterfaceName(ifaceNumber)
	if err != nil {
		klog.Warningf("getInterfaceName Error:%v", err)
		return "", err
	}
	klog.Infof("ifaceName:%s", hostVethName)

	return hostVethName, nil
}

// getInterfaceName executes an inline shell script to get the network interface name.
func getInterfaceName(containerID string) (string, error) {
	script := fmt.Sprintf(`
#!/bin/bash

# 获取 PID
CONTAINER_ID="%s"
PID=$(ctr -n k8s.io tasks ls | grep $CONTAINER_ID | awk '{print $2}' | tr -d '\n')

# 进入网络命名空间并提取 eth0@if
IFACE_NAME=$(nsenter -t $PID -n ip a | grep 'eth0@if' | awk '{print $2}' | tr -d '\n')

# 输出接口名称
echo "$IFACE_NAME"
`, containerID)

	cmd := exec.Command("bash", "-c", script)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run script: %s, output: %s", err, out.String())
	}

	return out.String(), nil
}

// extractInterfaceNumber extracts the number between 'if' and ':' from the input string.
func extractInterfaceNumber(iface string) (string, error) {
	// Define a regex pattern to match the desired part of the string
	re := regexp.MustCompile(`if(\d+):`)
	matches := re.FindStringSubmatch(iface)

	if len(matches) < MinMatches {
		return "", fmt.Errorf("no match found")
	}

	return matches[1], nil // Return the first capturing group which contains the number
}

// getInterfaceName retrieves the interface name based on the given identifier.
func getHostVethInterfaceName(identifier string) (string, error) {
	// Use the constant command template to format the command
	cmd := exec.Command("bash", "-c", fmt.Sprintf(VethInterfaceName, identifier))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // Capture stderr as well

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run command: %s, output: %s", err, out.String())
	}

	return strings.TrimSpace(out.String()), nil // Return the resulting interface name
}
