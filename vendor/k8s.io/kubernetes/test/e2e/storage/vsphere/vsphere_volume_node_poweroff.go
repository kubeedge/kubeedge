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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/vmware/govmomi/object"
	vimtypes "github.com/vmware/govmomi/vim25/types"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2edeployment "k8s.io/kubernetes/test/e2e/framework/deployment"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	admissionapi "k8s.io/pod-security-admission/api"
)

/*
Test to verify volume status after node power off:
1. Verify the pod got provisioned on a different node with volume attached to it
2. Verify the volume is detached from the powered off node
*/
var _ = utils.SIGDescribe("Node Poweroff [Feature:vsphere] [Slow] [Disruptive]", func() {
	f := framework.NewDefaultFramework("node-poweroff")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	var (
		client    clientset.Interface
		namespace string
	)

	ginkgo.BeforeEach(func(ctx context.Context) {
		e2eskipper.SkipUnlessProviderIs("vsphere")
		Bootstrap(f)
		client = f.ClientSet
		namespace = f.Namespace.Name
		framework.ExpectNoError(e2enode.WaitForAllNodesSchedulable(ctx, client, f.Timeouts.NodeSchedulable))
		nodeList, err := e2enode.GetReadySchedulableNodes(ctx, f.ClientSet)
		framework.ExpectNoError(err)
		if len(nodeList.Items) < 2 {
			framework.Failf("At least 2 nodes are required for this test, got instead: %v", len(nodeList.Items))
		}
	})

	/*
		Steps:
		1. Create a StorageClass
		2. Create a PVC with the StorageClass
		3. Create a Deployment with 1 replica, using the PVC
		4. Verify the pod got provisioned on a node
		5. Verify the volume is attached to the node
		6. Power off the node where pod got provisioned
		7. Verify the pod got provisioned on a different node
		8. Verify the volume is attached to the new node
		9. Verify the volume is detached from the old node
		10. Delete the Deployment and wait for the volume to be detached
		11. Delete the PVC
		12. Delete the StorageClass
	*/
	ginkgo.It("verify volume status after node power off", func(ctx context.Context) {
		ginkgo.By("Creating a Storage Class")
		storageClassSpec := getVSphereStorageClassSpec("test-sc", nil, nil, "")
		storageclass, err := client.StorageV1().StorageClasses().Create(ctx, storageClassSpec, metav1.CreateOptions{})
		framework.ExpectNoError(err, fmt.Sprintf("Failed to create storage class with err: %v", err))
		ginkgo.DeferCleanup(framework.IgnoreNotFound(client.StorageV1().StorageClasses().Delete), storageclass.Name, metav1.DeleteOptions{})

		ginkgo.By("Creating PVC using the Storage Class")
		pvclaimSpec := getVSphereClaimSpecWithStorageClass(namespace, "1Gi", storageclass)
		pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, pvclaimSpec)
		framework.ExpectNoError(err, fmt.Sprintf("Failed to create PVC with err: %v", err))
		ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

		ginkgo.By("Waiting for PVC to be in bound phase")
		pvclaims := []*v1.PersistentVolumeClaim{pvclaim}
		pvs, err := e2epv.WaitForPVClaimBoundPhase(ctx, client, pvclaims, f.Timeouts.ClaimProvision)
		framework.ExpectNoError(err, fmt.Sprintf("Failed to wait until PVC phase set to bound: %v", err))
		volumePath := pvs[0].Spec.VsphereVolume.VolumePath

		ginkgo.By("Creating a Deployment")
		deployment, err := e2edeployment.CreateDeployment(ctx, client, int32(1), map[string]string{"test": "app"}, nil, namespace, pvclaims, admissionapi.LevelRestricted, "")
		framework.ExpectNoError(err, fmt.Sprintf("Failed to create Deployment with err: %v", err))
		ginkgo.DeferCleanup(framework.IgnoreNotFound(client.AppsV1().Deployments(namespace).Delete), deployment.Name, metav1.DeleteOptions{})

		ginkgo.By("Get pod from the deployment")
		podList, err := e2edeployment.GetPodsForDeployment(ctx, client, deployment)
		framework.ExpectNoError(err, fmt.Sprintf("Failed to get pod from the deployment with err: %v", err))
		gomega.Expect(podList.Items).NotTo(gomega.BeEmpty())
		pod := podList.Items[0]
		node1 := pod.Spec.NodeName

		ginkgo.By(fmt.Sprintf("Verify disk is attached to the node: %v", node1))
		isAttached, err := diskIsAttached(ctx, volumePath, node1)
		framework.ExpectNoError(err)
		if !isAttached {
			framework.Failf("Volume: %s is not attached to the node: %v", volumePath, node1)
		}

		ginkgo.By(fmt.Sprintf("Power off the node: %v", node1))

		nodeInfo := TestContext.NodeMapper.GetNodeInfo(node1)
		vm := object.NewVirtualMachine(nodeInfo.VSphere.Client.Client, nodeInfo.VirtualMachineRef)
		_, err = vm.PowerOff(ctx)
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(vm.PowerOn)

		err = vm.WaitForPowerState(ctx, vimtypes.VirtualMachinePowerStatePoweredOff)
		framework.ExpectNoError(err, "Unable to power off the node")

		// Waiting for the pod to be failed over to a different node
		node2, err := waitForPodToFailover(ctx, client, deployment, node1)
		framework.ExpectNoError(err, "Pod did not fail over to a different node")

		ginkgo.By(fmt.Sprintf("Waiting for disk to be attached to the new node: %v", node2))
		err = waitForVSphereDiskToAttach(ctx, volumePath, node2)
		framework.ExpectNoError(err, "Disk is not attached to the node")

		ginkgo.By(fmt.Sprintf("Waiting for disk to be detached from the previous node: %v", node1))
		err = waitForVSphereDiskToDetach(ctx, volumePath, node1)
		framework.ExpectNoError(err, "Disk is not detached from the node")

		ginkgo.By(fmt.Sprintf("Power on the previous node: %v", node1))
		vm.PowerOn(ctx)
		err = vm.WaitForPowerState(ctx, vimtypes.VirtualMachinePowerStatePoweredOn)
		framework.ExpectNoError(err, "Unable to power on the node")
	})
})

