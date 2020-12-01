/*
Copyright 2020 The KubeEdge Authors.

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

package cloudstream

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type cloudStream struct {
	enable bool
}

func newCloudStream(enable bool) *cloudStream {
	return &cloudStream{
		enable: enable,
	}
}

func Register(controller *v1alpha1.CloudStream) {
	config.InitConfigure(controller)
	core.Register(newCloudStream(controller.Enable))
}

func (s *cloudStream) Name() string {
	return modules.CloudStreamModuleName
}

func (s *cloudStream) Group() string {
	return modules.CloudStreamGroupName
}

func (s *cloudStream) Start() {
	// TODO: Will improve in the future
	ok := <-cloudhub.DoneTLSTunnelCerts
	if ok {
		ts := newTunnelServer()

		// start new tunnel server
		go ts.Start()

		server := newStreamServer(ts)
		// start stream server to accept kube-apiserver connection
		go server.Start()
	}
}

func (s *cloudStream) Enable() bool {
	return s.enable
}
