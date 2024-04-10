/*
Copyright 2015 The Kubernetes Authors.

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
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
	e2ekubelet "k8s.io/kubernetes/test/e2e/framework/kubelet"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2erc "k8s.io/kubernetes/test/e2e/framework/rc"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	e2essh "k8s.io/kubernetes/test/e2e/framework/ssh"
	e2evolume "k8s.io/kubernetes/test/e2e/framework/volume"
	testutils "k8s.io/kubernetes/test/utils"
	imageutils "k8s.io/kubernetes/test/utils/image"
	admissionapi "k8s.io/pod-security-admission/api"

	"github.com/onsi/ginkgo/v2"
)

const (
	// Interval to framework.Poll /runningpods on a node
	pollInterval = 1 * time.Second
	// Interval to framework.Poll /stats/container on a node
	containerStatsPollingInterval = 5 * time.Second
	// Maximum number of nodes that we constraint to
	maxNodesToCheck = 10
)

// getPodMatches returns a set of pod names on the given node that matches the
// podNamePrefix and namespace.
func getPodMatches(ctx context.Context, c clientset.Interface, nodeName string, podNamePrefix string, namespace string) sets.String {
	matches := sets.NewString()
	framework.Logf("Checking pods on node %v via /runningpods endpoint", nodeName)
	runningPods, err := e2ekubelet.GetKubeletPods(ctx, c, nodeName)
	if err != nil {
		framework.Logf("Error checking running pods on %v: %v", nodeName, err)
		return matches
	}
	for _, pod := range runningPods.Items {
		if pod.Namespace == namespace && strings.HasPrefix(pod.Name, podNamePrefix) {
			matches.Insert(pod.Name)
		}
	}
	return matches
}

// waitTillNPodsRunningOnNodes polls the /runningpods endpoint on kubelet until
// it finds targetNumPods pods that match the given criteria (namespace and
// podNamePrefix). Note that we usually use label selector to filter pods that
// belong to the same RC. However, we use podNamePrefix with namespace here
// because pods returned from /runningpods do not contain the original label
// information; they are reconstructed by examining the container runtime. In
// the scope of this test, we do not expect pod naming conflicts so
// podNamePrefix should be sufficient to identify the pods.
func waitTillNPodsRunningOnNodes(ctx context.Context, c clientset.Interface, nodeNames sets.String, podNamePrefix string, namespace string, targetNumPods int, timeout time.Duration) error {
	return wait.PollWithContext(ctx, pollInterval, timeout, func(ctx context.Context) (bool, error) {
		matchCh := make(chan sets.String, len(nodeNames))
		for _, item := range nodeNames.List() {
			// Launch a goroutine per node to check the pods running on the nodes.
			nodeName := item
			go func() {
				matchCh <- getPodMatches(ctx, c, nodeName, podNamePrefix, namespace)
			}()
		}

		seen := sets.NewString()
		for i := 0; i < len(nodeNames.List()); i++ {
			seen = seen.Union(<-matchCh)
		}
		if seen.Len() == targetNumPods {
			return true, nil
		}
		framework.Logf("Waiting for %d pods to be running on the node; %d are currently running;", targetNumPods, seen.Len())
		return false, nil
	})
}

// Restart the passed-in nfs-server by issuing a `/usr/sbin/rpc.nfsd 1` command in the
// pod's (only) container. This command changes the number of nfs server threads from
// (presumably) zero back to 1, and therefore allows nfs to open connections again.
func restartNfsServer(serverPod *v1.Pod) {
	const startcmd = "/usr/sbin/rpc.nfsd 1"
	ns := fmt.Sprintf("--namespace=%v", serverPod.Namespace)
	e2ekubectl.RunKubectlOrDie(ns, "exec", ns, serverPod.Name, "--", "/bin/sh", "-c", startcmd)
}

// Stop the passed-in nfs-server by issuing a `/usr/sbin/rpc.nfsd 0` command in the
// pod's (only) container. This command changes the number of nfs server threads to 0,
// thus closing all open nfs connections.
func stopNfsServer(serverPod *v1.Pod) {
	const stopcmd = "/usr/sbin/rpc.nfsd 0"
	ns := fmt.Sprintf("--namespace=%v", serverPod.Namespace)
	e2ekubectl.RunKubectlOrDie(ns, "exec", ns, serverPod.Name, "--", "/bin/sh", "-c", stopcmd)
}

// Creates a pod that mounts an nfs volume that is served by the nfs-server pod. The container
// will execute the passed in shell cmd. Waits for the pod to start.
// Note: the nfs plugin is defined inline, no PV or PVC.
func createPodUsingNfs(ctx context.Context, f *framework.Framework, c clientset.Interface, ns, nfsIP, cmd string) *v1.Pod {
	ginkgo.By("create pod using nfs volume")

	isPrivileged := true
	cmdLine := []string{"-c", cmd}
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pod-nfs-vol-",
			Namespace:    ns,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "pod-nfs-vol",
					Image:   imageutils.GetE2EImage(imageutils.BusyBox),
					Command: []string{"/bin/sh"},
					Args:    cmdLine,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "nfs-vol",
							MountPath: "/mnt",
						},
					},
					SecurityContext: &v1.SecurityContext{
						Privileged: &isPrivileged,
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever, //don't restart pod
			Volumes: []v1.Volume{
				{
					Name: "nfs-vol",
					VolumeSource: v1.VolumeSource{
						NFS: &v1.NFSVolumeSource{
							Server:   nfsIP,
							Path:     "/exports",
							ReadOnly: false,
						},
					},
				},
			},
		},
	}
	rtnPod, err := c.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	err = e2epod.WaitTimeoutForPodReadyInNamespace(ctx, f.ClientSet, rtnPod.Name, f.Namespace.Name, framework.PodStartTimeout) // running & ready
	framework.ExpectNoError(err)

	rtnPod, err = c.CoreV1().Pods(ns).Get(ctx, rtnPod.Name, metav1.GetOptions{}) // return fresh pod
	framework.ExpectNoError(err)
	return rtnPod
}

// getHostExternalAddress gets the node for a pod and returns the first External
// address. Returns an error if the node the pod is on doesn't have an External
// address.
func getHostExternalAddress(ctx context.Context, client clientset.Interface, p *v1.Pod) (externalAddress string, err error) {
	node, err := client.CoreV1().Nodes().Get(ctx, p.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP {
			if address.Address != "" {
				externalAddress = address.Address
				break
			}
		}
	}
	if externalAddress == "" {
		err = fmt.Errorf("No external address for pod %v on node %v",
			p.Name, p.Spec.NodeName)
	}
	return
}

// Checks for a lingering nfs mount and/or uid directory on the pod's host. The host IP is used
// so that this test runs in GCE, where it appears that SSH cannot resolve the hostname.
// If expectClean is true then we expect the node to be cleaned up and thus commands like
// `ls <uid-dir>` should fail (since that dir was removed). If expectClean is false then we expect
// the node is not cleaned up, and thus cmds like `ls <uid-dir>` should succeed. We wait for the
// kubelet to be cleaned up, afterwhich an error is reported.
func checkPodCleanup(ctx context.Context, c clientset.Interface, pod *v1.Pod, expectClean bool) {
	timeout := 5 * time.Minute
	poll := 20 * time.Second
	podDir := filepath.Join("/var/lib/kubelet/pods", string(pod.UID))
	mountDir := filepath.Join(podDir, "volumes", "kubernetes.io~nfs")
	// use ip rather than hostname in GCE
	nodeIP, err := getHostExternalAddress(ctx, c, pod)
	framework.ExpectNoError(err)

	condMsg := "deleted"
	if !expectClean {
		condMsg = "present"
	}

	// table of host tests to perform (order may matter so not using a map)
	type testT struct {
		feature string // feature to test
		cmd     string // remote command to execute on node
	}
	tests := []testT{
		{
			feature: "pod UID directory",
			cmd:     fmt.Sprintf("sudo ls %v", podDir),
		},
		{
			feature: "pod nfs mount",
			cmd:     fmt.Sprintf("sudo mount | grep %v", mountDir),
		},
	}

	for _, test := range tests {
		framework.Logf("Wait up to %v for host's (%v) %q to be %v", timeout, nodeIP, test.feature, condMsg)
		err = wait.PollWithContext(ctx, poll, timeout, func(ctx context.Context) (bool, error) {
			result, err := e2essh.NodeExec(ctx, nodeIP, test.cmd, framework.TestContext.Provider)
			framework.ExpectNoError(err)
			e2essh.LogResult(result)
			ok := (result.Code == 0 && len(result.Stdout) > 0 && len(result.Stderr) == 0)
			if expectClean && ok { // keep trying
				return false, nil
			}
			if !expectClean && !ok { // stop wait loop
				return true, fmt.Errorf("%v is gone but expected to exist", test.feature)
			}
			return true, nil // done, host is as expected
		})
		framework.ExpectNoError(err, fmt.Sprintf("Host (%v) cleanup error: %v. Expected %q to be %v", nodeIP, err, test.feature, condMsg))
	}

	if expectClean {
		framework.Logf("Pod's host has been cleaned up")
	} else {
		framework.Logf("Pod's host has not been cleaned up (per expectation)")
	}
}

var _ = SIGDescribe("kubelet", func() {
	var (
		c  clientset.Interface
		ns string
	)
	f := framework.NewDefaultFramework("kubelet")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged

	ginkgo.BeforeEach(func() {
		c = f.ClientSet
		ns = f.Namespace.Name
	})

	ginkgo.Describe("Clean up pods on node", func() {
		var (
			numNodes        int
			nodeNames       sets.String
			nodeLabels      map[string]string
			resourceMonitor *e2ekubelet.ResourceMonitor
		)
		type DeleteTest struct {
			podsPerNode int
			timeout     time.Duration
		}

		deleteTests := []DeleteTest{
			{podsPerNode: 10, timeout: 1 * time.Minute},
		}

		// Must be called in each It with the context of the test.
		start := func(ctx context.Context) {
			// Use node labels to restrict the pods to be assigned only to the
			// nodes we observe initially.
			nodeLabels = make(map[string]string)
			nodeLabels["kubelet_cleanup"] = "true"
			nodes, err := e2enode.GetBoundedReadySchedulableNodes(ctx, c, maxNodesToCheck)
			numNodes = len(nodes.Items)
			framework.ExpectNoError(err)
			nodeNames = sets.NewString()
			for i := 0; i < len(nodes.Items); i++ {
				nodeNames.Insert(nodes.Items[i].Name)
			}
			for nodeName := range nodeNames {
				for k, v := range nodeLabels {
					e2enode.AddOrUpdateLabelOnNode(c, nodeName, k, v)
					ginkgo.DeferCleanup(e2enode.RemoveLabelOffNode, c, nodeName, k)
				}
			}

			// While we only use a bounded number of nodes in the test. We need to know
			// the actual number of nodes in the cluster, to avoid running resourceMonitor
			// against large clusters.
			actualNodes, err := e2enode.GetReadySchedulableNodes(ctx, c)
			framework.ExpectNoError(err)

			// Start resourceMonitor only in small clusters.
			if len(actualNodes.Items) <= maxNodesToCheck {
				resourceMonitor = e2ekubelet.NewResourceMonitor(f.ClientSet, e2ekubelet.TargetContainers(), containerStatsPollingInterval)
				resourceMonitor.Start(ctx)
				ginkgo.DeferCleanup(resourceMonitor.Stop)
			}
		}

		for _, itArg := range deleteTests {
			name := fmt.Sprintf(
				"kubelet should be able to delete %d pods per node in %v.", itArg.podsPerNode, itArg.timeout)
			itArg := itArg
			ginkgo.It(name, func(ctx context.Context) {
				start(ctx)
				totalPods := itArg.podsPerNode * numNodes
				ginkgo.By(fmt.Sprintf("Creating a RC of %d pods and wait until all pods of this RC are running", totalPods))
				rcName := fmt.Sprintf("cleanup%d-%s", totalPods, string(uuid.NewUUID()))

				err := e2erc.RunRC(ctx, testutils.RCConfig{
					Client:       f.ClientSet,
					Name:         rcName,
					Namespace:    f.Namespace.Name,
					Image:        imageutils.GetPauseImageName(),
					Replicas:     totalPods,
					NodeSelector: nodeLabels,
				})
				framework.ExpectNoError(err)
				// Perform a sanity check so that we know all desired pods are
				// running on the nodes according to kubelet. The timeout is set to
				// only 30 seconds here because e2erc.RunRC already waited for all pods to
				// transition to the running status.
				err = waitTillNPodsRunningOnNodes(ctx, f.ClientSet, nodeNames, rcName, ns, totalPods, time.Second*30)
				framework.ExpectNoError(err)
				if resourceMonitor != nil {
					resourceMonitor.LogLatest()
				}

				ginkgo.By("Deleting the RC")
				e2erc.DeleteRCAndWaitForGC(ctx, f.ClientSet, f.Namespace.Name, rcName)
				// Check that the pods really are gone by querying /runningpods on the
				// node. The /runningpods handler checks the container runtime (or its
				// cache) and  returns a list of running pods. Some possible causes of
				// failures are:
				//   - kubelet deadlock
				//   - a bug in graceful termination (if it is enabled)
				//   - docker slow to delete pods (or resource problems causing slowness)
				start := time.Now()
				err = waitTillNPodsRunningOnNodes(ctx, f.ClientSet, nodeNames, rcName, ns, 0, itArg.timeout)
				framework.ExpectNoError(err)
				framework.Logf("Deleting %d pods on %d nodes completed in %v after the RC was deleted", totalPods, len(nodeNames),
					time.Since(start))
				if resourceMonitor != nil {
					resourceMonitor.LogCPUSummary()
				}
			})
		}
	})

	// Test host cleanup when disrupting the volume environment.
	ginkgo.Describe("host cleanup with volume mounts [HostCleanup][Flaky]", func() {

		type hostCleanupTest struct {
			itDescr string
			podCmd  string
		}

		// Disrupt the nfs-server pod after a client pod accesses the nfs volume.
		// Note: the nfs-server is stopped NOT deleted. This is done to preserve its ip addr.
		//       If the nfs-server pod is deleted the client pod's mount can not be unmounted.
		//       If the nfs-server pod is deleted and re-created, due to having a different ip
		//       addr, the client pod's mount still cannot be unmounted.
		ginkgo.Context("Host cleanup after disrupting NFS volume [NFS]", func() {
			// issue #31272
			var (
				nfsServerPod *v1.Pod
				nfsIP        string
				pod          *v1.Pod // client pod
			)

			// fill in test slice for this context
			testTbl := []hostCleanupTest{
				{
					itDescr: "after stopping the nfs-server and deleting the (sleeping) client pod, the NFS mount and the pod's UID directory should be removed.",
					podCmd:  "sleep 6000", // keep pod running
				},
				{
					itDescr: "after stopping the nfs-server and deleting the (active) client pod, the NFS mount and the pod's UID directory should be removed.",
					podCmd:  "while true; do echo FeFieFoFum >>/mnt/SUCCESS; sleep 1; cat /mnt/SUCCESS; done",
				},
			}

			ginkgo.BeforeEach(func(ctx context.Context) {
				e2eskipper.SkipUnlessProviderIs(framework.ProvidersWithSSH...)
				_, nfsServerPod, nfsIP = e2evolume.NewNFSServer(ctx, c, ns, []string{"-G", "777", "/exports"})
			})

			ginkgo.AfterEach(func(ctx context.Context) {
				err := e2epod.DeletePodWithWait(ctx, c, pod)
				framework.ExpectNoError(err, "AfterEach: Failed to delete client pod ", pod.Name)
				err = e2epod.DeletePodWithWait(ctx, c, nfsServerPod)
				framework.ExpectNoError(err, "AfterEach: Failed to delete server pod ", nfsServerPod.Name)
			})

			// execute It blocks from above table of tests
			for _, t := range testTbl {
				t := t
				ginkgo.It(t.itDescr, func(ctx context.Context) {
					pod = createPodUsingNfs(ctx, f, c, ns, nfsIP, t.podCmd)

					ginkgo.By("Stop the NFS server")
					stopNfsServer(nfsServerPod)

					ginkgo.By("Delete the pod mounted to the NFS volume -- expect failure")
					err := e2epod.DeletePodWithWait(ctx, c, pod)
					framework.ExpectError(err)
					// pod object is now stale, but is intentionally not nil

					ginkgo.By("Check if pod's host has been cleaned up -- expect not")
					checkPodCleanup(ctx, c, pod, false)

					ginkgo.By("Restart the nfs server")
					restartNfsServer(nfsServerPod)

					ginkgo.By("Verify that the deleted client pod is now cleaned up")
					checkPodCleanup(ctx, c, pod, true)
				})
			}
		})
	})

	// Tests for NodeLogQuery feature
	ginkgo.Describe("kubectl get --raw \"/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=/<insert-log-file-name-here> [Feature:NodeLogQuery] [LinuxOnly]", func() {
		var (
			numNodes  int
			nodeNames sets.String
		)

		ginkgo.BeforeEach(func(ctx context.Context) {
			nodes, err := e2enode.GetBoundedReadySchedulableNodes(ctx, c, maxNodesToCheck)
			numNodes = len(nodes.Items)
			framework.ExpectNoError(err)
			nodeNames = sets.NewString()
			for i := 0; i < numNodes; i++ {
				nodeNames.Insert(nodes.Items[i].Name)
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?"
			returns an error or not!
		*/

		ginkgo.It("should return the error of running without --query option", func() {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)

			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?"
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				_, _, err := framework.StartCmdAndStreamOutput(cmd)
				if err != nil {
					framework.Failf("Failed to start kubectl command! Error: %v", err)
				}
				err = cmd.Wait()
				if err == nil {
					framework.Failf("Command kubectl get --raw" + queryCommand + " was expected to return an error!")
				}
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet"
			returns the kubelet logs or not!
		*/

		ginkgo.It("should return the logs ", func(ctx context.Context) {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)

			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				assertContains("kubelet", result)
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&pattern=container"
			returns something or not!
		*/

		ginkgo.It("should return the logs filtered by pattern container", func(ctx context.Context) {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)

			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&pattern=container\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				assertContains("container", result)
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&boot=0"
			returns something or not!
		*/

		ginkgo.It("should return the kubelet logs for the current boot", func(ctx context.Context) {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)

			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&boot=0\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				assertContains("kubelet", result)
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&boot=-1"
			returns something or not!
		*/

		ginkgo.It("should return the kubelet logs for the previous boot", func(ctx context.Context) {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)

			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&boot=-1\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				assertContains("kubelet", result)
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&tailLines=2"
			returns something or not!
		*/

		ginkgo.It("should return the kubelet logs for the previous boot", func(ctx context.Context) {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)
			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&tailLines=3\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				logs := journalctlCommandOnNode(nodeName, "-u kubelet -n 3 --utc")
				if result != logs {
					framework.Failf("Failed to receive the correct kubelet logs or the correct amount of lines of logs")
				}
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&tailLines=2&pattern=kubelet&boot=0"
			returns something or not!
		*/

		ginkgo.It("should return the kubelet logs for the current boot with pattern error and output in 2 lines", func(ctx context.Context) {
			ginkgo.By("Starting the command")
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)
			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&tailLines=3&boot=0&pattern=kubelet\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				logs := journalctlCommandOnNode(nodeName, "-u kubelet -n 3 --utc")
				if result != logs {
					framework.Failf("Failed to receive the correct kubelet logs")
				}
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&tailLines=3&untilTime=<now>"
			returns the kubelet logs or not!
		*/

		ginkgo.It("should return the logs for the requested query until the current date and time", func() {
			ginkgo.By("Starting the command")
			start := time.Now().UTC()
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)
			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&tailLines=3&sinceTime=" + start.Format(time.RFC3339) + "\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				logs := journalctlCommandOnNode(nodeName, "-u kubelet -n 3 --utc")
				if result != logs {
					framework.Failf("Failed to receive the correct kubelet logs or the correct amount of lines of logs")
				}
			}
		})

		/*
			Test if kubectl get --raw "/api/v1/nodes/<insert-node-name-here>/proxy/logs/?query=kubelet&tailLines=3&sinceTime=4 hours ago"
			returns the kubelet logs or not!
		*/

		ginkgo.It("should return the logs for the requested query since the current date and time", func() {
			ginkgo.By("Starting the command")
			start := time.Now().UTC()
			start = start.Add(time.Duration(-4) * time.Hour)
			tk := e2ekubectl.NewTestKubeconfig(framework.TestContext.CertDir, framework.TestContext.Host, framework.TestContext.KubeConfig, framework.TestContext.KubeContext, framework.TestContext.KubectlPath, ns)
			for nodeName := range nodeNames {
				queryCommand := "\"/api/v1/nodes/" + nodeName + "/proxy/logs/?query=kubelet&tailLines=3&sinceTime=" + start.Format(time.RFC3339) + "\""
				cmd := tk.KubectlCmd("get", "--raw", queryCommand)
				result := runKubectlCommand(cmd)
				logs := journalctlCommandOnNode(nodeName, "-u kubelet --utc")
				assertContains(result, logs)
			}
		})
	})
})

func runKubectlCommand(cmd *exec.Cmd) (result string) {
	stdout, stderr, err := framework.StartCmdAndStreamOutput(cmd)
	var buf bytes.Buffer
	if err != nil {
		framework.Failf("Failed to start kubectl command! Stderr: %v, error: %v", stderr, err)
	}
	defer stdout.Close()
	defer stderr.Close()
	defer framework.TryKill(cmd)

	b_read, err := io.Copy(&buf, stdout)
	if err != nil {
		framework.Failf("Expected output from kubectl alpha node-logs %s: %v\n Stderr: %v", cmd.Args, err, stderr)
	}
	out := ""
	if b_read >= 0 {
		out = buf.String()
	}

	framework.Logf("Kubectl output: %s", out)
	return out
}

func assertContains(expectedString string, result string) {
	if strings.Contains(result, expectedString) {
		return
	}
	framework.Failf("Failed to find \"%s\"", expectedString)
	return
}

func journalctlCommandOnNode(nodeName string, args string) string {
	result, err := e2essh.NodeExec(context.Background(), nodeName, "journalctl "+args, framework.TestContext.Provider)
	framework.ExpectNoError(err)
	e2essh.LogResult(result)
	return result.Stdout
}
