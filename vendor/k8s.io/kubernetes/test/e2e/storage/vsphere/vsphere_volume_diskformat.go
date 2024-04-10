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
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	admissionapi "k8s.io/pod-security-admission/api"
)

/*
	Test to verify diskformat specified in storage-class is being honored while volume creation.
	Valid and supported options are eagerzeroedthick, zeroedthick and thin

	Steps
	1. Create StorageClass with diskformat set to valid type
	2. Create PVC which uses the StorageClass created in step 1.
	3. Wait for PV to be provisioned.
	4. Wait for PVC's status to become Bound
	5. Create pod using PVC on specific node.
	6. Wait for Disk to be attached to the node.
	7. Get node VM's devices and find PV's Volume Disk.
	8. Get Backing Info of the Volume Disk and obtain EagerlyScrub and ThinProvisioned
	9. Based on the value of EagerlyScrub and ThinProvisioned, verify diskformat is correct.
	10. Delete pod and Wait for Volume Disk to be detached from the Node.
	11. Delete PVC, PV and Storage Class
*/

var _ = utils.SIGDescribe("Volume Disk Format [Feature:vsphere]", func() {
	f := framework.NewDefaultFramework("volume-disk-format")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	const (
		NodeLabelKey = "vsphere_e2e_label_volume_diskformat"
	)
	var (
		client            clientset.Interface
		namespace         string
		nodeName          string
		nodeKeyValueLabel map[string]string
		nodeLabelValue    string
	)
	ginkgo.BeforeEach(func(ctx context.Context) {
		e2eskipper.SkipUnlessProviderIs("vsphere")
		Bootstrap(f)
		client = f.ClientSet
		namespace = f.Namespace.Name
		nodeName = GetReadySchedulableRandomNodeInfo(ctx, client).Name
		nodeLabelValue = "vsphere_e2e_" + string(uuid.NewUUID())
		nodeKeyValueLabel = map[string]string{NodeLabelKey: nodeLabelValue}
		e2enode.AddOrUpdateLabelOnNode(client, nodeName, NodeLabelKey, nodeLabelValue)
		ginkgo.DeferCleanup(e2enode.RemoveLabelOffNode, client, nodeName, NodeLabelKey)
	})

	ginkgo.It("verify disk format type - eagerzeroedthick is honored for dynamically provisioned pv using storageclass", func(ctx context.Context) {
		ginkgo.By("Invoking Test for diskformat: eagerzeroedthick")
		invokeTest(ctx, f, client, namespace, nodeName, nodeKeyValueLabel, "eagerzeroedthick")
	})
	ginkgo.It("verify disk format type - zeroedthick is honored for dynamically provisioned pv using storageclass", func(ctx context.Context) {
		ginkgo.By("Invoking Test for diskformat: zeroedthick")
		invokeTest(ctx, f, client, namespace, nodeName, nodeKeyValueLabel, "zeroedthick")
	})
	ginkgo.It("verify disk format type - thin is honored for dynamically provisioned pv using storageclass", func(ctx context.Context) {
		ginkgo.By("Invoking Test for diskformat: thin")
		invokeTest(ctx, f, client, namespace, nodeName, nodeKeyValueLabel, "thin")
	})
})

