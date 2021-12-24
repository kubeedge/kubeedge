package application

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
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
			a := Agent{nodeName: metaserverconfig.Config.NodeName}
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

			app := a.Generate(ctx, "get", metav1.GetOptions{}, nil)
			app.countLock.Lock()
			app.count = 0
			app.countLock.Unlock()
			// make sure that the last closing time is more than 5 minutes from now
			app.timestamp = time.Unix(1469579899, 0)
			a.GC()
			_, ok := a.Applications.Load(app.Identifier())
			if ok == true {
				t.Errorf("Application delete failed")
			}
		})
	}
}
