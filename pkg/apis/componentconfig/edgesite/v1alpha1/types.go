/*
Copyright 2019 The KubeEdge Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cloudcoreconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	// DataBaseDriverName is sqlite3
	DataBaseDriverName = "sqlite3"
	// DataBaseAliasName is default
	DataBaseAliasName = "default"
	// DataBaseDataSource is edge.db
	DataBaseDataSource = "/var/lib/kubeedge/edgesite.db"
)

// EdgeSiteConfig indicates the EdgeSite config which read from EdgeSite config file
type EdgeSiteConfig struct {
	metav1.TypeMeta
	// DataBase indicates database info
	// +Required
	DataBase *edgecoreconfig.DataBase `json:"database,omitempty"`
	// KubeAPIConfig indicates the kubernetes cluster info which CloudCore will connected
	// +Required
	KubeAPIConfig *cloudcoreconfig.KubeAPIConfig `json:"kubeAPIConfig,omitempty"`
	// Modules indicates CloudCore modules config
	// +Required
	Modules *Modules `json:"modules,omitempty"`
}

// Modules indicates the modules which EdgeSite will be used
type Modules struct {
	// EdgeController indicates edgeController module config
	EdgeController *cloudcoreconfig.EdgeController `json:"edgeController,omitempty"`
	// Edged indicates edged module config
	// +Required
	Edged *edgecoreconfig.Edged `json:"edged,omitempty"`
	// MetaManager indicates meta module config
	// +Required
	MetaManager *edgecoreconfig.MetaManager `json:"metaManager,omitempty"`
}
