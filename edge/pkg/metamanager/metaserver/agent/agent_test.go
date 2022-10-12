/*
Copyright 2021 The KubeEdge Authors.

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

package agent

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	commontypes "github.com/kubeedge/kubeedge/common/types"
)

func TestApplicationGC(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			"Test ApplicationGC Func",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Agent{nodeName: "test"}
			requestInfo := &apirequest.RequestInfo{
				IsResourceRequest: true,
				Verb:              "GET",
				Path:              "http://127.0.0.1:10550/api/v1/nodes",
				APIPrefix:         "api",
				APIGroup:          "",
				APIVersion:        "v1",
				Resource:          "nodes",
			}
			ctx := apirequest.WithRequestInfo(context.Background(), requestInfo)
			ctx = context.WithValue(ctx, commontypes.AuthorizationKey, "Bearer xxxx")

			app, _ := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
			app.Close()
			// make sure that the last closing time is more than 5 minutes from now
			app.Timestamp = time.Unix(1469579899, 0)
			a.GC()
			_, ok := a.Applications.Load(app.Identifier())
			if ok == true {
				t.Errorf("Application delete failed")
			}
		})
	}
}
