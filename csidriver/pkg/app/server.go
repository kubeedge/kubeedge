package app

import (
	"fmt"

	"k8s.io/klog"
)

type CSIDriver struct {
	name       string
	nodeID     string
	version    string
	endpoint   string
	keEndpoint string

	ids *identityServer
	cs  *controllerServer
}

var (
	vendorVersion = "dev"
)

// NewCSIDriver creates a new server
func NewCSIDriver(driverName, nodeID, endpoint, keEndpoint, version string) (*CSIDriver, error) {
	if driverName == "" {
		return nil, fmt.Errorf("no driver name provided")
	}
	if nodeID == "" {
		return nil, fmt.Errorf("no node id provided")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("no driver endpoint provided")
	}
	if keEndpoint == "" {
		return nil, fmt.Errorf("no kubeedge endpoint provided")
	}
	if version != "" {
		vendorVersion = version
	}
	klog.Infof("driver: %s version: %s", driverName, vendorVersion)

	return &CSIDriver{
		name:       driverName,
		version:    vendorVersion,
		nodeID:     nodeID,
		endpoint:   endpoint,
		keEndpoint: keEndpoint,
	}, nil
}

func (cd *CSIDriver) Run() {
	// Create GRPC servers
	cd.ids = NewIdentityServer(cd.name, cd.version)
	cd.cs = NewControllerServer(cd.nodeID, cd.keEndpoint)

	s := NewNonBlockingGRPCServer()
	s.Start(cd.endpoint, cd.ids, cd.cs, nil)
	s.Wait()
}
