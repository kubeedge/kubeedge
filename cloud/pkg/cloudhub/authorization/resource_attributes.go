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

package authorization

import (
	"fmt"
	"strings"

	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/pkg/apis/authorization"
	"k8s.io/kubernetes/pkg/registry/authorization/util"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	cloudhubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	taskutil "github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	commonconstants "github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

const (
	kubeedgeResourceKey = "kubeedgeResource"
)

// getAuthorizerAttributes maps a beehive message to k8s api request
func getAuthorizerAttributes(router beehivemodel.MessageRoute, hubInfo cloudhubmodel.HubInfo) (authorizer.Attributes, error) {
	var (
		resAttrs    *authorization.ResourceAttributes
		nonResAttrs *authorization.NonResourceAttributes
		extra       = make(map[string]authorization.ExtraValue)
		err         error
	)
	if isKubeedgeResourceMessage(router) {
		nonResAttrs = getKubeedgeResourceAttributes(router)

		// put this key into extra to tells authorizers that this message operates kubeedge resource
		extra[kubeedgeResourceKey] = authorization.ExtraValue{}
	} else {
		resAttrs, err = getBuiltinResourceAttributes(router)
		if err != nil {
			return nil, err
		}
	}

	spec := authorization.SubjectAccessReviewSpec{
		ResourceAttributes:    resAttrs,
		NonResourceAttributes: nonResAttrs,
		User:                  constants.NodesUserPrefix + hubInfo.NodeID,
		Groups:                []string{constants.NodesGroup},
		Extra:                 extra,
	}
	attrs := util.AuthorizationAttributesFrom(spec)
	return &attrs, nil
}

// isKubeedgeResource judges whether a message accesses the kubeedge's resources
func isKubeedgeResourceMessage(router beehivemodel.MessageRoute) bool {
	switch router.Operation {
	case beehivemodel.ResponseOperation, beehivemodel.ResponseErrorOperation, beehivemodel.UploadOperation,
		taskutil.TaskPrePull, taskutil.TaskUpgrade, cloudhubmodel.OpKeepalive:
		return true
	}
	switch router.Source {
	case metaserver.MetaServerSource, cloudhubmodel.ResTwin:
		return true
	}
	if router.Resource == beehivemodel.ResourceTypeK8sCA || common.IsVolumeResource(router.Resource) {
		return true
	}

	_, resourceType, resourceName := splitResource(router.Resource)
	switch resourceType {
	case beehivemodel.ResourceTypeRuleStatus:
		return true
	}
	// kubeedge allows node to update a list of pod status
	if resourceType == beehivemodel.ResourceTypePodStatus && resourceName == "" {
		return true
	}

	return false
}

func getKubeedgeResourceAttributes(router beehivemodel.MessageRoute) *authorization.NonResourceAttributes {
	return &authorization.NonResourceAttributes{
		Path: router.Resource,
		Verb: router.Operation,
	}
}

func getBuiltinResourceAttributes(router beehivemodel.MessageRoute) (*authorization.ResourceAttributes, error) {
	namespace, resourceType, resourceName := splitResource(router.Resource)
	switch router.Operation {
	// nodestatus, podstatus is not allowed to insert
	case beehivemodel.InsertOperation:
		switch resourceType {
		case beehivemodel.ResourceTypePodStatus:
			resourceType = beehivemodel.ResourceTypePod
		case beehivemodel.ResourceTypeNodeStatus:
			resourceType = beehivemodel.ResourceTypeNode
		}
	}

	kubeRes, ok := resourceTypeToKubeResources[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type %q", resourceType)
	}

	var verb string
	switch router.Operation {
	case beehivemodel.InsertOperation:
		verb = "create"
	case beehivemodel.DeleteOperation:
		verb = "delete"
	case beehivemodel.UpdateOperation:
		verb = "update"
	case beehivemodel.PatchOperation:
		verb = "patch"
	case beehivemodel.QueryOperation:
		verb = "get"
		// the actual verb for serviceaccounts/token is `create`
		if resourceType == beehivemodel.ResourceTypeServiceAccountToken {
			verb = "create"
		}
	default:
		return nil, fmt.Errorf("unknown opeation %q", router.Operation)
	}

	if !kubeRes.namespaced {
		namespace = ""
	}
	return &authorization.ResourceAttributes{
		Namespace:   namespace,
		Verb:        verb,
		Group:       kubeRes.groupVersion.Group,
		Version:     kubeRes.groupVersion.Version,
		Resource:    kubeRes.resource,
		Subresource: kubeRes.subresource,
		Name:        resourceName,
	}, nil
}

func splitResource(resource string) (namespace string, resourceType string, resourceName string) {
	sli := strings.Split(resource, "/")
	for i := len(sli); i < 3; i++ {
		sli = append(sli, "")
	}
	namespace, resourceType, resourceName = sli[0], sli[1], sli[2]
	return
}

func isKubeedgeResourceAttributes(attrs authorizer.Attributes) bool {
	if attrs == nil || attrs.GetUser() == nil {
		return false
	}
	_, ok := attrs.GetUser().GetExtra()[kubeedgeResourceKey]
	return ok
}

type kubeResource struct {
	resource     string
	subresource  string
	groupVersion schema.GroupVersion
	namespaced   bool
}

var resourceTypeToKubeResources = map[string]kubeResource{
	beehivemodel.ResourceTypeNodeStatus:               {resource: "nodes", subresource: "status", groupVersion: v1.SchemeGroupVersion, namespaced: false},
	beehivemodel.ResourceTypePodStatus:                {resource: "pods", subresource: "status", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	beehivemodel.ResourceTypeConfigmap:                {resource: "configmaps", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	beehivemodel.ResourceTypeSecret:                   {resource: "secrets", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	beehivemodel.ResourceTypeServiceAccountToken:      {resource: "serviceaccounts", subresource: "token", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	commonconstants.ResourceTypePersistentVolume:      {resource: "persistentvolumes", groupVersion: v1.SchemeGroupVersion, namespaced: false},
	commonconstants.ResourceTypePersistentVolumeClaim: {resource: "persistentvolumeclaims", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	commonconstants.ResourceTypeVolumeAttachment:      {resource: "volumeattachments", groupVersion: storagev1.SchemeGroupVersion, namespaced: false},
	beehivemodel.ResourceTypeNode:                     {resource: "nodes", groupVersion: v1.SchemeGroupVersion, namespaced: false},
	beehivemodel.ResourceTypePod:                      {resource: "pods", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	beehivemodel.ResourceTypeNodePatch:                {resource: "nodes", subresource: "status", groupVersion: v1.SchemeGroupVersion, namespaced: false},
	beehivemodel.ResourceTypePodPatch:                 {resource: "pods", subresource: "status", groupVersion: v1.SchemeGroupVersion, namespaced: true},
	beehivemodel.ResourceTypeLease:                    {resource: "leases", groupVersion: coordinationv1.SchemeGroupVersion, namespaced: true},
	beehivemodel.ResourceTypeCSR:                      {resource: "certificatesigningrequests", groupVersion: certificatesv1.SchemeGroupVersion, namespaced: false},
}
