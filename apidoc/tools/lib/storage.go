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
	"context"
	"net/http"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
)

// StandardREST define CRUD api for resources.
type StandardREST struct {
	cfg ResourceInfo
}

// StatusREST define status endpoint for resources.
type StatusREST struct {
	cfg StatusInfo
}

// ProxyREST define proxy endpoint for resources.
type ProxyREST struct{}

// Implement below interfaces for StandardREST.
var _ rest.GroupVersionKindProvider = &StandardREST{}
var _ rest.Scoper = &StandardREST{}
var _ rest.StandardStorage = &StandardREST{}
var _ rest.SingularNameProvider = &StandardREST{}

// Implement below interfaces for StatusREST.
var _ rest.Patcher = &StatusREST{}

// Implement below interfaces for ProxyREST.
var _ rest.Connecter = &ProxyREST{}

// GroupVersionKind implement GroupVersionKind interface.
func (r *StandardREST) GroupVersionKind(_ schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

// NamespaceScoped implement NamespaceScoped interface.
func (r *StandardREST) NamespaceScoped() bool {
	return r.cfg.namespaceScoped
}

// New implement New interface.
func (r *StandardREST) New() runtime.Object {
	return r.cfg.obj
}

// Create implement Create interface.
func (r *StandardREST) Create(_ context.Context, _ runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Get implement Get interface.
func (r *StandardREST) Get(_ context.Context, _ string, _ *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// NewList implement NewList interface.
func (r *StandardREST) NewList() runtime.Object {
	return r.cfg.list
}

// List implement List interface.
func (r *StandardREST) List(_ context.Context, _ *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

// ConvertToTable implement ConvertToTable interface.
func (r *StandardREST) ConvertToTable(_ context.Context, _ runtime.Object, _ runtime.Object) (*metav1.Table, error) {
	return nil, nil
}

// Update implement Update interface.
func (r *StandardREST) Update(_ context.Context, _ string, _ rest.UpdatedObjectInfo, _ rest.ValidateObjectFunc, _ rest.ValidateObjectUpdateFunc, _ bool, _ *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

// Delete implement Delete interface.
func (r *StandardREST) Delete(_ context.Context, _ string, _ rest.ValidateObjectFunc, _ *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

// DeleteCollection implement DeleteCollection interface.
func (r *StandardREST) DeleteCollection(_ context.Context, _ rest.ValidateObjectFunc, _ *metav1.DeleteOptions, _ *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

// Watch implement Watch interface.
func (r *StandardREST) Watch(_ context.Context, _ *metainternalversion.ListOptions) (watch.Interface, error) {
	return nil, nil
}

// Destroy cleans up its resources on shutdown.
func (r *StandardREST) Destroy() {
	// Given no underlying store, so we don't
	// need to destroy anything.
}

// GetSingularName implements the SingularNameProvider interfaces.
func (r *StandardREST) GetSingularName() string {
	return ""
}

// GroupVersionKind implement GroupVersionKind interface.
func (r *StatusREST) GroupVersionKind(_ schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

// New returns Cluster object.
func (r *StatusREST) New() runtime.Object {
	return r.cfg.obj
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(_ context.Context, _ string, _ rest.UpdatedObjectInfo, _ rest.ValidateObjectFunc, _ rest.ValidateObjectUpdateFunc, _ bool, _ *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

// Get retrieves the status object.
func (r *StatusREST) Get(_ context.Context, _ string, _ *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Destroy cleans up its resources on shutdown.
func (r *StatusREST) Destroy() {
	// Given no underlying store, so we don't
	// need to destroy anything.
}

// New returns an empty cluster proxy subresource.
func (r *ProxyREST) New() runtime.Object {
	//return &clusterv1alpha1.ClusterProxyOptions{}
	return nil
}

// ConnectMethods returns the list of HTTP methods handled by Connect.
func (r *ProxyREST) ConnectMethods() []string {
	return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
}

// NewConnectOptions returns versioned resource that represents proxy parameters.
func (r *ProxyREST) NewConnectOptions() (runtime.Object, bool, string) {
	//return &clusterv1alpha1.ClusterProxyOptions{}, true, "path"
	return nil, true, "path"
}

// Connect implement Connect interface.
func (r *ProxyREST) Connect(_ context.Context, _ string, _ runtime.Object, _ rest.Responder) (http.Handler, error) {
	return nil, nil
}

// Destroy cleans up its resources on shutdown.
func (r *ProxyREST) Destroy() {
	// Given no underlying store, so we don't
	// need to destroy anything.
}

// ResourceInfo is content of StandardREST.
type ResourceInfo struct {
	gvk             schema.GroupVersionKind
	obj             runtime.Object
	list            runtime.Object
	namespaceScoped bool
}

// StatusInfo is content of StatusREST.
type StatusInfo struct {
	gvk schema.GroupVersionKind
	obj runtime.Object
}
