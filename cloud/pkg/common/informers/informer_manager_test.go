/*
Copyright 2026 The KubeEdge Authors.

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

package informers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestInformersResourceFor(t *testing.T) {
	gv := schema.GroupVersion{Group: "networking.istio.io", Version: "v1alpha3"}
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gv})
	// The plural is defined by the CRD, it is not derived from the kind.
	mapper.AddSpecific(gv.WithKind("Gateway"), gv.WithResource("gateways"), gv.WithResource("gateway"), meta.RESTScopeNamespace)
	ifs := &informers{mapper: mapper}

	gvr, err := ifs.ResourceFor(gv.WithKind("Gateway"))
	assert.NoError(t, err)
	assert.Equal(t, gv.WithResource("gateways"), gvr)

	_, err = ifs.ResourceFor(gv.WithKind("Unknown"))
	assert.Error(t, err)
}
