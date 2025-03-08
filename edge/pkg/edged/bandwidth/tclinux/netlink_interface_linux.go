package tclinux

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

const (
	Prefix          = "bwp"
	Length          = 15
	commandTemplate = "ip a |grep %s | awk '/mtu/ {print $5}'"
)

// key:podName,val:deviceName array
var podNetworkDeviceMapping = make(map[string][]string)

func StorePodNetworkDeviceMapping(podName, deviceName string) {
	// check the device exists in the current pod
	if _, ok := podNetworkDeviceMapping[podName]; ok {
		devices := podNetworkDeviceMapping[podName]
		if len(devices) == expectedDeviceCount {
			// If it is already included, there is no need to replace it.
			if containsNetlink(devices, deviceName) {
				klog.Infof("pod `%s` %s netlink has existed", podName, deviceName)
				return
			}
			klog.Warningf("pod `%s` has existed two netlink:%v,now replace as %s", podName, devices, deviceName)
			// replace network device
			index := getNetlinkIndex(deviceName, devices)
			podNetworkDeviceMapping[podName][index] = deviceName
			return
		}
		// check the current device is included
		if containsNetlink(devices, deviceName) {
			return
		}
		podNetworkDeviceMapping[podName] = append(podNetworkDeviceMapping[podName], deviceName)
	} else {
		// add device
		podNetworkDeviceMapping[podName] = []string{deviceName}
	}
}

func containsNetlink(devices []string, deviceName string) bool {
	if deviceName == "" || len(devices) == 0 {
		return false
	}
	for _, v := range devices {
		if strings.EqualFold(deviceName, v) {
			return true
		}
	}
	return false
}

func getNetlinkIndex(deviceName string, devices []string) int {
	if deviceName == "" || len(devices) == 0 {
		return 0
	}
	if strings.HasPrefix(deviceName, Prefix) {
		// get the subscript of ifb network interface
		for index, v := range devices {
			if strings.HasPrefix(v, Prefix) {
				return index
			}
		}
		return 1
	}
	return 0
}

// get pod host network interface vethxxx
func GetNetworkDeviceByPod(podName string) string {
	if v, ok := podNetworkDeviceMapping[podName]; ok {
		// check is a container host network interface
		if v, ok := isHostNetworkDeviceExist(v); ok {
			return v
		}
		return ""
	}
	return ""
}

func GetIfbDeviceByPod(podName string) string {
	if v, ok := podNetworkDeviceMapping[podName]; ok {
		// check is a container host network interface
		if name, flag := isIfbDeviceExist(v); flag {
			return name
		}
		return ""
	}
	return ""
}

func DeleteNetworkDevice(podName string) {
	if v, ok := podNetworkDeviceMapping[podName]; ok {
		for _, ifblink := range v {
			if strings.HasPrefix(ifblink, Prefix) {
				klog.Infof("delete pod `%s` ifb netlink interface:%s", podName, ifblink)
				// delete ifb interface
				err := DelLinkByName(ifblink)
				if err != nil {
					klog.Warningf("delete ifb netlink interface device failed,please delete it manual,ip "+
						"link del dev %s,err:%v", ifblink, err)
				}
			}
		}
		// delete the key->podName (cache)）
		delete(podNetworkDeviceMapping, podName)
		return
	}
	klog.Warningf("pod `%s` netlink interface device does not exist", podName)
}

// getMTU retrieves the MTU value based on the given interface name.
func GetMTU(interfaceName string) (int, error) {
	// Use the constant command template to format the command
	cmd := exec.Command("bash", "-c", fmt.Sprintf(commandTemplate, interfaceName))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // Capture stderr as well

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to run command: %s, output: %s", err, out.String())
	}

	// Trim whitespace and convert output to integer
	mtuStr := strings.TrimSpace(out.String())
	mtu, err := strconv.Atoi(mtuStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert MTU to int: %s", err)
	}

	return mtu, nil // Return the resulting MTU value as an integer
}

// calculate ifb device name to ensure standard and unique
func CalIfbName(containerIDOrName string) string {
	output := sha512.Sum512([]byte(containerIDOrName))
	return fmt.Sprintf("%s%x", Prefix, output)[:Length]
}

func isIfbDeviceExist(devices []string) (string, bool) {
	for _, v := range devices {
		if strings.HasPrefix(v, Prefix) {
			return v, true
		}
	}
	return "", false
}

func isHostNetworkDeviceExist(devices []string) (string, bool) {
	flag := false
	hostDevice := ""
	if len(devices) == 0 {
		return hostDevice, false
	}
	for _, v := range devices {
		if !strings.HasPrefix(v, Prefix) {
			flag = true
		}
	}
	if len(devices) == 1 && flag {
		return devices[0], true
	}
	if len(devices) == expectedDeviceCount && flag {
		// get ifb interface
		for _, v := range devices {
			if !strings.HasPrefix(v, Prefix) {
				hostDevice = v
				return hostDevice, flag
			}
		}
		return hostDevice, true
	}
	return hostDevice, flag
}
