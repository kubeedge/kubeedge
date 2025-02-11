package deviceplugin

import "github.com/onsi/ginkgo/v2"

// GroupDescribe annotates the test with the group label.
func GroupDescribe(text string, body func()) bool {
	return ginkgo.Describe("[KubeEdge-DevicePlugin] "+text, body)
}