/*
Copyright 2017 The Kubernetes Authors.

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
	"fmt"
	"strconv"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eoutput "k8s.io/kubernetes/test/e2e/framework/pod/output"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	admissionapi "k8s.io/pod-security-admission/api"
)

var _ = utils.SIGDescribe("Volume Placement [Feature:vsphere]", func() {
	f := framework.NewDefaultFramework("volume-placement")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	const (
		NodeLabelKey = "vsphere_e2e_label_volume_placement"
	)
	var (
		c                  clientset.Interface
		ns                 string
		volumePaths        []string
		node1Name          string
		node1KeyValueLabel map[string]string
		node2Name          string
		node2KeyValueLabel map[string]string
		nodeInfo           *NodeInfo
		vsp                *VSphere
	)
	ginkgo.BeforeEach(func(ctx context.Context) {
		e2eskipper.SkipUnlessProviderIs("vsphere")
		Bootstrap(f)
		c = f.ClientSet
		ns = f.Namespace.Name
		framework.ExpectNoError(e2enode.WaitForAllNodesSchedulable(ctx, c, f.Timeouts.NodeSchedulable))
		node1Name, node1KeyValueLabel, node2Name, node2KeyValueLabel = testSetupVolumePlacement(ctx, c, ns)
		ginkgo.DeferCleanup(func() {
			if len(node1KeyValueLabel) > 0 {
				e2enode.RemoveLabelOffNode(c, node1Name, NodeLabelKey)
			}
			if len(node2KeyValueLabel) > 0 {
				e2enode.RemoveLabelOffNode(c, node2Name, NodeLabelKey)
			}
		})
		nodeInfo = TestContext.NodeMapper.GetNodeInfo(node1Name)
		vsp = nodeInfo.VSphere
		ginkgo.By("creating vmdk")
		volumePath, err := vsp.CreateVolume(&VolumeOptions{}, nodeInfo.DataCenterRef)
		framework.ExpectNoError(err)
		volumePaths = append(volumePaths, volumePath)
		ginkgo.DeferCleanup(func() {
			for _, volumePath := range volumePaths {
				vsp.DeleteVolume(volumePath, nodeInfo.DataCenterRef)
			}
			volumePaths = nil
		})
	})

	/*
		Steps

		1. Create pod Spec with volume path of the vmdk and NodeSelector set to label assigned to node1.
		2. Create pod and wait for pod to become ready.
		3. Verify volume is attached to the node1.
		4. Create empty file on the volume to verify volume is writable.
		5. Verify newly created file and previously created files exist on the volume.
		6. Delete pod.
		7. Wait for volume to be detached from the node1.
		8. Repeat Step 1 to 7 and make sure back to back pod creation on same worker node with the same volume is working as expected.

	*/

	ginkgo.It("should create and delete pod with the same volume source on the same worker node", func(ctx context.Context) {
		var volumeFiles []string
		pod := createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)

		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		newEmptyFileName := fmt.Sprintf("/mnt/volume1/%v_1.txt", ns)
		volumeFiles = append(volumeFiles, newEmptyFileName)
		createAndVerifyFilesOnVolume(ns, pod.Name, []string{newEmptyFileName}, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node1Name, volumePaths)

		ginkgo.By(fmt.Sprintf("Creating pod on the same node: %v", node1Name))
		pod = createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)

		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		newEmptyFileName = fmt.Sprintf("/mnt/volume1/%v_2.txt", ns)
		volumeFiles = append(volumeFiles, newEmptyFileName)
		createAndVerifyFilesOnVolume(ns, pod.Name, []string{newEmptyFileName}, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node1Name, volumePaths)
	})

	/*
		Steps

		1. Create pod Spec with volume path of the vmdk1 and NodeSelector set to node1's label.
		2. Create pod and wait for POD to become ready.
		3. Verify volume is attached to the node1.
		4. Create empty file on the volume to verify volume is writable.
		5. Verify newly created file and previously created files exist on the volume.
		6. Delete pod.
		7. Wait for volume to be detached from the node1.
		8. Create pod Spec with volume path of the vmdk1 and NodeSelector set to node2's label.
		9. Create pod and wait for pod to become ready.
		10. Verify volume is attached to the node2.
		11. Create empty file on the volume to verify volume is writable.
		12. Verify newly created file and previously created files exist on the volume.
		13. Delete pod.
	*/

	ginkgo.It("should create and delete pod with the same volume source attach/detach to different worker nodes", func(ctx context.Context) {
		var volumeFiles []string
		pod := createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)
		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		newEmptyFileName := fmt.Sprintf("/mnt/volume1/%v_1.txt", ns)
		volumeFiles = append(volumeFiles, newEmptyFileName)
		createAndVerifyFilesOnVolume(ns, pod.Name, []string{newEmptyFileName}, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node1Name, volumePaths)

		ginkgo.By(fmt.Sprintf("Creating pod on the another node: %v", node2Name))
		pod = createPodWithVolumeAndNodeSelector(ctx, c, ns, node2Name, node2KeyValueLabel, volumePaths)

		newEmptyFileName = fmt.Sprintf("/mnt/volume1/%v_2.txt", ns)
		volumeFiles = append(volumeFiles, newEmptyFileName)
		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		createAndVerifyFilesOnVolume(ns, pod.Name, []string{newEmptyFileName}, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node2Name, volumePaths)
	})

	/*
		Test multiple volumes from same datastore within the same pod
		1. Create volumes - vmdk2
		2. Create pod Spec with volume path of vmdk1 (vmdk1 is created in test setup) and vmdk2.
		3. Create pod using spec created in step-2 and wait for pod to become ready.
		4. Verify both volumes are attached to the node on which pod are created. Write some data to make sure volume are accessible.
		5. Delete pod.
		6. Wait for vmdk1 and vmdk2 to be detached from node.
		7. Create pod using spec created in step-2 and wait for pod to become ready.
		8. Verify both volumes are attached to the node on which PODs are created. Verify volume contents are matching with the content written in step 4.
		9. Delete POD.
		10. Wait for vmdk1 and vmdk2 to be detached from node.
	*/

	ginkgo.It("should create and delete pod with multiple volumes from same datastore", func(ctx context.Context) {
		ginkgo.By("creating another vmdk")
		volumePath, err := vsp.CreateVolume(&VolumeOptions{}, nodeInfo.DataCenterRef)
		framework.ExpectNoError(err)
		volumePaths = append(volumePaths, volumePath)

		ginkgo.By(fmt.Sprintf("Creating pod on the node: %v with volume: %v and volume: %v", node1Name, volumePaths[0], volumePaths[1]))
		pod := createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)
		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		volumeFiles := []string{
			fmt.Sprintf("/mnt/volume1/%v_1.txt", ns),
			fmt.Sprintf("/mnt/volume2/%v_1.txt", ns),
		}
		createAndVerifyFilesOnVolume(ns, pod.Name, volumeFiles, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node1Name, volumePaths)
		ginkgo.By(fmt.Sprintf("Creating pod on the node: %v with volume :%v and volume: %v", node1Name, volumePaths[0], volumePaths[1]))
		pod = createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)
		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		newEmptyFilesNames := []string{
			fmt.Sprintf("/mnt/volume1/%v_2.txt", ns),
			fmt.Sprintf("/mnt/volume2/%v_2.txt", ns),
		}
		volumeFiles = append(volumeFiles, newEmptyFilesNames[0])
		volumeFiles = append(volumeFiles, newEmptyFilesNames[1])
		createAndVerifyFilesOnVolume(ns, pod.Name, newEmptyFilesNames, volumeFiles)
	})

	/*
		Test multiple volumes from different datastore within the same pod
		1. Create volumes - vmdk2 on non default shared datastore.
		2. Create pod Spec with volume path of vmdk1 (vmdk1 is created in test setup on default datastore) and vmdk2.
		3. Create pod using spec created in step-2 and wait for pod to become ready.
		4. Verify both volumes are attached to the node on which pod are created. Write some data to make sure volume are accessible.
		5. Delete pod.
		6. Wait for vmdk1 and vmdk2 to be detached from node.
		7. Create pod using spec created in step-2 and wait for pod to become ready.
		8. Verify both volumes are attached to the node on which PODs are created. Verify volume contents are matching with the content written in step 4.
		9. Delete POD.
		10. Wait for vmdk1 and vmdk2 to be detached from node.
	*/
	ginkgo.It("should create and delete pod with multiple volumes from different datastore", func(ctx context.Context) {
		ginkgo.By("creating another vmdk on non default shared datastore")
		var volumeOptions *VolumeOptions
		volumeOptions = new(VolumeOptions)
		volumeOptions.CapacityKB = 2097152
		volumeOptions.Name = "e2e-vmdk-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		volumeOptions.Datastore = GetAndExpectStringEnvVar(SecondSharedDatastore)
		volumePath, err := vsp.CreateVolume(volumeOptions, nodeInfo.DataCenterRef)

		framework.ExpectNoError(err)
		volumePaths = append(volumePaths, volumePath)

		ginkgo.By(fmt.Sprintf("Creating pod on the node: %v with volume :%v  and volume: %v", node1Name, volumePaths[0], volumePaths[1]))
		pod := createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)

		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		volumeFiles := []string{
			fmt.Sprintf("/mnt/volume1/%v_1.txt", ns),
			fmt.Sprintf("/mnt/volume2/%v_1.txt", ns),
		}
		createAndVerifyFilesOnVolume(ns, pod.Name, volumeFiles, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node1Name, volumePaths)

		ginkgo.By(fmt.Sprintf("Creating pod on the node: %v with volume :%v  and volume: %v", node1Name, volumePaths[0], volumePaths[1]))
		pod = createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, volumePaths)
		// Create empty files on the mounted volumes on the pod to verify volume is writable
		// Verify newly and previously created files present on the volume mounted on the pod
		newEmptyFileNames := []string{
			fmt.Sprintf("/mnt/volume1/%v_2.txt", ns),
			fmt.Sprintf("/mnt/volume2/%v_2.txt", ns),
		}
		volumeFiles = append(volumeFiles, newEmptyFileNames[0])
		volumeFiles = append(volumeFiles, newEmptyFileNames[1])
		createAndVerifyFilesOnVolume(ns, pod.Name, newEmptyFileNames, volumeFiles)
		deletePodAndWaitForVolumeToDetach(ctx, f, c, pod, node1Name, volumePaths)
	})

	/*
		Test Back-to-back pod creation/deletion with different volume sources on the same worker node
		    1. Create volumes - vmdk2
		    2. Create pod Spec - pod-SpecA with volume path of vmdk1 and NodeSelector set to label assigned to node1.
		    3. Create pod Spec - pod-SpecB with volume path of vmdk2 and NodeSelector set to label assigned to node1.
		    4. Create pod-A using pod-SpecA and wait for pod to become ready.
		    5. Create pod-B using pod-SpecB and wait for POD to become ready.
		    6. Verify volumes are attached to the node.
		    7. Create empty file on the volume to make sure volume is accessible. (Perform this step on pod-A and pod-B)
		    8. Verify file created in step 5 is present on the volume. (perform this step on pod-A and pod-B)
		    9. Delete pod-A and pod-B
		    10. Repeatedly (5 times) perform step 4 to 9 and verify associated volume's content is matching.
		    11. Wait for vmdk1 and vmdk2 to be detached from node.
	*/
	ginkgo.It("test back to back pod creation and deletion with different volume sources on the same worker node", func(ctx context.Context) {
		var (
			podA                *v1.Pod
			podB                *v1.Pod
			testvolumePathsPodA []string
			testvolumePathsPodB []string
			podAFiles           []string
			podBFiles           []string
		)

		defer func() {
			ginkgo.By("clean up undeleted pods")
			framework.ExpectNoError(e2epod.DeletePodWithWait(ctx, c, podA), "defer: Failed to delete pod ", podA.Name)
			framework.ExpectNoError(e2epod.DeletePodWithWait(ctx, c, podB), "defer: Failed to delete pod ", podB.Name)
			ginkgo.By(fmt.Sprintf("wait for volumes to be detached from the node: %v", node1Name))
			for _, volumePath := range volumePaths {
				framework.ExpectNoError(waitForVSphereDiskToDetach(ctx, volumePath, node1Name))
			}
		}()

		testvolumePathsPodA = append(testvolumePathsPodA, volumePaths[0])
		// Create another VMDK Volume
		ginkgo.By("creating another vmdk")
		volumePath, err := vsp.CreateVolume(&VolumeOptions{}, nodeInfo.DataCenterRef)
		framework.ExpectNoError(err)
		volumePaths = append(volumePaths, volumePath)
		testvolumePathsPodB = append(testvolumePathsPodA, volumePath)

		for index := 0; index < 5; index++ {
			ginkgo.By(fmt.Sprintf("Creating pod-A on the node: %v with volume: %v", node1Name, testvolumePathsPodA[0]))
			podA = createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, testvolumePathsPodA)

			ginkgo.By(fmt.Sprintf("Creating pod-B on the node: %v with volume: %v", node1Name, testvolumePathsPodB[0]))
			podB = createPodWithVolumeAndNodeSelector(ctx, c, ns, node1Name, node1KeyValueLabel, testvolumePathsPodB)

			podAFileName := fmt.Sprintf("/mnt/volume1/podA_%v_%v.txt", ns, index+1)
			podBFileName := fmt.Sprintf("/mnt/volume1/podB_%v_%v.txt", ns, index+1)
			podAFiles = append(podAFiles, podAFileName)
			podBFiles = append(podBFiles, podBFileName)

			// Create empty files on the mounted volumes on the pod to verify volume is writable
			ginkgo.By("Creating empty file on volume mounted on pod-A")
			e2eoutput.CreateEmptyFileOnPod(ns, podA.Name, podAFileName)

			ginkgo.By("Creating empty file volume mounted on pod-B")
			e2eoutput.CreateEmptyFileOnPod(ns, podB.Name, podBFileName)

			// Verify newly and previously created files present on the volume mounted on the pod
			ginkgo.By("Verify newly Created file and previously created files present on volume mounted on pod-A")
			verifyFilesExistOnVSphereVolume(ns, podA.Name, podAFiles...)
			ginkgo.By("Verify newly Created file and previously created files present on volume mounted on pod-B")
			verifyFilesExistOnVSphereVolume(ns, podB.Name, podBFiles...)

			ginkgo.By("Deleting pod-A")
			framework.ExpectNoError(e2epod.DeletePodWithWait(ctx, c, podA), "Failed to delete pod ", podA.Name)
			ginkgo.By("Deleting pod-B")
			framework.ExpectNoError(e2epod.DeletePodWithWait(ctx, c, podB), "Failed to delete pod ", podB.Name)
		}
	})
})

