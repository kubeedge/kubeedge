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

package main

import (
	"fmt"
	generatedopenapi "github.com/kubeedge/kubeedge/apidoc/generated/openapi"
	"github.com/kubeedge/kubeedge/apidoc/tools/lib"
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
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func main() {
	// Create a new Schema object
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

	// Create a default REST mapper
	mapper := meta.NewDefaultRESTMapper(nil)

	// NodeGroup
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

	// Add mapping for Operations v1alpha1 version resources
	mapper.AddSpecific(operationsv1alpha1.SchemeGroupVersion.WithKind("ImagePrePullJob"),
		operationsv1alpha1.SchemeGroupVersion.WithResource("imageprepulljobs"),
		operationsv1alpha1.SchemeGroupVersion.WithResource("imageprepulljob"), meta.RESTScopeNamespace)

	// Add mapping for Policy v1alpha1 version resources
	mapper.AddSpecific(policyv1alpha1.SchemeGroupVersion.WithKind("ServiceAccountAccess"),
		policyv1alpha1.SchemeGroupVersion.WithResource("serviceaccountaccesss"),
		policyv1alpha1.SchemeGroupVersion.WithResource("serviceaccountaccess"), meta.RESTScopeNamespace)

	// Add mapping for ReliableSyncs v1alpha1 version resources
	mapper.AddSpecific(reliablesyncsv1alpha1.SchemeGroupVersion.WithKind("ObjectSync"),
		reliablesyncsv1alpha1.SchemeGroupVersion.WithResource("objectsyncs"),
		reliablesyncsv1alpha1.SchemeGroupVersion.WithResource("objectsync"), meta.RESTScopeNamespace)

	// Add mapping for Rules v1 version resources
	mapper.AddSpecific(rulesv1.SchemeGroupVersion.WithKind("Rule"),
		rulesv1.SchemeGroupVersion.WithResource("rules"),
		rulesv1.SchemeGroupVersion.WithResource("rule"), meta.RESTScopeRoot)

	// Set OpenAPI spec information
	spec, err := lib.RenderOpenAPISpec(lib.Config{
		Info: spec.InfoProps{
			Title:       "Kubeedge OpenAPI",
			Version:     "unversioned",
			Description: "Kubeedge is Open, Multi-Cloud, Multi-Cluster Kubernetes Orchestration System. For more information, please see https://github.com/Kubeedge/Kubeedge.",
			License: &spec.License{
				Name: "Apache 2.0",                                       // License name
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html", // License URL
			},
		},
		Scheme: Scheme,                             // Used Schema
		Codecs: serializer.NewCodecFactory(Scheme), // Used codecs
		OpenAPIDefinitions: []common.GetOpenAPIDefinitions{
			generatedopenapi.GetOpenAPIDefinitions, // GetOpenAPI definitions function
		},
		Resources: []lib.ResourceWithNamespaceScoped{
			// Define resources and their namespace scoped and resource mapping correspondingly
			{GVR: appsv1alpha1.SchemeGroupVersion.WithResource("edgeapplications"), NamespaceScoped: false},
			{GVR: appsv1alpha1.SchemeGroupVersion.WithResource("nodegroups"), NamespaceScoped: false},
			{GVR: devicesv1alpha2.SchemeGroupVersion.WithResource("devices"), NamespaceScoped: false},
			{GVR: devicesv1beta1.SchemeGroupVersion.WithResource("devices"), NamespaceScoped: false},
			{GVR: operationsv1alpha1.SchemeGroupVersion.WithResource("imageprepulljobs"), NamespaceScoped: false},
			{GVR: policyv1alpha1.SchemeGroupVersion.WithResource("serviceaccountaccesss"), NamespaceScoped: false},
			{GVR: reliablesyncsv1alpha1.SchemeGroupVersion.WithResource("objectsyncs"), NamespaceScoped: false},
			{GVR: rulesv1.SchemeGroupVersion.WithResource("rules"), NamespaceScoped: false},
		},
		Mapper: mapper,
	})
	if err != nil {
		klog.Fatal(err.Error())
	}
	fmt.Println(spec) // Print generated OpenAPI spec
}
