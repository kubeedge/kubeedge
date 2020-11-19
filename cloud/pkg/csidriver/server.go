/*
Copyright 2019 The KubeEdge Authors.

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

package csidriver

import (
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/cmd/csidriver/app/options"
)

type CSIDriver struct {
	options.CSIDriverOptions

	ids *identityServer
	cs  *controllerServer
}

func NewCSIDriver(opts *options.CSIDriverOptions) (*CSIDriver, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("no driver endpoint provided")
	}
	if opts.DriverName == "" {
		return nil, fmt.Errorf("no driver name provided")
	}
	if opts.NodeID == "" {
		return nil, fmt.Errorf("no node id provided")
	}
	if opts.KubeEdgeEndpoint == "" {
		return nil, fmt.Errorf("no kubeedge endpoint provided")
	}
	if opts.Version == "" {
		return nil, fmt.Errorf("no version provided")
	}
	return &CSIDriver{
		CSIDriverOptions: *opts,
	}, nil
}

func (cd *CSIDriver) Run() {
	klog.Infof("driver information: %v", cd)

	// Create GRPC servers
	cd.ids = newIdentityServer(cd.DriverName, cd.Version)
	cd.cs = newControllerServer(cd.NodeID, cd.KubeEdgeEndpoint)

	s := newNonBlockingGRPCServer()
	s.Start(cd.Endpoint, cd.ids, cd.cs, nil)
	s.Wait()
}
