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

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

const (
	kubeConfigEnvKey = "KUBECONFIG"

	edgeNodeLabelKey = "node-role.kubernetes.io/edge"
)

type Tests struct {
	TestName    string `yaml:"testname"`
	CodeName    string `yaml:"codename"`
	Description string `yaml:"description"`
	Release     string `yaml:"release"`
	File        string `yaml:"file"`
}

func GetEnvWithDefault(envKey, defaultValue string) string {
	value := os.Getenv(envKey)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func SkipCommands() ([]string, error) {
	tests, err := skipCases()
	if err != nil {
		return nil, err
	}

	var skipCommands []string
	for _, test := range tests {
		skipCommands = append(skipCommands, test.CodeName)
	}

	return skipCommands, nil
}

func skipCases() ([]Tests, error) {
	data, err := Read("/testdata/edge_skip_case.yaml")
	if err != nil {
		return nil, fmt.Errorf("read skip test case err: %v", err)
	}

	var skipTests []Tests

	if err := yaml.Unmarshal(data, &skipTests); err != nil {
		return nil, fmt.Errorf("unmarshal skip test case err: %v", err)
	}

	return skipTests, err
}

func Read(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		// Not an error (yet), some other provider may have the file.
		return nil, nil
	}
	return data, err
}

func CmdInfo(cmd *exec.Cmd) string {
	return fmt.Sprintf(
		`Command env: %v
Run from directory: %v
Executable path: %v
Args (comma-delimited): %v`, cmd.Env, cmd.Dir, cmd.Path, strings.Join(cmd.Args, ","),
	)
}

// tempTaints is temporarily added to center node when run kubeEdge conformance
// to make sure that all the pod created by conformance to run on the edge node
var tempTaints = &v1.Taint{
	Key:    "node.kubeedge.io/conformance",
	Value:  "remove-when-completed",
	Effect: v1.TaintEffectNoSchedule,
}

var updateTaintBackoff = wait.Backoff{
	Steps:    5,
	Duration: 100 * time.Millisecond,
	Jitter:   1.0,
}

// BeforeRunConformance do prepare work before run conformance
func BeforeRunConformance() error {
	kubeClient, err := getKubeClient()
	if err != nil {
		return err
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodeList.Items {
		if isEdgeNode(node) {
			continue
		}

		err = addConformanceTaintOnNode(kubeClient, &node)
		if err != nil {
			return err
		}
	}

	return nil
}

// AfterRunConformance do clean work after conformance done
func AfterRunConformance() error {
	kubeClient, err := getKubeClient()
	if err != nil {
		return err
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodeList.Items {
		if isEdgeNode(node) {
			continue
		}

		err = deleteConformanceTaintOnNode(kubeClient, &node)
		if err != nil {
			log.Printf("failed delete taint for node:%v\n", node.Name)
		}
	}

	return nil
}

func addConformanceTaintOnNode(c kubernetes.Interface, node *v1.Node) error {
	newNode, updated := addTaint(node, tempTaints)
	if !updated {
		return nil
	}

	return retry.RetryOnConflict(updateTaintBackoff, func() error {
		return patchNodeTaints(c, node, newNode)
	})
}

func deleteConformanceTaintOnNode(c kubernetes.Interface, node *v1.Node) error {
	newNode, updated := removeTaint(node, tempTaints)
	if !updated {
		return nil
	}

	return retry.RetryOnConflict(updateTaintBackoff, func() error {
		return patchNodeTaints(c, node, newNode)
	})
}

func isEdgeNode(node v1.Node) bool {
	if node.Labels == nil {
		return false
	}

	_, ok := node.Labels[edgeNodeLabelKey]
	return ok
}

func getKubeClient() (kubernetes.Interface, error) {
	configPath := GetEnvWithDefault(kubeConfigEnvKey, "")
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}

	kubeConfig.ContentType = runtime.ContentTypeProtobuf
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	return kubeClient, nil
}

func addTaint(node *v1.Node, taint *v1.Taint) (*v1.Node, bool) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints

	var newTaints []v1.Taint
	for i := range nodeTaints {
		if taint.MatchTaint(&nodeTaints[i]) {
			log.Printf("taint already exist for node:%v\n", node.Name)
			return node, false
		}

		newTaints = append(newTaints, nodeTaints[i])
	}

	newTaints = append(newTaints, *taint)
	newNode.Spec.Taints = newTaints

	return newNode, true
}

func removeTaint(node *v1.Node, taintToDelete *v1.Taint) (*v1.Node, bool) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints
	if len(nodeTaints) == 0 {
		return newNode, false
	}

	var newTaints []v1.Taint
	deleted := false
	for i := range nodeTaints {
		if taintToDelete.MatchTaint(&nodeTaints[i]) {
			deleted = true
			continue
		}
		newTaints = append(newTaints, nodeTaints[i])
	}

	newNode.Spec.Taints = newTaints

	return newNode, deleted
}

func patchNodeTaints(c kubernetes.Interface, oldNode *v1.Node, newNode *v1.Node) error {
	oldData, err := json.Marshal(oldNode)
	if err != nil {
		return fmt.Errorf("failed to marshal old node %#v for node %q: %v", oldNode, oldNode.Name, err)
	}

	newTaints := newNode.Spec.Taints
	newNodeClone := oldNode.DeepCopy()
	newNodeClone.Spec.Taints = newTaints
	newData, err := json.Marshal(newNodeClone)
	if err != nil {
		return fmt.Errorf("failed to marshal new node %#v for node %q: %v", newNodeClone, oldNode.Name, err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1.Node{})
	if err != nil {
		return fmt.Errorf("failed to create patch for node %q: %v", oldNode.Name, err)
	}

	_, err = c.CoreV1().Nodes().Patch(context.TODO(), oldNode.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}
