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

package utils

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
)

func CreateMapperDeployment(c clientset.Interface, replica int32, deplName string) *corev1.PodList {
	ginkgo.By(fmt.Sprintf("create deployment %s", deplName))
	d := NewMapperDeployment(replica)
	_, err := CreateDeployment(c, d)
	gomega.Expect(err).To(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("get deployment %s", deplName))
	_, err = GetDeployment(c, corev1.NamespaceDefault, deplName)
	gomega.Expect(err).To(gomega.BeNil())

	time.Sleep(time.Second * 10)

	ginkgo.By(fmt.Sprintf("get pod for deployment %s", deplName))
	labelSelector := labels.SelectorFromSet(map[string]string{"app": deplName})
	podList, err := GetPods(c, corev1.NamespaceDefault, labelSelector, nil)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(podList).NotTo(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("wait for pod of deployment %s running", deplName))
	WaitForPodsRunning(c, podList, 240*time.Second)

	return podList
}

func DeleteMapperDeployment(c clientset.Interface) {
	ginkgo.By(fmt.Sprintf("get deployment %s", constants.MapperName))
	deployment, err := GetDeployment(c, corev1.NamespaceDefault, constants.MapperName)
	gomega.Expect(err).To(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("list pod for deploy %s", constants.MapperName))
	labelSelector := labels.SelectorFromSet(map[string]string{"app": constants.MapperName})
	_, err = GetPods(c, metav1.NamespaceDefault, labelSelector, nil)
	gomega.Expect(err).To(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("delete deploy %s", constants.MapperName))
	err = DeleteDeployment(c, deployment.Namespace, deployment.Name)
	gomega.Expect(err).To(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("wait for pod of deploy %s to disappear", constants.MapperName))
	err = WaitForPodsToDisappear(c, metav1.NamespaceDefault, labelSelector, constants.Interval, constants.Timeout)
	gomega.Expect(err).To(gomega.BeNil())
	PrintTestcaseNameandStatus()
}
