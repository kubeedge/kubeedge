//go:build windows

package dmiserver

import (
	"fmt"
	"os"

	"k8s.io/klog/v2"
)

func initSock(sockPath string) error {
	klog.Infof("init uds socket: %s", sockPath)
	err := os.Remove(sockPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		klog.Error(err)
		return fmt.Errorf("fail to stat uds socket path")
	}
	return nil
}
