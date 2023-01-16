/*
Copyright 2019 The KubeEdge Authors.

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

package testsuite

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func CreateDeploymentTest(c clientset.Interface, replica int32, deplName string) *v1.PodList {
	ginkgo.By(fmt.Sprintf("create deployment %s", deplName))
	d := utils.NewDeployment(deplName, utils.LoadConfig().AppImageURL[1], replica)
	_, err := utils.CreateDeployment(c, d)
	gomega.Expect(err).To(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("get deployment %s", deplName))
	_, err = utils.GetDeployment(c, v1.NamespaceDefault, deplName)
	gomega.Expect(err).To(gomega.BeNil())

	time.Sleep(time.Second * 1)

	ginkgo.By(fmt.Sprintf("get pod for deployment %s", deplName))
	labelSelector := labels.SelectorFromSet(map[string]string{"app": deplName})
	podList, err := utils.GetPods(c, v1.NamespaceDefault, labelSelector, nil)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(podList).NotTo(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("wait for pod of deployment %s running", deplName))
	utils.WaitForPodsRunning(c, podList, 240*time.Second)

	return podList
}

func CreatePodTest(c clientset.Interface, pod *v1.Pod) *v1.PodList {
	ginkgo.By(fmt.Sprintf("create pod %s/%s", pod.Namespace, pod.Name))
	_, err := utils.CreatePod(c, pod)
	gomega.Expect(err).To(gomega.BeNil())

	time.Sleep(time.Second * 1)

	ginkgo.By("get pods")
	labelSelector := labels.SelectorFromSet(map[string]string{"app": pod.Name})
	podList, err := utils.GetPods(c, v1.NamespaceDefault, labelSelector, nil)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(podList).NotTo(gomega.BeNil())

	ginkgo.By(fmt.Sprintf("wait pod %s/%s running", pod.Namespace, pod.Name))
	utils.WaitForPodsRunning(c, podList, 240*time.Second)

	return podList
}
