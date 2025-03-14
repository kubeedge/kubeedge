package tclinux

// tc related functions only take effect under the linux system, and tc related functions are disguised under the mac system

func DeleteNetworkDevice(_ string) {
}

func CreateIngressQdisc(_, _ uint64, _ string) error {
	return nil
}

func StorePodNetworkDeviceMapping(_, _ string) {
}

func GetMTU(_ string) (int, error) {
	return -1, nil
}

func GetIfbDeviceByPod(_ string) string {
	return ""
}

func CalIfbName(_ string) string {
	return ""
}

func CreateIfb(_ string, _ int) error {
	return nil
}

func CreateEgressQdisc(_, _ uint64, _, _ string) error {
	return nil
}

func GetNetworkDeviceByPod(_ string) string {
	return ""
}
