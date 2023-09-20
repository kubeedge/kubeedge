//go:build !windows

package dmiserver

import (
	"fmt"
	"k8s.io/klog/v2"
	"os"
)

func initSock(sockPath string) error {
	klog.Infof("init uds socket: %s", sockPath)
	_, err := os.Stat(sockPath)
	if err == nil {
		err = os.Remove(sockPath)
		if err != nil {
			return err
		}
		return nil
	} else if os.IsNotExist(err) {
		return nil
	} else {
		return fmt.Errorf("fail to stat uds socket path")
	}
}
