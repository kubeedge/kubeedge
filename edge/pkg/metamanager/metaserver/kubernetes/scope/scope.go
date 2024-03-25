package scope

import (
	"k8s.io/apiextensions-apiserver/pkg/crdserverscheme"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/fakers"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/serializer"
)

func NewRequestScope() *handlers.RequestScope {
	typeConverter, err := managedfields.NewTypeConverter(make(map[string]*spec.Schema), false)
	if err != nil {
		klog.Errorf("Failed to create TypeConverter: %v\n", err)
		return nil
	}
	fieldManager, err := managedfields.NewDefaultFieldManager(
		typeConverter,
		fakers.NewFakeObjectConvertor(),
		fakers.NewFakeObjectDefaulter(),
		unstructuredscheme.NewUnstructuredCreator(),
		schema.GroupVersionKind{},
		schema.GroupVersion{},
		"",
		make(map[fieldpath.APIVersion]*fieldpath.Set),
	)
	if err != nil {
		klog.Errorf("Failed to create FieldManager: %v\n", err)
		return nil
	}

	requestScope := handlers.RequestScope{
		Namer: handlers.ContextBasedNaming{
			Namer:         meta.NewAccessor(),
			ClusterScoped: false,
		},

		Serializer:     serializer.NewNegotiatedSerializer(),
		ParameterCodec: scheme.ParameterCodec,

		StandardSerializers: make([]runtime.SerializerInfo, 0),

		Creater:         unstructuredscheme.NewUnstructuredCreator(),
		Convertor:       fakers.NewFakeObjectConvertor(),
		Defaulter:       fakers.NewFakeObjectDefaulter(),
		Typer:           crdserverscheme.NewUnstructuredObjectTyper(),
		UnsafeConvertor: fakers.NewFakeObjectConvertor(),
		Authorizer:      fakers.NewAlwaysAllowAuthorizer(),

		EquivalentResourceMapper: runtime.NewEquivalentResourceRegistry(),

		TableConvertor: rest.NewDefaultTableConvertor(schema.GroupResource{}),
		FieldManager:   fieldManager,

		Resource: schema.GroupVersionResource{},
		Kind:     schema.GroupVersionKind{},

		AcceptsGroupVersionDelegate: nil,

		Subresource: "",

		MetaGroupVersion: metav1.SchemeGroupVersion,

		HubGroupVersion: schema.GroupVersion{},

		MaxRequestBodyBytes: int64(3 * 1024 * 1024),
	}
	return &requestScope
}
