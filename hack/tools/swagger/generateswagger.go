package main

import (
	"fmt"
	"os"

	"github.com/kubeedge/kubeedge/hack/tools/swagger/lib"

	appsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/apps/v1alpha1"
	devicesv1alpha2 "github.com/kubeedge/kubeedge/pkg/apis/devices/v1alpha2"
	devicesv1beta1 "github.com/kubeedge/kubeedge/pkg/apis/devices/v1beta1"
	operationsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	policyv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/policy/v1alpha1"
	reliablesyncsv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	rulesv1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	generatedopenapi "github.com/kubeedge/kubeedge/pkg/generated/openapi"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func main() {
	// 创建新的 Schema 对象
	Scheme := runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(Scheme)) // add Kubernetes schemes
	utilruntime.Must(appsv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(devicesv1alpha2.AddToScheme(Scheme))
	utilruntime.Must(devicesv1beta1.AddToScheme(Scheme))
	utilruntime.Must(operationsv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(policyv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(reliablesyncsv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(policyv1alpha1.AddToScheme(Scheme))
	utilruntime.Must(rulesv1.AddToScheme(Scheme))

	// 创建默认的 REST 映射器
	mapper := meta.NewDefaultRESTMapper(nil)

	//NodeGroup
	mapper.AddSpecific(appsv1alpha1.SchemeGroupVersion.WithKind("EdgeApplication"),
		appsv1alpha1.SchemeGroupVersion.WithResource("edgeapplications"),
		appsv1alpha1.SchemeGroupVersion.WithResource("edgeapplication"), meta.RESTScopeRoot)

	mapper.AddSpecific(appsv1alpha1.SchemeGroupVersion.WithKind("NodeGroup"),
		appsv1alpha1.SchemeGroupVersion.WithResource("nodegroups"),
		appsv1alpha1.SchemeGroupVersion.WithResource("nodegroup"), meta.RESTScopeRoot)

	mapper.AddSpecific(devicesv1alpha2.SchemeGroupVersion.WithKind("Device"),
		devicesv1alpha2.SchemeGroupVersion.WithResource("devices"),
		devicesv1alpha2.SchemeGroupVersion.WithResource("device"), meta.RESTScopeNamespace)

	mapper.AddSpecific(devicesv1beta1.SchemeGroupVersion.WithKind("Device"),
		devicesv1beta1.SchemeGroupVersion.WithResource("devices"),
		devicesv1beta1.SchemeGroupVersion.WithResource("device"), meta.RESTScopeNamespace)

	// 为Operations v1alpha1 版本资源添加映射
	mapper.AddSpecific(operationsv1alpha1.SchemeGroupVersion.WithKind("ImagePrePullJob"),
		operationsv1alpha1.SchemeGroupVersion.WithResource("imageprepulljobs"),
		operationsv1alpha1.SchemeGroupVersion.WithResource("imageprepulljob"), meta.RESTScopeNamespace)

	// 为Policy v1alpha1 版本资源添加映射
	mapper.AddSpecific(policyv1alpha1.SchemeGroupVersion.WithKind("ServiceAccountAccess"),
		policyv1alpha1.SchemeGroupVersion.WithResource("serviceaccountaccesss"),
		policyv1alpha1.SchemeGroupVersion.WithResource("serviceaccountaccess"), meta.RESTScopeNamespace)

	// 为ReliableSyncs v1alpha1 版本资源添加映射
	mapper.AddSpecific(reliablesyncsv1alpha1.SchemeGroupVersion.WithKind("ObjectSync"),
		reliablesyncsv1alpha1.SchemeGroupVersion.WithResource("objectsyncs"),
		reliablesyncsv1alpha1.SchemeGroupVersion.WithResource("objectsync"), meta.RESTScopeNamespace)

	// 为Rules v1 版本资源添加映射
	mapper.AddSpecific(rulesv1.SchemeGroupVersion.WithKind("Rule"),
		rulesv1.SchemeGroupVersion.WithResource("rules"),
		rulesv1.SchemeGroupVersion.WithResource("rule"), meta.RESTScopeRoot)

	// 设置OpenAPI规范的信息
	spec, err := lib.RenderOpenAPISpec(lib.Config{
		Info: spec.InfoProps{
			Title:       "Kubeedge OpenAPI",
			Version:     "unversioned",
			Description: "Kubeedge is Open, Multi-Cloud, Multi-Cluster Kubernetes Orchestration System. For more information, please see https://github.com/Kubeedge/Kubeedge.",
			License: &spec.License{
				Name: "Apache 2.0",                                       // 许可证名称
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html", // 许可证 URL
			},
		},
		Scheme: Scheme,                             // 使用的 Schema
		Codecs: serializer.NewCodecFactory(Scheme), // 使用的编解码器NewCodecFactory-k8s.io
		OpenAPIDefinitions: []common.GetOpenAPIDefinitions{ //GetOpenAPIDefinitions-k8s.io
			generatedopenapi.GetOpenAPIDefinitions, // 获取 OpenAPI 定义的函数 GetOpenAPIDefinitions-zz_ge~openapi.go
		},
		Resources: []lib.ResourceWithNamespaceScoped{
			// 定义资源和它们的命名空间作用域和资源映射相对应
			{GVR: appsv1alpha1.SchemeGroupVersion.WithResource("edgeapplications"), NamespaceScoped: false},
			{GVR: appsv1alpha1.SchemeGroupVersion.WithResource("nodegroups"), NamespaceScoped: false},
			{GVR: devicesv1alpha2.SchemeGroupVersion.WithResource("devices"), NamespaceScoped: false},
			{GVR: devicesv1beta1.SchemeGroupVersion.WithResource("devices"), NamespaceScoped: false},
			{GVR: operationsv1alpha1.SchemeGroupVersion.WithResource("imageprepulljobs"), NamespaceScoped: false},
			{GVR: policyv1alpha1.SchemeGroupVersion.WithResource("serviceaccountaccesss"), NamespaceScoped: false},
			{GVR: reliablesyncsv1alpha1.SchemeGroupVersion.WithResource("objectsyncs"), NamespaceScoped: false},
			{GVR: rulesv1.SchemeGroupVersion.WithResource("rules"), NamespaceScoped: false},
		},
		// Resources: nil,
		Mapper: mapper,
	})
	if err != nil {
		klog.Fatal(err.Error())
	}
	fmt.Println(spec) // 打印生成的 OpenAPI 规范

	err = os.WriteFile("./api/openapi-spec/swagger.json", []byte(spec), 0644)
	if err != nil {
		klog.Fatal(err.Error())
	}
}
