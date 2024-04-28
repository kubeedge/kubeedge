package lib

import (
	"encoding/json"
	"fmt"
	"net"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/common/restfuladapter"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// ResourceWithNamespaceScoped 包含资源的组、版本、资源（GVR）信息以及是否支持命名空间作用域。
type ResourceWithNamespaceScoped struct {
	GVR             schema.GroupVersionResource
	NamespaceScoped bool
}

// Config 用于配置swagger信息的结构体。
type Config struct {
	Scheme             *runtime.Scheme                // 运行时Scheme，用于类型注册和对象实例化。
	Codecs             serializer.CodecFactory        // 编解码器工厂，用于处理对象的编码和解码。
	Info               spec.InfoProps                 // OpenAPI规范的基础信息，如标题、版本等。
	OpenAPIDefinitions []common.GetOpenAPIDefinitions // OpenAPI定义的函数数组。
	Resources          []ResourceWithNamespaceScoped  // 包含GVR和命名空间作用域信息的资源列表。
	Mapper             *meta.DefaultRESTMapper        // REST映射器，用于GVR和GVK之间的转换。
}

// GetOpenAPIDefinitions 获取OpenAPI定义。
func (c *Config) GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	out := map[string]common.OpenAPIDefinition{} // 创建一个空的OpenAPIDefinition映射。
	for _, def := range c.OpenAPIDefinitions {   // 遍历OpenAPIDefinitions函数数组。
		for k, v := range def(ref) { // 调用每个函数，传入引用回调，获取定义映射。
			out[k] = v // 将获取的定义添加到输出映射中。
		}
	}
	return out // 返回包含所有定义的映射。
}

// RenderOpenAPISpec 创建Swagger的OpenAPI规范。
func RenderOpenAPISpec(cfg Config) (string, error) {
	// 创建并配置服务器选项。
	options := genericoptions.NewRecommendedOptions("/registry/kubeedge.io", cfg.Codecs.LegacyCodec()) //规定api服务器的前缀为/registry/kubeedge.io
	options.SecureServing.ServerCert.CertDirectory = "/tmp/kubeedge-swagger"                           // 设置证书目录。
	options.SecureServing.BindPort = 6446                                                              // 设置绑定端口。
	// 禁用不需要的服务器组件。
	options.Etcd = nil
	options.Authentication = nil
	options.Authorization = nil
	options.CoreAPI = nil
	options.Admission = nil
	// 尝试使用自签名证书配置安全服务。
	if err := options.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		klog.Fatal(fmt.Errorf("error creating self-signed certificates: %v", err))
	}
	// 应用安全服务配置,创建服务器配置。
	serverConfig := genericapiserver.NewRecommendedConfig(cfg.Codecs)
	if err := options.SecureServing.ApplyTo(&serverConfig.Config.SecureServing, &serverConfig.Config.LoopbackClientConfig); err != nil {
		klog.Fatal(err)
		return "", err
	}
	// 配置OpenAPI和OpenAPI V3。
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(cfg.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(cfg.Scheme))
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(cfg.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(cfg.Scheme))
	serverConfig.OpenAPIConfig.Info.InfoProps = cfg.Info
	// 创建并配置API服务器，创建一个名为 "kubeedge-openapi-server" 的新的 API 服务器实例
	genericServer, err := serverConfig.Complete().New("kubeedge-openapi-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	// 创建资源路由表。
	table, err := createRouterTable(&cfg)
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	// 为每个资源组配置API。
	for g, resmap := range table {
		apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(g, cfg.Scheme, metav1.ParameterCodec, cfg.Codecs)
		storage := map[string]map[string]rest.Storage{}
		for r, info := range resmap {
			if storage[info.gvk.Version] == nil {
				storage[info.gvk.Version] = map[string]rest.Storage{}
			}
			storage[info.gvk.Version][r.Resource] = &StandardREST{info}
			// Add status router for all resources.
			storage[info.gvk.Version][r.Resource+"/status"] = &StatusREST{StatusInfo{
				gvk: info.gvk,
				obj: info.obj,
			}}

			// To define additional endpoints for CRD resources, we need to
			// implement our own REST interface and add it to our custom path.
			if r.Resource == "clusters" {
				storage[info.gvk.Version][r.Resource+"/proxy"] = &ProxyREST{}
			}
		}

		for version, s := range storage {
			apiGroupInfo.VersionedResourcesStorageMap[version] = s
		}

		// Install api to apiserver.
		if err := genericServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			klog.Fatal(err)
			return "", err
		}
	}
	// 创建Swagger规范。
	//BuildOpenAPISpecFromRoutes
	// Create Swagger Spec.
	spec, err := builder.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(genericServer.Handler.GoRestfulContainer.RegisteredWebServices()), serverConfig.OpenAPIConfig)
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	// 将规范序列化为JSON字符串。
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	return string(data), nil
}

// createRouterTable create router map for every resource.
func createRouterTable(cfg *Config) (map[string]map[schema.GroupVersionResource]ResourceInfo, error) {
	table := map[string]map[schema.GroupVersionResource]ResourceInfo{}
	// Create router map for every resource
	for _, ti := range cfg.Resources {
		var resmap map[schema.GroupVersionResource]ResourceInfo
		if m, found := table[ti.GVR.Group]; found {
			resmap = m
		} else {
			resmap = map[schema.GroupVersionResource]ResourceInfo{}
			table[ti.GVR.Group] = resmap
		}

		gvk, err := cfg.Mapper.KindFor(ti.GVR)
		if err != nil {
			klog.Fatal(err)
			return nil, err
		}
		obj, err := cfg.Scheme.New(gvk)

		if err != nil {
			klog.Fatal(err)
			return nil, err
		}

		list, err := cfg.Scheme.New(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
		if err != nil {
			klog.Fatal(err)
			return nil, err
		}

		resmap[ti.GVR] = ResourceInfo{
			gvk:             gvk,
			obj:             obj,
			list:            list,
			namespaceScoped: ti.NamespaceScoped,
		}
	}

	return table, nil
}
