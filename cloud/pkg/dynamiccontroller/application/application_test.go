package application

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	fakerest "k8s.io/client-go/rest/fake"

	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

func TestCenter_passThroughRequest(t *testing.T) {
	failureResp := &http.Response{
		Status:     "500 Internal Error",
		StatusCode: http.StatusInternalServerError,
	}
	successResp := &http.Response{
		Status:     "200 ok",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{version: 1.27}")),
	}
	getVersions := func(key, verb string) *fakerest.RESTClient {
		if key == "/version" && verb == "get" {
			return &fakerest.RESTClient{
				Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
					return successResp, nil
				}),
				NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
			}
		}
		return &fakerest.RESTClient{
			Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
				return failureResp, nil
			}),
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		}
	}

	tests := []struct {
		name    string
		app     *metaserver.Application
		want    interface{}
		wantErr bool
	}{
		{
			name: "get version success",
			app: &metaserver.Application{
				Key:  "/version",
				Verb: "get",
			},
			want:    []byte("{version: 1.27}"),
			wantErr: false,
		}, {
			name: "pass through failed",
			app: &metaserver.Application{
				Key:  "/healthz",
				Verb: "get",
			},
			want:    []byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			center := &Center{
				kubeClient: &kubernetes.Clientset{
					DiscoveryClient: discovery.NewDiscoveryClient(getVersions(tt.app.Key, string(tt.app.Verb))),
				},
			}
			got, err := center.passThroughRequest(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("passThroughRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("passThroughRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
