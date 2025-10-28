/*
Copyright 2025 The KubeEdge Authors.

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

package clouddatastream

import (
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/clouddatastream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
)

type cloudDataStream struct {
	enable bool

	tunnelPort int
}

var _ core.Module = (*cloudDataStream)(nil)

func newCloudDataStream(enable bool, tunnelPort int) *cloudDataStream {
	return &cloudDataStream{
		enable: enable,

		tunnelPort: tunnelPort,
	}
}

func Register(controller *v1alpha1.CloudDataStream, commonConfig *v1alpha1.CommonConfig) {
	config.InitConfigure(controller)
	core.Register(newCloudDataStream(controller.Enable, commonConfig.TunnelPort))
}

func (s *cloudDataStream) Name() string {
	return modules.CloudDataStreamModuleName
}

func (s *cloudDataStream) Group() string {
	return modules.CloudDataStreamGroupName
}

func (s *cloudDataStream) Start() {
	ok := <-cloudhub.DoneTLSTunnelCerts

	if ok {
		ts := newTunnelServer(s.tunnelPort)

		go ts.Start()

		server := newStreamServer(ts)
		go server.Start()
	}
}

func (s *cloudDataStream) Enable() bool {
	return s.enable
}
