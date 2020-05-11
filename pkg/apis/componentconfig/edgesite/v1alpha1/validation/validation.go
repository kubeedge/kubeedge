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

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	cloudvalidation "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1/validation"
	edgevalidation "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1/validation"
	edgesiteconfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgesite/v1alpha1"
)

// ValidateEdgeSiteConfiguration validates `c` and returns an errorList if it is invalid
func ValidateEdgeSiteConfiguration(c *edgesiteconfig.EdgeSiteConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, edgevalidation.ValidateDataBase(*c.DataBase)...)
	allErrs = append(allErrs, cloudvalidation.ValidateKubeAPIConfig(*c.KubeAPIConfig)...)

	allErrs = append(allErrs, cloudvalidation.ValidateModuleEdgeController(*c.Modules.EdgeController)...)
	allErrs = append(allErrs, edgevalidation.ValidateModuleEdged(*c.Modules.Edged)...)
	allErrs = append(allErrs, edgevalidation.ValidateModuleMetaManager(*c.Modules.MetaManager)...)
	return allErrs
}
