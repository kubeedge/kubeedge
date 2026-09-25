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

package controllermanager

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
)

// unreachableAPIServer is a host no API server listens on, so that the tests
// never reach a real cluster.
const unreachableAPIServer = "127.0.0.1:1"

// newTestRESTMapper maps every kind of the kubeedge scheme, so that the manager
// can set up its informers without discovering the kinds from an API server.
func newTestRESTMapper() meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper(nil)
	for gvk := range kubeedgeScheme.AllKnownTypes() {
		mapper.Add(gvk, meta.RESTScopeNamespace)
	}
	return mapper
}

func TestSetupControllers(t *testing.T) {
	mgr, err := controllerruntime.NewManager(&rest.Config{Host: unreachableAPIServer}, controllerruntime.Options{
		Scheme: kubeedgeScheme,
		// Controller names are registered process wide, so skip the uniqueness
		// check to keep this test repeatable.
		Controller: config.Controller{SkipNameValidation: ptr.To(true)},
		MapperProvider: func(_ *rest.Config, _ *http.Client) (meta.RESTMapper, error) {
			return newTestRESTMapper(), nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Every controller, including the node task ones that used to be given a
	// separate cache, is set up with the cache owned by the manager.
	assert.NoError(t, setupControllers(ctx, mgr))
}

func TestNewControllerManager(t *testing.T) {
	// Setting up the controllers needs the API server, so it fails here and the
	// error has to be returned instead of a manager.
	mgr, err := NewControllerManager(context.Background(), &rest.Config{Host: unreachableAPIServer}, "")
	assert.Error(t, err)
	assert.Nil(t, mgr)
}
