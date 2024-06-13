/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

// ResourceWithNamespaceScoped contains information about the group, version, resource (GVR), and whether it supports namespace scope.
type ResourceWithNamespaceScoped struct {
	GVR             schema.GroupVersionResource
	NamespaceScoped bool
}

// Config is the struct used to configure Swagger information.
type Config struct {
	Scheme             *runtime.Scheme                // Runtime Scheme used for type registration and object instantiation.
	Codecs             serializer.CodecFactory        // Codec factory used for encoding and decoding objects.
	Info               spec.InfoProps                 // Basic information for the OpenAPI specification, such as title, version, etc.
	OpenAPIDefinitions []common.GetOpenAPIDefinitions // Array of functions to obtain OpenAPI definitions.
	Resources          []ResourceWithNamespaceScoped  // List of resources containing GVR and namespace scope information.
	Mapper             *meta.DefaultRESTMapper        // REST mapper for conversion between GVR and GVK.
}

// GetOpenAPIDefinitions gets OpenAPI definitions.
func (c *Config) GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	out := map[string]common.OpenAPIDefinition{} // Create an empty OpenAPIDefinition map.
	for _, def := range c.OpenAPIDefinitions {   // Iterate through the OpenAPIDefinitions function array.
		for k, v := range def(ref) { // Call each function, passing in the reference callback, to get the definition map.
			out[k] = v // Add the obtained definition to the output map.
		}
	}
	return out // Return the map containing all definitions.
}

// RenderOpenAPISpec creates the OpenAPI specification for Swagger.
func RenderOpenAPISpec(cfg Config) (string, error) {
	// Create and configure server options.
	options := genericoptions.NewRecommendedOptions("/registry/kubeedge.io", cfg.Codecs.LegacyCodec()) // Set the prefix for the API server to /registry/kubeedge.io.
	options.SecureServing.ServerCert.CertDirectory = "/tmp/kubeedge-tools"                             // Set the certificate directory.
	options.SecureServing.BindPort = 6446                                                              // Set the bind port.
	// Disable unnecessary server components.
	options.Etcd = nil
	options.Authentication = nil
	options.Authorization = nil
	options.CoreAPI = nil
	options.Admission = nil
	// Attempt to configure secure serving with self-signed certificates.
	if err := options.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		klog.Fatal(fmt.Errorf("error creating self-signed certificates: %v", err))
	}
	// Apply secure serving configuration, create server configuration.
	serverConfig := genericapiserver.NewRecommendedConfig(cfg.Codecs)
	if err := options.SecureServing.ApplyTo(&serverConfig.Config.SecureServing, &serverConfig.Config.LoopbackClientConfig); err != nil {
		klog.Fatal(err)
		return "", err
	}
	// Configure OpenAPI and OpenAPI V3.
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(cfg.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(cfg.Scheme))
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(cfg.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(cfg.Scheme))
	serverConfig.OpenAPIConfig.Info.InfoProps = cfg.Info
	// Create and configure the API server, creating a new API server instance named "kubeedge-generated-server".
	genericServer, err := serverConfig.Complete().New("kubeedge-generated-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	// Create a resource router table.
	table, err := createRouterTable(&cfg)
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	// Configure APIs for each resource group.
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

		// Install API to API server.
		if err := genericServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			klog.Fatal(err)
			return "", err
		}
	}
	// Create Swagger specification.
	// BuildOpenAPISpecFromRoutes
	// Create Swagger Spec.
	spec, err := builder.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(genericServer.Handler.GoRestfulContainer.RegisteredWebServices()), serverConfig.OpenAPIConfig)
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	// Serialize the specification to a JSON string.
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		klog.Fatal(err)
		return "", err
	}
	return string(data), nil
}

// createRouterTable creates a router map for every resource.
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