// Wait until the pod failed over to a different node, or time out after 3 minutes
func waitForPodToFailover(ctx context.Context, client clientset.Interface, deployment *appsv1.Deployment, oldNode string) (string, error) {
	var (
		timeout  = 3 * time.Minute
		pollTime = 10 * time.Second
	)

	waitErr := wait.PollWithContext(ctx, pollTime, timeout, func(ctx context.Context) (bool, error) {
		currentNode, err := getNodeForDeployment(ctx, client, deployment)
		if err != nil {
			return true, err
		}

		if currentNode != oldNode {
			framework.Logf("The pod has been failed over from %q to %q", oldNode, currentNode)
			return true, nil
		}

		framework.Logf("Waiting for pod to be failed over from %q", oldNode)
		return false, nil
	})

	if waitErr != nil {
		if waitErr == wait.ErrWaitTimeout {
			return "", fmt.Errorf("pod has not failed over after %v: %v", timeout, waitErr)
		}
		return "", fmt.Errorf("pod did not fail over from %q: %v", oldNode, waitErr)
	}

	return getNodeForDeployment(ctx, client, deployment)
}

// getNodeForDeployment returns node name for the Deployment
func getNodeForDeployment(ctx context.Context, client clientset.Interface, deployment *appsv1.Deployment) (string, error) {
	podList, err := e2edeployment.GetPodsForDeployment(ctx, client, deployment)
	if err != nil {
		return "", err
	}
	return podList.Items[0].Spec.NodeName, nil
}