func invokeTest(ctx context.Context, f *framework.Framework, client clientset.Interface, namespace string, nodeName string, nodeKeyValueLabel map[string]string, diskFormat string) {

	framework.Logf("Invoking Test for DiskFomat: %s", diskFormat)
	scParameters := make(map[string]string)
	scParameters["diskformat"] = diskFormat

	ginkgo.By("Creating Storage Class With DiskFormat")
	storageClassSpec := getVSphereStorageClassSpec("thinsc", scParameters, nil, "")
	storageclass, err := client.StorageV1().StorageClasses().Create(ctx, storageClassSpec, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	ginkgo.DeferCleanup(framework.IgnoreNotFound(client.StorageV1().StorageClasses().Delete), storageclass.Name, metav1.DeleteOptions{})

	ginkgo.By("Creating PVC using the Storage Class")
	pvclaimSpec := getVSphereClaimSpecWithStorageClass(namespace, "2Gi", storageclass)
	pvclaim, err := client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvclaimSpec, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	ginkgo.DeferCleanup(framework.IgnoreNotFound(client.CoreV1().PersistentVolumeClaims(namespace).Delete), pvclaimSpec.Name, metav1.DeleteOptions{})

	ginkgo.By("Waiting for claim to be in bound phase")
	err = e2epv.WaitForPersistentVolumeClaimPhase(ctx, v1.ClaimBound, client, pvclaim.Namespace, pvclaim.Name, framework.Poll, f.Timeouts.ClaimProvision)
	framework.ExpectNoError(err)

	// Get new copy of the claim
	pvclaim, err = client.CoreV1().PersistentVolumeClaims(pvclaim.Namespace).Get(ctx, pvclaim.Name, metav1.GetOptions{})
	framework.ExpectNoError(err)

	// Get the bound PV
	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvclaim.Spec.VolumeName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	/*
		PV is required to be attached to the Node. so that using govmomi API we can grab Disk's Backing Info
		to check EagerlyScrub and ThinProvisioned property
	*/
	ginkgo.By("Creating pod to attach PV to the node")
	// Create pod to attach Volume to Node
	podSpec := getVSpherePodSpecWithClaim(pvclaim.Name, nodeKeyValueLabel, "while true ; do sleep 2 ; done")
	pod, err := client.CoreV1().Pods(namespace).Create(ctx, podSpec, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	ginkgo.By("Waiting for pod to be running")
	gomega.Expect(e2epod.WaitForPodNameRunningInNamespace(ctx, client, pod.Name, namespace)).To(gomega.Succeed())

	isAttached, err := diskIsAttached(ctx, pv.Spec.VsphereVolume.VolumePath, nodeName)
	if !isAttached {
		framework.Failf("Volume: %s is not attached to the node: %v", pv.Spec.VsphereVolume.VolumePath, nodeName)
	}
	framework.ExpectNoError(err)

	ginkgo.By("Verify Disk Format")
	framework.ExpectEqual(verifyDiskFormat(ctx, client, nodeName, pv.Spec.VsphereVolume.VolumePath, diskFormat), true, "DiskFormat Verification Failed")

	var volumePaths []string
	volumePaths = append(volumePaths, pv.Spec.VsphereVolume.VolumePath)

	ginkgo.By("Delete pod and wait for volume to be detached from node")
	deletePodAndWaitForVolumeToDetach(ctx, f, client, pod, nodeName, volumePaths)

}

func verifyDiskFormat(ctx context.Context, client clientset.Interface, nodeName string, pvVolumePath string, diskFormat string) bool {
	ginkgo.By("Verifying disk format")
	eagerlyScrub := false
	thinProvisioned := false
	diskFound := false
	pvvmdkfileName := filepath.Base(pvVolumePath) + filepath.Ext(pvVolumePath)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	nodeInfo := TestContext.NodeMapper.GetNodeInfo(nodeName)
	vm := object.NewVirtualMachine(nodeInfo.VSphere.Client.Client, nodeInfo.VirtualMachineRef)
	vmDevices, err := vm.Device(ctx)
	framework.ExpectNoError(err)

	disks := vmDevices.SelectByType((*types.VirtualDisk)(nil))

	for _, disk := range disks {
		backing := disk.GetVirtualDevice().Backing.(*types.VirtualDiskFlatVer2BackingInfo)
		backingFileName := filepath.Base(backing.FileName) + filepath.Ext(backing.FileName)
		if backingFileName == pvvmdkfileName {
			diskFound = true
			if backing.EagerlyScrub != nil {
				eagerlyScrub = *backing.EagerlyScrub
			}
			if backing.ThinProvisioned != nil {
				thinProvisioned = *backing.ThinProvisioned
			}
			break
		}
	}

	if !diskFound {
		framework.Failf("Failed to find disk: %s", pvVolumePath)
	}
	isDiskFormatCorrect := false
	if diskFormat == "eagerzeroedthick" {
		if eagerlyScrub == true && thinProvisioned == false {
			isDiskFormatCorrect = true
		}
	} else if diskFormat == "zeroedthick" {
		if eagerlyScrub == false && thinProvisioned == false {
			isDiskFormatCorrect = true
		}
	} else if diskFormat == "thin" {
		if eagerlyScrub == false && thinProvisioned == true {
			isDiskFormatCorrect = true
		}
	}
	return isDiskFormatCorrect
}
