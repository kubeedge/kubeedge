//go:build linux

package metaserver

import (
	"net"
	"os"

	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/pkg/util"
)

func setupDummyInterface() error {
	dummyIP, dummyPort, err := net.SplitHostPort(metaserverconfig.Config.DummyServer)
	if err != nil {
		return err
	}

	if err := os.Setenv("METASERVER_DUMMY_IP", dummyIP); err != nil {
		return err
	}
	if err := os.Setenv("METASERVER_DUMMY_PORT", dummyPort); err != nil {
		return err
	}

	manager := util.NewDummyDeviceManager()
	_, err = manager.EnsureDummyDevice("edge-dummy0")
	if err != nil {
		return err
	}

	_, err = manager.EnsureAddressBind(dummyIP, "edge-dummy0")
	return err
}
