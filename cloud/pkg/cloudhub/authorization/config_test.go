/*
Copyright 2024 The KubeEdge Authors.

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

package authorization

import (
	"testing"

	"k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

func TestNewAuthorizer(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "all supported modes",
			config: Config{
				AuthorizationModes:       []string{constants.ModeNode, constants.ModeAlwaysAllow, constants.ModeAlwaysDeny},
				VersionedInformerFactory: informers.NewFakeInformerManager().GetKubeInformerFactory(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.New()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("New(): unexpect error: %v", err)
				}
				return
			}
		})
	}
}
