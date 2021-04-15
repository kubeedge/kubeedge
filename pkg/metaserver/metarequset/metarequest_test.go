package metarequset

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"
)

type partReqInfo struct {
	Path     string
	Resource string
	Name     string
}

func TestDecorateMetaRequest(t *testing.T) {
	baseReqInfo := &apirequest.RequestInfo{
		IsResourceRequest: true,
		Verb:              "get",
		APIPrefix:         "apis",
		APIGroup:          MetaGroup,
		APIVersion:        MetaVersion,
	}
	Cases := []struct {
		Path string
		Want partReqInfo
	}{
		{
			"/api",
			partReqInfo{
				"/apis/meta/v1/" + APIVersionsResource + "/" + DefaultAPIVersionsObjName,
				APIVersionsResource,
				DefaultAPIVersionsObjName,
			},
		},
		{
			"/apis",
			partReqInfo{
				"/apis/meta/v1/" + APIGroupListResource + "/" + DefaultAPIGroupListObjName,
				APIGroupListResource,
				DefaultAPIGroupListObjName,
			},
		},
		{
			"/apis/apps",
			partReqInfo{
				"/apis/meta/v1/" + APIGroupResource + "/" + "apps",
				APIGroupResource,
				"apps",
			},
		},
		{
			"/apis/apps/v1",
			partReqInfo{
				"/apis/meta/v1/" + APIResourceListResource + "/" + "apps---v1",
				APIResourceListResource,
				"apps---v1",
			},
		},
		{
			"/api/v1",
			partReqInfo{
				"/apis/meta/v1/" + APIResourceListResource + "/" + "---v1",
				APIResourceListResource,
				"---v1",
			},
		},
		{
			"/apis/apps///////",
			partReqInfo{
				"/apis/meta/v1/" + APIGroupResource + "/" + "apps",
				APIGroupResource,
				"apps",
			},
		},
	}
	for _, test := range Cases {
		t.Run("DecorateMetaRequest", func(t *testing.T) {
			var exportError error
			baseCheckHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				exportError = nil
				reqInfo, ok := apirequest.RequestInfoFrom(req.Context())
				if !ok {
					exportError = fmt.Errorf("failed to get req info from req ctx")
					return
				}
				stdReqInfo := baseReqInfo
				stdReqInfo.Path = test.Want.Path
				stdReqInfo.Resource = test.Want.Resource
				stdReqInfo.Name = test.Want.Name
				if !reflect.DeepEqual(stdReqInfo, reqInfo) {
					exportError = fmt.Errorf("failed to get wanted req info:\nwant(%+v)\n get(%+v)", stdReqInfo, reqInfo)
				}
			})
			req := &http.Request{}
			req = req.WithContext(apirequest.WithRequestInfo(req.Context(), &apirequest.RequestInfo{Path: test.Path}))
			DecorateMetaRequest(baseCheckHandler).ServeHTTP(nil, req)
			if exportError != nil {
				t.Errorf("DecorateMetaRequest(%v) Error, %v", test.Path, exportError)
			}
		})
	}
}
