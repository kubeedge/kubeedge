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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func TestName(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{
		name: "test-controller",
	}

	assert.NotNil(bc.name)
	assert.Equal("test-controller", bc.Name())
}

func TestStart(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{}

	err := bc.Start()
	assert.Error(err)
	assert.EqualError(err, "controller not implemented")
}

func TestStageCompleted(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{}

	result := bc.StageCompleted("task-id", "state")
	assert.NotNil(result)
	assert.False(result)
}

func TestGetNodeStatus(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{}

	status, err := bc.GetNodeStatus("test-node")

	assert.Error(err)
	assert.Nil(status)
	assert.EqualError(err, "function GetNodeStatus need to be init")
}

func TestUpdateNodeStatus(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{}

	err := bc.UpdateNodeStatus("test-node", []v1alpha1.TaskStatus{})
	assert.EqualError(err, "function UpdateNodeStatus need to be init", "Expected error message 'function UpdateNodeStatus need to be init'")
}

func TestIsNodeReady(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		node           *v1.Node
		expectedResult bool
	}{
		{
			name: "Node is ready",
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "Node is not ready",
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "Node ready condition is unknown",
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeReady,
							Status: v1.ConditionUnknown,
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "Node has no ready condition",
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{
							Type:   v1.NodeDiskPressure,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isNodeReady(tc.node)
			assert.Equal(tc.expectedResult, result)
		})
	}
}

func TestReportNodeStatus(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{}

	event := fsm.Event{
		Type:   "test-event",
		Action: api.Action("test-action"),
		Msg:    "test-message",
	}

	nodeStatus, err := bc.ReportNodeStatus("test-node", "test-status", event)
	stdResult := api.State("")

	assert.Equal(stdResult, nodeStatus)
	assert.EqualError(err, "function ReportNodeStatus need to be init")
}

func TestReportTaskStatus(t *testing.T) {
	assert := assert.New(t)

	bc := &BaseController{}

	event := fsm.Event{
		Type:   "test-event",
		Action: api.Action("test-action"),
		Msg:    "test-message",
	}

	taskStatus, err := bc.ReportTaskStatus("test-node", event)
	stdResult := api.State("")

	assert.Equal(stdResult, taskStatus)
	assert.EqualError(err, "function ReportTaskStatus need to be init")
}

func TestRegister(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		controllerName string
		controller     Controller
		expectedLen    int
	}{
		{
			name:           "Register a new controller",
			controllerName: "test-controller-1",
			controller:     &BaseController{name: "mock-controller-1"},
			expectedLen:    1,
		},
		{
			name:           "Register a duplicate controller, won't register",
			controllerName: "test-controller-1",
			controller:     &BaseController{name: "mock-controller-2"},
			expectedLen:    1,
		},
		{
			name:           "Register another new controller",
			controllerName: "test-controller-2",
			controller:     &BaseController{name: "mock-controller-3"},
			expectedLen:    2,
		},
	}

	// Cleaning the controllers map first
	controllers = make(map[string]Controller)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			Register(test.controllerName, test.controller)

			assert.Len(controllers, test.expectedLen)
			assert.Contains(controllers, test.controllerName)
			assert.Equal(test.controller, controllers[test.controllerName])
		})
	}
}

func TestStartAllController(t *testing.T) {
	assert := assert.New(t)

	// Cleaning the controllers map first
	controllers = make(map[string]Controller)

	ctrl1 := &BaseController{name: "test-controller-1"}
	ctrl2 := &BaseController{name: "test-controller-2"}

	Register("test-controller-1", ctrl1)
	Register("test-controller-2", ctrl2)

	err := StartAllController()

	assert.Error(err)

	/*
		Complete expected error is:
		"start test-controller-1 controller failed: controller not implemented"
		or "start test-controller-2 controller failed: controller not implemented"
	*/
	assert.ErrorContains(err, "controller failed: controller not implemented")
}

func TestGetController(t *testing.T) {
	assert := assert.New(t)

	ctrl1 := &BaseController{name: "test-controller-1"}
	ctrl2 := &BaseController{name: "test-controller-2"}

	// Cleaning the controllers map first
	controllers = make(map[string]Controller)

	Register("test-controller-1", ctrl1)
	Register("test-controller-2", ctrl2)

	expectedCtrl1, err := GetController("test-controller-1")
	assert.Equal(expectedCtrl1, ctrl1)
	assert.NoError(err)

	exceptedCtrl2, err := GetController("test-controller-2")
	assert.Equal(exceptedCtrl2, ctrl2)
	assert.NoError(err)

	ctrl3, err := GetController("non-existent-controller")
	assert.Nil(ctrl3)
	assert.Error(err)
	assert.EqualError(err, "controller non-existent-controller is not registered")
}
