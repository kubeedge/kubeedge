/*
Copyright 2016 The Kubernetes Authors.

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

package node

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	imageutils "k8s.io/kubernetes/test/utils/image"
	admissionapi "k8s.io/pod-security-admission/api"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = SIGDescribe("Container Lifecycle Hook", func() {
	f := framework.NewDefaultFramework("container-lifecycle-hook")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelBaseline
	var podClient *e2epod.PodClient
	const (
		podCheckInterval     = 1 * time.Second
		postStartWaitTimeout = 2 * time.Minute
		preStopWaitTimeout   = 30 * time.Second
	)
	ginkgo.Context("when create a pod with lifecycle hook", func() {
		var (
			targetIP, targetURL, targetNode string

			httpPorts = []v1.ContainerPort{
				{
					ContainerPort: 8080,
					Protocol:      v1.ProtocolTCP,
				},
			}
			httpsPorts = []v1.ContainerPort{
				{
					ContainerPort: 9090,
					Protocol:      v1.ProtocolTCP,
				},
			}
			httpsArgs = []string{
				"netexec",
				"--http-port", "9090",
				"--udp-port", "9091",
				"--tls-cert-file", "/localhost.crt",
				"--tls-private-key-file", "/localhost.key",
			}
		)

		podHandleHookRequest := e2epod.NewAgnhostPodFromContainers(
			"", "pod-handle-http-request", nil,
			e2epod.NewAgnhostContainer("container-handle-http-request", nil, httpPorts, "netexec"),
			e2epod.NewAgnhostContainer("container-handle-https-request", nil, httpsPorts, httpsArgs...),
		)

		ginkgo.BeforeEach(func(ctx context.Context) {
			node, err := e2enode.GetRandomReadySchedulableNode(ctx, f.ClientSet)
			framework.ExpectNoError(err)
			targetNode = node.Name
			nodeSelection := e2epod.NodeSelection{}
			e2epod.SetAffinity(&nodeSelection, targetNode)
			e2epod.SetNodeSelection(&podHandleHookRequest.Spec, nodeSelection)

			podClient = e2epod.NewPodClient(f)
			ginkgo.By("create the container to handle the HTTPGet hook request.")
			newPod := podClient.CreateSync(ctx, podHandleHookRequest)
			targetIP = newPod.Status.PodIP
			targetURL = targetIP
			if strings.Contains(targetIP, ":") {
				targetURL = fmt.Sprintf("[%s]", targetIP)
			}
		})
		testPodWithHook := func(ctx context.Context, podWithHook *v1.Pod) {
			ginkgo.By("create the pod with lifecycle hook")
			podClient.CreateSync(ctx, podWithHook)
			const (
				defaultHandler = iota
				httpsHandler
			)
			handlerContainer := defaultHandler
			if podWithHook.Spec.Containers[0].Lifecycle.PostStart != nil {
				ginkgo.By("check poststart hook")
				if podWithHook.Spec.Containers[0].Lifecycle.PostStart.HTTPGet != nil {
					if v1.URISchemeHTTPS == podWithHook.Spec.Containers[0].Lifecycle.PostStart.HTTPGet.Scheme {
						handlerContainer = httpsHandler
					}
				}
				gomega.Eventually(ctx, func(ctx context.Context) error {
					return podClient.MatchContainerOutput(ctx, podHandleHookRequest.Name, podHandleHookRequest.Spec.Containers[handlerContainer].Name,
						`GET /echo\?msg=poststart`)
				}, postStartWaitTimeout, podCheckInterval).Should(gomega.BeNil())
			}
			ginkgo.By("delete the pod with lifecycle hook")
			podClient.DeleteSync(ctx, podWithHook.Name, *metav1.NewDeleteOptions(15), e2epod.DefaultPodDeletionTimeout)
			if podWithHook.Spec.Containers[0].Lifecycle.PreStop != nil {
				ginkgo.By("check prestop hook")
				if podWithHook.Spec.Containers[0].Lifecycle.PreStop.HTTPGet != nil {
					if v1.URISchemeHTTPS == podWithHook.Spec.Containers[0].Lifecycle.PreStop.HTTPGet.Scheme {
						handlerContainer = httpsHandler
					}
				}
				gomega.Eventually(ctx, func(ctx context.Context) error {
					return podClient.MatchContainerOutput(ctx, podHandleHookRequest.Name, podHandleHookRequest.Spec.Containers[handlerContainer].Name,
						`GET /echo\?msg=prestop`)
				}, preStopWaitTimeout, podCheckInterval).Should(gomega.BeNil())
			}
		}
		/*
			Release: v1.9
			Testname: Pod Lifecycle, post start exec hook
			Description: When a post start handler is specified in the container lifecycle using a 'Exec' action, then the handler MUST be invoked after the start of the container. A server pod is created that will serve http requests, create a second pod with a container lifecycle specifying a post start that invokes the server pod using ExecAction to validate that the post start is executed.
		*/
		framework.ConformanceIt("should execute poststart exec hook properly [NodeConformance]", func(ctx context.Context) {
			lifecycle := &v1.Lifecycle{
				PostStart: &v1.LifecycleHandler{
					Exec: &v1.ExecAction{
						Command: []string{"sh", "-c", "curl http://" + targetURL + ":8080/echo?msg=poststart"},
					},
				},
			}
			podWithHook := getPodWithHook("pod-with-poststart-exec-hook", imageutils.GetE2EImage(imageutils.Agnhost), lifecycle)

			testPodWithHook(ctx, podWithHook)
		})
		/*
			Release: v1.9
			Testname: Pod Lifecycle, prestop exec hook
			Description: When a pre-stop handler is specified in the container lifecycle using a 'Exec' action, then the handler MUST be invoked before the container is terminated. A server pod is created that will serve http requests, create a second pod with a container lifecycle specifying a pre-stop that invokes the server pod using ExecAction to validate that the pre-stop is executed.
		*/
		framework.ConformanceIt("should execute prestop exec hook properly [NodeConformance]", func(ctx context.Context) {
			lifecycle := &v1.Lifecycle{
				PreStop: &v1.LifecycleHandler{
					Exec: &v1.ExecAction{
						Command: []string{"sh", "-c", "curl http://" + targetURL + ":8080/echo?msg=prestop"},
					},
				},
			}
			podWithHook := getPodWithHook("pod-with-prestop-exec-hook", imageutils.GetE2EImage(imageutils.Agnhost), lifecycle)
			testPodWithHook(ctx, podWithHook)
		})
		/*
			Release: v1.9
			Testname: Pod Lifecycle, post start http hook
			Description: When a post start handler is specified in the container lifecycle using a HttpGet action, then the handler MUST be invoked after the start of the container. A server pod is created that will serve http requests, create a second pod on the same node with a container lifecycle specifying a post start that invokes the server pod to validate that the post start is executed.
		*/
		framework.ConformanceIt("should execute poststart http hook properly [NodeConformance]", func(ctx context.Context) {
			lifecycle := &v1.Lifecycle{
				PostStart: &v1.LifecycleHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/echo?msg=poststart",
						Host: targetIP,
						Port: intstr.FromInt(8080),
					},
				},
			}
			podWithHook := getPodWithHook("pod-with-poststart-http-hook", imageutils.GetPauseImageName(), lifecycle)
			// make sure we spawn the test pod on the same node as the webserver.
			nodeSelection := e2epod.NodeSelection{}
			e2epod.SetAffinity(&nodeSelection, targetNode)
			e2epod.SetNodeSelection(&podWithHook.Spec, nodeSelection)
			testPodWithHook(ctx, podWithHook)
		})
		/*
			Release : v1.23
			Testname: Pod Lifecycle, poststart https hook
			Description: When a post-start handler is specified in the container lifecycle using a 'HttpGet' action, then the handler MUST be invoked before the container is terminated. A server pod is created that will serve https requests, create a second pod on the same node with a container lifecycle specifying a post-start that invokes the server pod to validate that the post-start is executed.
		*/
		ginkgo.It("should execute poststart https hook properly [MinimumKubeletVersion:1.23] [NodeConformance]", func(ctx context.Context) {
			lifecycle := &v1.Lifecycle{
				PostStart: &v1.LifecycleHandler{
					HTTPGet: &v1.HTTPGetAction{
						Scheme: v1.URISchemeHTTPS,
						Path:   "/echo?msg=poststart",
						Host:   targetIP,
						Port:   intstr.FromInt(9090),
					},
				},
			}
			podWithHook := getPodWithHook("pod-with-poststart-https-hook", imageutils.GetPauseImageName(), lifecycle)
			// make sure we spawn the test pod on the same node as the webserver.
			nodeSelection := e2epod.NodeSelection{}
			e2epod.SetAffinity(&nodeSelection, targetNode)
			e2epod.SetNodeSelection(&podWithHook.Spec, nodeSelection)
			testPodWithHook(ctx, podWithHook)
		})
		/*
			Release : v1.9
			Testname: Pod Lifecycle, prestop http hook
			Description: When a pre-stop handler is specified in the container lifecycle using a 'HttpGet' action, then the handler MUST be invoked before the container is terminated. A server pod is created that will serve http requests, create a second pod on the same node with a container lifecycle specifying a pre-stop that invokes the server pod to validate that the pre-stop is executed.
		*/
		framework.ConformanceIt("should execute prestop http hook properly [NodeConformance]", func(ctx context.Context) {
			lifecycle := &v1.Lifecycle{
				PreStop: &v1.LifecycleHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/echo?msg=prestop",
						Host: targetIP,
						Port: intstr.FromInt(8080),
					},
				},
			}
			podWithHook := getPodWithHook("pod-with-prestop-http-hook", imageutils.GetPauseImageName(), lifecycle)
			// make sure we spawn the test pod on the same node as the webserver.
			nodeSelection := e2epod.NodeSelection{}
			e2epod.SetAffinity(&nodeSelection, targetNode)
			e2epod.SetNodeSelection(&podWithHook.Spec, nodeSelection)
			testPodWithHook(ctx, podWithHook)
		})
		/*
			Release : v1.23
			Testname: Pod Lifecycle, prestop https hook
			Description: When a pre-stop handler is specified in the container lifecycle using a 'HttpGet' action, then the handler MUST be invoked before the container is terminated. A server pod is created that will serve https requests, create a second pod on the same node with a container lifecycle specifying a pre-stop that invokes the server pod to validate that the pre-stop is executed.
		*/
		ginkgo.It("should execute prestop https hook properly [MinimumKubeletVersion:1.23] [NodeConformance]", func(ctx context.Context) {
			lifecycle := &v1.Lifecycle{
				PreStop: &v1.LifecycleHandler{
					HTTPGet: &v1.HTTPGetAction{
						Scheme: v1.URISchemeHTTPS,
						Path:   "/echo?msg=prestop",
						Host:   targetIP,
						Port:   intstr.FromInt(9090),
					},
				},
			}
			podWithHook := getPodWithHook("pod-with-prestop-https-hook", imageutils.GetPauseImageName(), lifecycle)
			// make sure we spawn the test pod on the same node as the webserver.
			nodeSelection := e2epod.NodeSelection{}
			e2epod.SetAffinity(&nodeSelection, targetNode)
			e2epod.SetNodeSelection(&podWithHook.Spec, nodeSelection)
			testPodWithHook(ctx, podWithHook)
		})
	})
})

func getPodWithHook(name string, image string, lifecycle *v1.Lifecycle) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:      name,
					Image:     image,
					Lifecycle: lifecycle,
				},
			},
		},
	}
}
