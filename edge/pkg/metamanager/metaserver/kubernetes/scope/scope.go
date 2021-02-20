package scope

import (
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/fakers"
	"k8s.io/apiextensions-apiserver/pkg/crdserverscheme"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/serializer"
)

func NewRequestScope() *handlers.RequestScope {
	requestScope := handlers.RequestScope{
		Namer: handlers.ContextBasedNaming{
			SelfLinker:         meta.NewAccessor(),
			ClusterScoped:      false,
			SelfLinkPathPrefix: "",
			SelfLinkPathSuffix: "",
		},

		Serializer:     serializer.NewNegotiatedSerializer(),
		ParameterCodec: scheme.ParameterCodec,
		//Creater:         nil,
		Convertor: fakers.NewFakeObjectConvertor(),
		Defaulter: fakers.NewFakeObjectDefaulter(),
		Typer:     crdserverscheme.NewUnstructuredObjectTyper(),
		//UnsafeConvertor: nil,
		Authorizer: fakers.NewAlwaysAllowAuthorizer(),

		EquivalentResourceMapper: runtime.NewEquivalentResourceRegistry(),

		TableConvertor: nil,

		Resource:    schema.GroupVersionResource{},
		Subresource: "",
		Kind:        schema.GroupVersionKind{},

		HubGroupVersion: schema.GroupVersion{},

		MetaGroupVersion: metav1.SchemeGroupVersion,

		MaxRequestBodyBytes: int64(3 * 1024 * 1024),
	}
	return &requestScope
}
