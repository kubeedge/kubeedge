package scope

import (
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
		Convertor:       &fakeObjectConvertor{},
		Defaulter:       nil,
		Typer:           nil,
		UnsafeConvertor: nil,
		Authorizer:      nil,

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

type fakeObjectConvertor struct{}

func (c *fakeObjectConvertor) Convert(in, out, context interface{}) error {
	return nil
}

func (c *fakeObjectConvertor) ConvertToVersion(in runtime.Object, _ runtime.GroupVersioner) (runtime.Object, error) {
	return in, nil
}

func (c *fakeObjectConvertor) ConvertFieldLabel(_ schema.GroupVersionKind, label, field string) (string, string, error) {
	return label, field, nil
}
