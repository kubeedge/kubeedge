package tclinux

// tc related functions only take effect under the linux system, and tc related functions are disguised under the windows system

func DeleteNetworkDevice(podName string) {
}

func CreateIngressQdisc(rateInBits, burstInBits uint64, hostDeviceName string) error {
	return nil
}

func StorePodNetworkDeviceMapping(podName, deviceName string) {
}

func GetMTU(deviceName string) (int, error) {
	return -1, nil
}

func GetIfbDeviceByPod(podName string) string {
	return ""
}

func CalIfbName(containerIDOrName string) string {
	return ""
}

func CreateIfb(ifbDeviceName string, mtu int) error {
	return nil
}

func CreateEgressQdisc(rateInBits, burstInBits uint64, hostDeviceName, ifbDeviceName string) error {
	return nil
}

func GetNetworkDeviceByPod(podName string) string {
	return ""
}
