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

package options

import (
	cliflag "k8s.io/component-base/cli/flag"
)

// CSIDriverOptions config
type CSIDriverOptions struct {
	Endpoint         string
	DriverName       string
	NodeID           string
	KubeEdgeEndpoint string
	Version          string
}

// NewCSIDriverOptions returns options object
func NewCSIDriverOptions() *CSIDriverOptions {
	return &CSIDriverOptions{}
}

// Flags return flag sets
func (o *CSIDriverOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("csidriver")
	fs.StringVar(&o.Endpoint, "endpoint", "unix:///csi/csi.sock", "CSI endpoint")
	fs.StringVar(&o.DriverName, "drivername", "csidriver", "name of the driver")
	fs.StringVar(&o.NodeID, "nodeid", "", "node id determines which node will be used to create/delete volumes")
	fs.StringVar(&o.KubeEdgeEndpoint, "kubeedge-endpoint", "unix:///kubeedge/kubeedge.sock", "kubeedge endpoint")
	fs.StringVar(&o.Version, "version", "dev", "version")
	return
}
