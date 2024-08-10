/*
Copyright 2023 The KubeEdge Authors.

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
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformer "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/manager"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

type Controller interface {
	Name() string
	Start() error
	ReportNodeStatus(string, string, fsm.Event) (api.State, error)
	ReportTaskStatus(string, fsm.Event) (api.State, error)
	ValidateNode(util.TaskMessage) []v1.Node
	GetNodeStatus(string) ([]v1alpha1.TaskStatus, error)
	UpdateNodeStatus(string, []v1alpha1.TaskStatus) error
	StageCompleted(taskID string, state api.State) bool
}

type BaseController struct {
	name        string
	Informer    k8sinformer.SharedInformerFactory
	TaskManager *manager.TaskCache
	MessageChan chan util.TaskMessage
	KubeClient  kubernetes.Interface
	CrdClient   crdClientset.Interface
}

func (bc *BaseController) Name() string {
	return bc.name
}

func (bc *BaseController) Start() error {
	return fmt.Errorf("controller not implemented")
}

func (bc *BaseController) StageCompleted(string, api.State) bool {
	return false
}

func (bc *BaseController) ValidateNode(taskMessage util.TaskMessage) []v1.Node {
	var validateNodes []v1.Node
	nodes, err := bc.getNodeList(taskMessage.NodeNames, taskMessage.LabelSelector)
	if err != nil {
		klog.Warningf("get node list error: %s", err.Error())
		return nil
	}
	for _, node := range nodes {
		if !util.IsEdgeNode(node) {
			klog.Warningf("Node(%s) is not edge node", node.Name)
			continue
		}
		ready := isNodeReady(node)
		if !ready {
			continue
		}
		validateNodes = append(validateNodes, *node)
	}
	return validateNodes
}

func (bc *BaseController) GetNodeStatus(string) ([]v1alpha1.TaskStatus, error) {
	return nil, fmt.Errorf("function GetNodeStatus need to be init")
}

func (bc *BaseController) UpdateNodeStatus(string, []v1alpha1.TaskStatus) error {
	return fmt.Errorf("function UpdateNodeStatus need to be init")
}

func isNodeReady(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status != v1.ConditionTrue {
			klog.Warningf("Node(%s) is in NotReady state", node.Name)
			return false
		}
	}
	return true
}

func (bc *BaseController) ReportNodeStatus(string, string, fsm.Event) (api.State, error) {
	return "", fmt.Errorf("function ReportNodeStatus need to be init")
}

func (bc *BaseController) ReportTaskStatus(string, fsm.Event) (api.State, error) {
	return "", fmt.Errorf("function ReportTaskStatus need to be init")
}

var (
	controllers = map[string]Controller{}
)

func Register(name string, controller Controller) {
	if _, ok := controllers[name]; ok {
		klog.Warningf("controller %s exists ", name)
	}
	controllers[name] = controller
}

func StartAllController() error {
	for name, controller := range controllers {
		err := controller.Start()
		if err != nil {
			return fmt.Errorf("start %s controller failed: %s", name, err.Error())
		}
	}
	return nil
}

func GetController(name string) (Controller, error) {
	controller, ok := controllers[name]
	if !ok {
		return nil, fmt.Errorf("controller %s is not registered", name)
	}
	return controller, nil
}

func (bc *BaseController) getNodeList(nodeNames []string, labelSelector *metav1.LabelSelector) ([]*v1.Node, error) {
	var nodesToUpgrade []*v1.Node

	if len(nodeNames) != 0 {
		for _, name := range nodeNames {
			node, err := bc.Informer.Core().V1().Nodes().Lister().Get(name)
			if err != nil {
				return nil, fmt.Errorf("failed to get node with name %s: %v", name, err)
			}
			nodesToUpgrade = append(nodesToUpgrade, node)
		}
	} else if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return nil, fmt.Errorf("labelSelector(%s) is not valid: %v", labelSelector, err)
		}

		nodes, err := bc.Informer.Core().V1().Nodes().Lister().List(selector)
		if err != nil {
			return nil, fmt.Errorf("failed to get nodes with label %s: %v", selector.String(), err)
		}
		nodesToUpgrade = nodes
	}

	return nodesToUpgrade, nil
}