func testSetupVolumePlacement(ctx context.Context, client clientset.Interface, namespace string) (node1Name string, node1KeyValueLabel map[string]string, node2Name string, node2KeyValueLabel map[string]string) {
	nodes, err := e2enode.GetBoundedReadySchedulableNodes(ctx, client, 2)
	framework.ExpectNoError(err)
	if len(nodes.Items) < 2 {
		e2eskipper.Skipf("Requires at least %d nodes (not %d)", 2, len(nodes.Items))
	}
	node1Name = nodes.Items[0].Name
	node2Name = nodes.Items[1].Name
	node1LabelValue := "vsphere_e2e_" + string(uuid.NewUUID())
	node1KeyValueLabel = make(map[string]string)
	node1KeyValueLabel[NodeLabelKey] = node1LabelValue
	e2enode.AddOrUpdateLabelOnNode(client, node1Name, NodeLabelKey, node1LabelValue)

	node2LabelValue := "vsphere_e2e_" + string(uuid.NewUUID())
	node2KeyValueLabel = make(map[string]string)
	node2KeyValueLabel[NodeLabelKey] = node2LabelValue
	e2enode.AddOrUpdateLabelOnNode(client, node2Name, NodeLabelKey, node2LabelValue)
	return node1Name, node1KeyValueLabel, node2Name, node2KeyValueLabel
}

