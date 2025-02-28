package application

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	v1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	fakerest "k8s.io/client-go/rest/fake"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/config"
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

func TestCheckNodePermission(t *testing.T) {
	originalEnableAuthorization := config.Config.EnableAuthorization
	config.Config.EnableAuthorization = true
	defer func() {
		config.Config.EnableAuthorization = originalEnableAuthorization
	}()

	tests := []struct {
		name    string
		app     *metaserver.Application
		allowed bool
		err     error
		wantErr bool
	}{
		{
			name: "get version success",
			app: &metaserver.Application{
				Verb:        "get",
				Key:         "/version",
				Subresource: "",
				Nodename:    "test-node",
			},
			allowed: true,
			wantErr: false,
		}, {
			name: "get version success",
			app: &metaserver.Application{
				Verb:        "get",
				Key:         "/version",
				Subresource: "",
				Nodename:    "test-node",
			},
			allowed: true,
			err:     errors.New("permission denied"),
			wantErr: true,
		}, {
			name: "get configmap failed",
			app: &metaserver.Application{
				Verb:        "get",
				Key:         "/core/v1/configmaps/ns/test-cm",
				Subresource: "",
				Nodename:    "test-node",
			},
			allowed: false,
			wantErr: true,
		},
	}

	fakeClientSet := fake.NewSimpleClientset()
	center := &Center{kubeClient: fakeClientSet}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientSet.PrependReactor("create", "subjectaccessreviews", func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &v1.SubjectAccessReview{Status: v1.SubjectAccessReviewStatus{Allowed: tt.allowed}}, tt.err
			})

			err := center.checkNodePermission(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkNodePermission() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
