package metarequset

import (
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
)

const (
	MetaGroup           = "meta"
	MetaVersion         = "v1"
	resourceListNameSep = "---" //not export

	APIVersionsKind     = "APIVersions"
	APIGroupListKind    = "APIGroupList"
	APIGroupKind        = "APIGroup"
	APIResourceListKind = "APIResourceList"

	APIVersionsResource     = "apiversions"
	APIGroupListResource    = "apigrouplists"
	APIGroupResource        = "apigroups"
	APIResourceListResource = "apiresourcelists"

	DefaultAPIVersionsObjName  = "cloud-apiversions"
	DefaultAPIGroupListObjName = "cloud-apigrouplist"
)

var (
	MetaGroupVersion = schema.GroupVersion{Group: MetaGroup, Version: MetaVersion}
	APIVersions      = MetaGroupVersion.WithResource(APIVersionsResource)
	APIGroupLists    = MetaGroupVersion.WithResource(APIGroupListResource)
	APIGroups        = MetaGroupVersion.WithResource(APIGroupResource)
	APIResourceLists = MetaGroupVersion.WithResource(APIResourceListResource)
)

// DecorateMetaRequest check if the request is meta request and transform it to normal get request.
// meta request here is that wants to get resources which belongs to Group "meta", and it's request
// path is abnormal:
// |     Path     |              NewGetPath                    |         Want Resources        |
// |--------------|--------------------------------------------|-------------------------------|
// |/api          |/apis/meta/v1/apiversions/cloud-apiversions |APIVersions      only one      |
// |/apis         |/apis/meta/v1/apigrouplists/cloud-apigroups |APIGroupList     only one      |
// |/apis/{g}     |/apis/meta/v1/apigroups/{g}                 |APIGroup                       |
// |/apis/{g}/{v} |/apis/meta/v1/apiresourcelists/{g}{SEP}{v}  |APIResourcesList below {g}/{v} |
// |/api/v1       |/apis/meta/v1/apiresourcelists/---v1        |APIResourcesList below /v1     |
// {g}={group}, {v}={version}, {SEP} = resourceListNameSep
// "only one" means there is only one obj of this kind resource in cluster-wide
func DecorateMetaRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		reqInfo, ok := apirequest.RequestInfoFrom(ctx)
		if ok {
			if gvr, name, ok := isMetaRequest(reqInfo); ok {
				// transform to normal get request
				reqInfo.IsResourceRequest = true
				reqInfo.Verb = "get"
				reqInfo.APIPrefix = "apis"
				reqInfo.APIGroup = gvr.Group
				reqInfo.APIVersion = gvr.Version
				reqInfo.Resource = gvr.Resource
				reqInfo.Path = "/apis/meta/v1/" + gvr.Resource + "/" + name
				reqInfo.Name = name
				req = req.WithContext(apirequest.WithRequestInfo(req.Context(), reqInfo))
			}
		}
		handler.ServeHTTP(w, req)
	})
}

func isMetaRequest(info *apirequest.RequestInfo) (gvr schema.GroupVersionResource, name string, ok bool) {
	switch info.Path {
	case "/api":
		return APIVersions, DefaultAPIVersionsObjName, true
	case "/apis":
		return APIGroupLists, DefaultAPIGroupListObjName, true
	case "/api/v1":
		gv := schema.GroupVersion{Group: "", Version: "v1"}
		return APIResourceLists, ResourceListNameParser.New(gv), true
	}
	if strings.HasPrefix(info.Path, "/apis") {
		path := strings.TrimRight(info.Path, "/")
		slice := strings.Split(path, "/")
		switch len(slice) {
		case 3: // /apis/group
			groupIndex := 2
			return APIGroups, slice[groupIndex], true
		case 4: // /apis/group/version
			groupIndex, versionIndex := 2, 3
			gv := schema.GroupVersion{Group: slice[groupIndex], Version: slice[versionIndex]}
			return APIResourceLists, ResourceListNameParser.New(gv), true
		}
	}
	return schema.GroupVersionResource{}, "", false
}

// ResourceListName contains information of group and version, which is uesd
// to construct request's path to API Server.
type resourceListNameParser struct{}

var ResourceListNameParser = resourceListNameParser{}

// New construct a resouceListName according to given group/version,
// format is "{Group}{SEP}{Version}"
func (p *resourceListNameParser) New(gv schema.GroupVersion) string {
	return gv.Group + resourceListNameSep + gv.Version
}

// Parse parse out group/version according to given name and format, remember
// it fails if group or version contains resourceListNameSep.
func (p *resourceListNameParser) Parse(name string) schema.GroupVersion {
	slice := strings.Split(name, resourceListNameSep)
	if len(slice) != 2 { // group/version
		return schema.GroupVersion{}
	}
	groupIndex := 0
	versionIndex := 1
	return schema.GroupVersion{Group: slice[groupIndex], Version: slice[versionIndex]}
}

// meta api miss information of name, it prevents key generation for meta api obj
// so add this kind of information here.
type NamedAPIVersions struct {
	metav1.APIVersions `json:",inline"`
	metav1.ObjectMeta  `json:"metadata,omitempty"`
}
type NamedAPIGroupList struct {
	metav1.APIGroupList `json:",inline"`
	metav1.ObjectMeta   `json:"metadata,omitempty"`
}
type NamedAPIGroup struct {
	metav1.APIGroup   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}
type NamedAPIResourceList struct {
	metav1.APIResourceList `json:",inline"`
	metav1.ObjectMeta      `json:"metadata,omitempty"`
}
