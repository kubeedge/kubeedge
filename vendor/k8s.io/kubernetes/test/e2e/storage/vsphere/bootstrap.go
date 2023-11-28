/*
Copyright 2018 The Kubernetes Authors.

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

package vsphere

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
)

var once sync.Once
var waiting = make(chan bool)

// Bootstrap takes care of initializing necessary test context for vSphere tests
func Bootstrap(fw *framework.Framework) {
	done := make(chan bool)
	go func() {
		once.Do(func() {
			bootstrapOnce(fw)
		})
		<-waiting
		done <- true
	}()
	<-done
}

func bootstrapOnce(f *framework.Framework) {
	// 1. Read vSphere conf and get VSphere instances
	vsphereInstances, err := GetVSphereInstances()
	if err != nil {
		framework.Failf("Failed to bootstrap vSphere with error: %v", err)
	}
	// 2. Get all nodes
	nodeList, err := f.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		framework.Failf("Failed to get nodes: %v", err)
	}
	TestContext = Context{NodeMapper: NewNodeMapper(), VSphereInstances: vsphereInstances}
	// 3. Get Node to VSphere mapping
	err = TestContext.NodeMapper.GenerateNodeMap(vsphereInstances, *nodeList)
	if err != nil {
		framework.Failf("Failed to bootstrap vSphere with error: %v", err)
	}
	// 4. Generate Zone to Datastore mapping
	err = TestContext.NodeMapper.GenerateZoneToDatastoreMap()
	if err != nil {
		framework.Failf("Failed to generate zone to datastore mapping with error: %v", err)
	}
	close(waiting)
}
