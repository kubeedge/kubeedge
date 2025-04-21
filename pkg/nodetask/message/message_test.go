/*
Copyright 2025 The KubeEdge Authors.

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

package message

import (
	"testing"

	"github.com/stretchr/testify/require"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestResourceCheck(t *testing.T) {
	var (
		res Resource
		err error
	)
	err = res.Check()
	require.ErrorContains(t, err, "the APIVersion field must not be blank")
	res.APIVersion = operationsv1alpha2.SchemeGroupVersion.String()
	err = res.Check()
	require.ErrorContains(t, err, "the ResourceType field must not be blank")
	res.ResourceType = operationsv1alpha2.ResourceNodeUpgradeJob
	err = res.Check()
	require.ErrorContains(t, err, "the job name field must not be blank")
	res.JobName = "test"
	err = res.Check()
	require.ErrorContains(t, err, "the node name field must not be blank")
	res.NodeName = "node1"
	err = res.Check()
	require.NoError(t, err)
}

func TestResource(t *testing.T) {
	res := Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      "test",
		NodeName:     "node1",
	}
	require.True(t, IsNodeJobResource(res.String()))
	parsed := ParseResource(res.String())
	require.NotNil(t, parsed)
	require.Equal(t, res.APIVersion, parsed.APIVersion)
	require.Equal(t, res.ResourceType, parsed.ResourceType)
	require.Equal(t, res.JobName, parsed.JobName)
	require.Equal(t, res.NodeName, parsed.NodeName)
}