func createPodWithVolumeAndNodeSelector(ctx context.Context, client clientset.Interface, namespace string, nodeName string, nodeKeyValueLabel map[string]string, volumePaths []string) *v1.Pod {
	var pod *v1.Pod
	var err error
	ginkgo.By(fmt.Sprintf("Creating pod on the node: %v", nodeName))
	podspec := getVSpherePodSpecWithVolumePaths(volumePaths, nodeKeyValueLabel, nil)

	pod, err = client.CoreV1().Pods(namespace).Create(ctx, podspec, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	ginkgo.By("Waiting for pod to be ready")
	gomega.Expect(e2epod.WaitForPodNameRunningInNamespace(ctx, client, pod.Name, namespace)).To(gomega.Succeed())

	ginkgo.By(fmt.Sprintf("Verify volume is attached to the node:%v", nodeName))
	for _, volumePath := range volumePaths {
		isAttached, err := diskIsAttached(ctx, volumePath, nodeName)
		framework.ExpectNoError(err)
		if !isAttached {
			framework.Failf("Volume: %s is not attached to the node: %v", volumePath, nodeName)
		}
	}
	return pod
}

func createAndVerifyFilesOnVolume(namespace string, podname string, newEmptyfilesToCreate []string, filesToCheck []string) {
	// Create empty files on the mounted volumes on the pod to verify volume is writable
	ginkgo.By(fmt.Sprintf("Creating empty file on volume mounted on: %v", podname))
	createEmptyFilesOnVSphereVolume(namespace, podname, newEmptyfilesToCreate)

	// Verify newly and previously created files present on the volume mounted on the pod
	ginkgo.By(fmt.Sprintf("Verify newly Created file and previously created files present on volume mounted on: %v", podname))
	verifyFilesExistOnVSphereVolume(namespace, podname, filesToCheck...)
}

func deletePodAndWaitForVolumeToDetach(ctx context.Context, f *framework.Framework, c clientset.Interface, pod *v1.Pod, nodeName string, volumePaths []string) {
	ginkgo.By("Deleting pod")
	framework.ExpectNoError(e2epod.DeletePodWithWait(ctx, c, pod), "Failed to delete pod ", pod.Name)

	ginkgo.By("Waiting for volume to be detached from the node")
	for _, volumePath := range volumePaths {
		framework.ExpectNoError(waitForVSphereDiskToDetach(ctx, volumePath, nodeName))
	}
}
