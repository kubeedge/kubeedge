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
	"context"
	"errors"
	"fmt"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/klog/v2"
)

type kubeedgeResourceAuthorizer struct {
}

func (kubeedgeResourceAuthorizer) Authorize(_ context.Context, attrs authorizer.Attributes) (authorized authorizer.Decision, reason string, err error) {
	// allows all the kubeedge custom requests
	if isKubeedgeResourceAttributes(attrs) {
		klog.V(4).Infof("allow kubeedge request. verb=%s resource=%s, subresource=%s", attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource())
		return authorizer.DecisionAllow, "", nil
	}

	klog.V(4).Infof("deny request. verb=%s resource=%s, subresource=%s", attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource())
	return authorizer.DecisionNoOpinion, fmt.Sprintf("unknown request: verb=%s resource=%s, subresource=%s", attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource()), nil
}

func (kubeedgeResourceAuthorizer) RulesFor(user.Info, string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	return nil, nil, true, errors.New("kubeedge resource authorizer does not support user rule resolution")
}
