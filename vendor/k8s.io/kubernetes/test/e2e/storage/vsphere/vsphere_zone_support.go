/*
Copyright 2019 The Kubernetes Authors.

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
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	volumeevents "k8s.io/kubernetes/pkg/controller/volume/events"
	"k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	admissionapi "k8s.io/pod-security-admission/api"
)

/*
   Test to verify multi-zone support for dynamic volume provisioning in kubernetes.
   The test environment is illustrated below:

   datacenter-1
        --->cluster-vsan-1 (zone-a)                              ____________________    _________________
                --->host-1         : master                     |                    |  |                 |
                --->host-2         : node1  ___________________ |                    |  |                 |
                --->host-3 (zone-c): node2 |                   ||    vsanDatastore   |  |                 |
                                           |  localDatastore   ||                    |  |                 |
                                           |___________________||____________________|  |   sharedVmfs-0  |
        --->cluster-vsan-2 (zone-b)                              ____________________   |                 |
                --->host-4         : node3                      |                    |  |                 |
                --->host-5         : node4                      |  vsanDatastore (1) |  |                 |
                --->host-6                                      |                    |  |                 |
                                                                |____________________|  |_________________|
        --->cluster-3 (zone-c)              ___________________
                --->host-7         : node5 |                   |
                                           | localDatastore (1)|
                                           |___________________|
   datacenter-2
        --->cluster-1 (zone-d)             ___________________
                --->host-8        : node6 |                   |
                                          |  localDatastore   |
                                          |___________________|

	Testbed description :
	1. cluster-vsan-1 is tagged with zone-a. So, vsanDatastore inherits zone-a since all the hosts under zone-a have vsanDatastore mounted on them.
	2. cluster-vsan-2 is tagged with zone-b. So, vsanDatastore (1) inherits zone-b since all the hosts under zone-b have vsanDatastore (1) mounted on them.
	3. sharedVmfs-0 inherits both zone-a and zone-b since all the hosts in both zone-a and zone-b have this datastore mounted on them.
	4. cluster-3 is tagged with zone-c. cluster-3 only contains host-7.
	5. host-3 under cluster-vsan-1 is tagged with zone-c.
	6. Since there are no shared datastores between host-7 under cluster-3 and host-3 under cluster-vsan-1, no datastores in the environment inherit zone-c.
	7. host-8 under datacenter-2 and cluster-1 is tagged with zone-d. So, localDatastore attached to host-8 inherits zone-d.
	8. The six worker nodes are distributed among the hosts as shown in the above illustration.
	9. Two storage policies are created on VC. One is a VSAN storage policy named as compatpolicy with hostFailuresToTolerate capability set to 1.

	Testsuite description :
	1. Tests to verify that zone labels are set correctly on a dynamically created PV.
	2. Tests to verify dynamic pv creation fails if availability zones are not specified or if there are no shared datastores under the specified zones.
	3. Tests to verify dynamic pv creation using availability zones works in combination with other storage class parameters such as storage policy,
	   datastore and VSAN capabilities.
	4. Tests to verify dynamic pv creation using availability zones fails in combination with other storage class parameters such as storage policy,
	   datastore and VSAN capabilities specifications when any of the former mentioned parameters are incompatible with the rest.
	5. Tests to verify dynamic pv creation using availability zones work across different datacenters in the same VC.
*/

var _ = utils.SIGDescribe("Zone Support [Feature:vsphere]", func() {
	f := framework.NewDefaultFramework("zone-support")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	var (
		scParameters    map[string]string
		zones           []string
		vsanDatastore1  string
		vsanDatastore2  string
		localDatastore  string
		compatPolicy    string
		nonCompatPolicy string
		zoneA           string
		zoneB           string
		zoneC           string
		zoneD           string
		invalidZone     string
	)
	ginkgo.BeforeEach(func(ctx context.Context) {
		e2eskipper.SkipUnlessProviderIs("vsphere")
		Bootstrap(f)
		e2eskipper.SkipUnlessMultizone(ctx, f.ClientSet)
		vsanDatastore1 = GetAndExpectStringEnvVar(VCPZoneVsanDatastore1)
		vsanDatastore2 = GetAndExpectStringEnvVar(VCPZoneVsanDatastore2)
		localDatastore = GetAndExpectStringEnvVar(VCPZoneLocalDatastore)
		compatPolicy = GetAndExpectStringEnvVar(VCPZoneCompatPolicyName)
		nonCompatPolicy = GetAndExpectStringEnvVar(VCPZoneNonCompatPolicyName)
		zoneA = GetAndExpectStringEnvVar(VCPZoneA)
		zoneB = GetAndExpectStringEnvVar(VCPZoneB)
		zoneC = GetAndExpectStringEnvVar(VCPZoneC)
		zoneD = GetAndExpectStringEnvVar(VCPZoneD)
		invalidZone = GetAndExpectStringEnvVar(VCPInvalidZone)
		scParameters = make(map[string]string)
		zones = make([]string, 0)
		_, err := e2enode.GetRandomReadySchedulableNode(ctx, f.ClientSet)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Verify dynamically created pv with allowed zones specified in storage class, shows the right zone information on its labels", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with the following zones : %s", zoneA))
		zones = append(zones, zoneA)
		verifyPVZoneLabels(ctx, f, nil, zones)
	})

	ginkgo.It("Verify dynamically created pv with multiple zones specified in the storage class, shows both the zones on its labels", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with the following zones : %s, %s", zoneA, zoneB))
		zones = append(zones, zoneA)
		zones = append(zones, zoneB)
		verifyPVZoneLabels(ctx, f, nil, zones)
	})

	ginkgo.It("Verify PVC creation with invalid zone specified in storage class fails", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with unknown zone : %s", invalidZone))
		zones = append(zones, invalidZone)
		err := verifyPVCCreationFails(ctx, f, nil, zones, "")
		framework.ExpectError(err)
		errorMsg := "Failed to find a shared datastore matching zone [" + invalidZone + "]"
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on allowed zones specified in storage class ", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zones :%s", zoneA))
		zones = append(zones, zoneA)
		verifyPVCAndPodCreationSucceeds(ctx, f, nil, zones, "")
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on multiple zones specified in storage class ", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zones :%s, %s", zoneA, zoneB))
		zones = append(zones, zoneA)
		zones = append(zones, zoneB)
		verifyPVCAndPodCreationSucceeds(ctx, f, nil, zones, "")
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on the allowed zones and datastore specified in storage class", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s and datastore :%s", zoneA, vsanDatastore1))
		scParameters[Datastore] = vsanDatastore1
		zones = append(zones, zoneA)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify PVC creation with incompatible datastore and zone combination specified in storage class fails", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s and datastore :%s", zoneC, vsanDatastore1))
		scParameters[Datastore] = vsanDatastore1
		zones = append(zones, zoneC)
		err := verifyPVCCreationFails(ctx, f, scParameters, zones, "")
		errorMsg := "No matching datastores found in the kubernetes cluster for zone " + zoneC
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on the allowed zones and storage policy specified in storage class", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s and storage policy :%s", zoneA, compatPolicy))
		scParameters[SpbmStoragePolicy] = compatPolicy
		zones = append(zones, zoneA)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify a pod is created on a non-Workspace zone and attached to a dynamically created PV, based on the allowed zones and storage policy specified in storage class", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s and storage policy :%s", zoneB, compatPolicy))
		scParameters[SpbmStoragePolicy] = compatPolicy
		zones = append(zones, zoneB)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify PVC creation with incompatible storagePolicy and zone combination specified in storage class fails", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s and storage policy :%s", zoneA, nonCompatPolicy))
		scParameters[SpbmStoragePolicy] = nonCompatPolicy
		zones = append(zones, zoneA)
		err := verifyPVCCreationFails(ctx, f, scParameters, zones, "")
		errorMsg := "No compatible datastores found that satisfy the storage policy requirements"
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on the allowed zones, datastore and storage policy specified in storage class", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s datastore :%s and storagePolicy :%s", zoneA, vsanDatastore1, compatPolicy))
		scParameters[SpbmStoragePolicy] = compatPolicy
		scParameters[Datastore] = vsanDatastore1
		zones = append(zones, zoneA)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify PVC creation with incompatible storage policy along with compatible zone and datastore combination specified in storage class fails", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s datastore :%s and storagePolicy :%s", zoneA, vsanDatastore1, nonCompatPolicy))
		scParameters[SpbmStoragePolicy] = nonCompatPolicy
		scParameters[Datastore] = vsanDatastore1
		zones = append(zones, zoneA)
		err := verifyPVCCreationFails(ctx, f, scParameters, zones, "")
		errorMsg := "User specified datastore is not compatible with the storagePolicy: \\\"" + nonCompatPolicy + "\\\"."
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation with incompatible zone along with compatible storagePolicy and datastore combination specified in storage class fails", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s datastore :%s and storagePolicy :%s", zoneC, vsanDatastore2, compatPolicy))
		scParameters[SpbmStoragePolicy] = compatPolicy
		scParameters[Datastore] = vsanDatastore2
		zones = append(zones, zoneC)
		err := verifyPVCCreationFails(ctx, f, scParameters, zones, "")
		errorMsg := "No matching datastores found in the kubernetes cluster for zone " + zoneC
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation fails if no zones are specified in the storage class (No shared datastores exist among all the nodes)", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with no zones"))
		err := verifyPVCCreationFails(ctx, f, nil, nil, "")
		errorMsg := "No shared datastores found in the Kubernetes cluster"
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation fails if only datastore is specified in the storage class (No shared datastores exist among all the nodes)", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with datastore :%s", vsanDatastore1))
		scParameters[Datastore] = vsanDatastore1
		err := verifyPVCCreationFails(ctx, f, scParameters, nil, "")
		errorMsg := "No shared datastores found in the Kubernetes cluster"
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation fails if only storage policy is specified in the storage class (No shared datastores exist among all the nodes)", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with storage policy :%s", compatPolicy))
		scParameters[SpbmStoragePolicy] = compatPolicy
		err := verifyPVCCreationFails(ctx, f, scParameters, nil, "")
		errorMsg := "No shared datastores found in the Kubernetes cluster"
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation with compatible policy and datastore without any zones specified in the storage class fails (No shared datastores exist among all the nodes)", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with storage policy :%s and datastore :%s", compatPolicy, vsanDatastore1))
		scParameters[SpbmStoragePolicy] = compatPolicy
		scParameters[Datastore] = vsanDatastore1
		err := verifyPVCCreationFails(ctx, f, scParameters, nil, "")
		errorMsg := "No shared datastores found in the Kubernetes cluster"
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation fails if the availability zone specified in the storage class have no shared datastores under it.", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s", zoneC))
		zones = append(zones, zoneC)
		err := verifyPVCCreationFails(ctx, f, nil, zones, "")
		errorMsg := "No matching datastores found in the kubernetes cluster for zone " + zoneC
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on multiple zones specified in the storage class. (No shared datastores exist among both zones)", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with the following zones :%s and %s", zoneA, zoneC))
		zones = append(zones, zoneA)
		zones = append(zones, zoneC)
		err := verifyPVCCreationFails(ctx, f, nil, zones, "")
		errorMsg := "No matching datastores found in the kubernetes cluster for zone " + zoneC
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify PVC creation with an invalid VSAN capability along with a compatible zone combination specified in storage class fails", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with %s :%s and zone :%s", PolicyHostFailuresToTolerate, HostFailuresToTolerateCapabilityInvalidVal, zoneA))
		scParameters[PolicyHostFailuresToTolerate] = HostFailuresToTolerateCapabilityInvalidVal
		zones = append(zones, zoneA)
		err := verifyPVCCreationFails(ctx, f, scParameters, zones, "")
		errorMsg := "Invalid value for " + PolicyHostFailuresToTolerate + "."
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on a VSAN capability, datastore and compatible zone specified in storage class", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with %s :%s, %s :%s, datastore :%s and zone :%s", PolicyObjectSpaceReservation, ObjectSpaceReservationCapabilityVal, PolicyIopsLimit, IopsLimitCapabilityVal, vsanDatastore1, zoneA))
		scParameters[PolicyObjectSpaceReservation] = ObjectSpaceReservationCapabilityVal
		scParameters[PolicyIopsLimit] = IopsLimitCapabilityVal
		scParameters[Datastore] = vsanDatastore1
		zones = append(zones, zoneA)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on the allowed zones specified in storage class when the datastore under the zone is present in another datacenter", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s", zoneD))
		zones = append(zones, zoneD)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV, based on the allowed zones and datastore specified in storage class when there are multiple datastores with the same name under different zones across datacenters", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with zone :%s and datastore name :%s", zoneD, localDatastore))
		scParameters[Datastore] = localDatastore
		zones = append(zones, zoneD)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, "")
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV with storage policy specified in storage class in waitForFirstConsumer binding mode", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with waitForFirstConsumer mode and storage policy :%s", compatPolicy))
		scParameters[SpbmStoragePolicy] = compatPolicy
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, nil, storagev1.VolumeBindingWaitForFirstConsumer)
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV with storage policy specified in storage class in waitForFirstConsumer binding mode with allowedTopologies", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with waitForFirstConsumer mode, storage policy :%s and zone :%s", compatPolicy, zoneA))
		scParameters[SpbmStoragePolicy] = compatPolicy
		zones = append(zones, zoneA)
		verifyPVCAndPodCreationSucceeds(ctx, f, scParameters, zones, storagev1.VolumeBindingWaitForFirstConsumer)
	})

	ginkgo.It("Verify a pod is created and attached to a dynamically created PV with storage policy specified in storage class in waitForFirstConsumer binding mode with multiple allowedTopologies", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with waitForFirstConsumer mode and zones : %s, %s", zoneA, zoneB))
		zones = append(zones, zoneA)
		zones = append(zones, zoneB)
		verifyPVCAndPodCreationSucceeds(ctx, f, nil, zones, storagev1.VolumeBindingWaitForFirstConsumer)
	})

	ginkgo.It("Verify a PVC creation fails when multiple zones are specified in the storage class without shared datastores among the zones in waitForFirstConsumer binding mode", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with waitForFirstConsumer mode and following zones :%s and %s", zoneA, zoneC))
		zones = append(zones, zoneA)
		zones = append(zones, zoneC)
		err := verifyPodAndPvcCreationFailureOnWaitForFirstConsumerMode(ctx, f, nil, zones)
		framework.ExpectError(err)
		errorMsg := "No matching datastores found in the kubernetes cluster for zone " + zoneC
		if !strings.Contains(err.Error(), errorMsg) {
			framework.ExpectNoError(err, errorMsg)
		}
	})

	ginkgo.It("Verify a pod fails to get scheduled when conflicting volume topology (allowedTopologies) and pod scheduling constraints(nodeSelector) are specified", func(ctx context.Context) {
		ginkgo.By(fmt.Sprintf("Creating storage class with waitForFirstConsumerMode, storage policy :%s and zone :%s", compatPolicy, zoneA))
		scParameters[SpbmStoragePolicy] = compatPolicy
		// allowedTopologies set as zoneA
		zones = append(zones, zoneA)
		nodeSelectorMap := map[string]string{
			// nodeSelector set as zoneB
			v1.LabelTopologyZone: zoneB,
		}
		verifyPodSchedulingFails(ctx, f, nodeSelectorMap, scParameters, zones, storagev1.VolumeBindingWaitForFirstConsumer)
	})
})

func verifyPVCAndPodCreationSucceeds(ctx context.Context, f *framework.Framework, scParameters map[string]string, zones []string, volumeBindingMode storagev1.VolumeBindingMode) {
	client := f.ClientSet
	namespace := f.Namespace.Name
	timeouts := f.Timeouts
	storageclass, err := client.StorageV1().StorageClasses().Create(ctx, getVSphereStorageClassSpec("zone-sc", scParameters, zones, volumeBindingMode), metav1.CreateOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("Failed to create storage class with err: %v", err))
	ginkgo.DeferCleanup(client.StorageV1().StorageClasses().Delete, storageclass.Name, metav1.DeleteOptions{})

	ginkgo.By("Creating PVC using the Storage Class")
	pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, getVSphereClaimSpecWithStorageClass(namespace, "2Gi", storageclass))
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

	var pvclaims []*v1.PersistentVolumeClaim
	pvclaims = append(pvclaims, pvclaim)

	var persistentvolumes []*v1.PersistentVolume
	// If WaitForFirstConsumer mode, verify pvc binding status after pod creation. For immediate mode, do now.
	if volumeBindingMode != storagev1.VolumeBindingWaitForFirstConsumer {
		persistentvolumes = waitForPVClaimBoundPhase(ctx, client, pvclaims, timeouts.ClaimProvision)
	}

	ginkgo.By("Creating pod to attach PV to the node")
	pod, err := e2epod.CreatePod(ctx, client, namespace, nil, pvclaims, f.NamespacePodSecurityLevel, "")
	framework.ExpectNoError(err)

	if volumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
		persistentvolumes = waitForPVClaimBoundPhase(ctx, client, pvclaims, timeouts.ClaimProvision)
	}

	if zones != nil {
		ginkgo.By("Verify persistent volume was created on the right zone")
		verifyVolumeCreationOnRightZone(ctx, persistentvolumes, pod.Spec.NodeName, zones)
	}

	ginkgo.By("Verify the volume is accessible and available in the pod")
	verifyVSphereVolumesAccessible(ctx, client, pod, persistentvolumes)

	ginkgo.By("Deleting pod")
	framework.ExpectNoError(e2epod.DeletePodWithWait(ctx, client, pod))

	ginkgo.By("Waiting for volumes to be detached from the node")
	framework.ExpectNoError(waitForVSphereDiskToDetach(ctx, persistentvolumes[0].Spec.VsphereVolume.VolumePath, pod.Spec.NodeName))
}

func verifyPodAndPvcCreationFailureOnWaitForFirstConsumerMode(ctx context.Context, f *framework.Framework, scParameters map[string]string, zones []string) error {
	client := f.ClientSet
	namespace := f.Namespace.Name
	storageclass, err := client.StorageV1().StorageClasses().Create(ctx, getVSphereStorageClassSpec("zone-sc", scParameters, zones, storagev1.VolumeBindingWaitForFirstConsumer), metav1.CreateOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("Failed to create storage class with err: %v", err))
	ginkgo.DeferCleanup(client.StorageV1().StorageClasses().Delete, storageclass.Name, metav1.DeleteOptions{})

	ginkgo.By("Creating PVC using the Storage Class")
	pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, getVSphereClaimSpecWithStorageClass(namespace, "2Gi", storageclass))
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

	var pvclaims []*v1.PersistentVolumeClaim
	pvclaims = append(pvclaims, pvclaim)

	ginkgo.By("Creating a pod")
	pod := e2epod.MakePod(namespace, nil, pvclaims, f.NamespacePodSecurityLevel, "")
	pod, err = client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epod.DeletePodWithWait, client, pod)

	ginkgo.By("Waiting for claim to be in bound phase")
	err = e2epv.WaitForPersistentVolumeClaimPhase(ctx, v1.ClaimBound, client, pvclaim.Namespace, pvclaim.Name, framework.Poll, 2*time.Minute)
	framework.ExpectError(err)

	eventList, err := client.CoreV1().Events(pvclaim.Namespace).List(ctx, metav1.ListOptions{})
	framework.ExpectNoError(err)

	// Look for PVC ProvisioningFailed event and return the message.
	for _, event := range eventList.Items {
		if event.Source.Component == "persistentvolume-controller" && event.Reason == volumeevents.ProvisioningFailed {
			return fmt.Errorf("Failure message: %s", event.Message)
		}
	}
	return nil
}

func waitForPVClaimBoundPhase(ctx context.Context, client clientset.Interface, pvclaims []*v1.PersistentVolumeClaim, timeout time.Duration) []*v1.PersistentVolume {
	ginkgo.By("Waiting for claim to be in bound phase")
	persistentvolumes, err := e2epv.WaitForPVClaimBoundPhase(ctx, client, pvclaims, timeout)
	framework.ExpectNoError(err)
	return persistentvolumes
}

func verifyPodSchedulingFails(ctx context.Context, f *framework.Framework, nodeSelector map[string]string, scParameters map[string]string, zones []string, volumeBindingMode storagev1.VolumeBindingMode) {
	client := f.ClientSet
	namespace := f.Namespace.Name
	storageclass, err := client.StorageV1().StorageClasses().Create(ctx, getVSphereStorageClassSpec("zone-sc", scParameters, zones, volumeBindingMode), metav1.CreateOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("Failed to create storage class with err: %v", err))
	ginkgo.DeferCleanup(client.StorageV1().StorageClasses().Delete, storageclass.Name, metav1.DeleteOptions{})

	ginkgo.By("Creating PVC using the Storage Class")
	pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, getVSphereClaimSpecWithStorageClass(namespace, "2Gi", storageclass))
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

	var pvclaims []*v1.PersistentVolumeClaim
	pvclaims = append(pvclaims, pvclaim)

	ginkgo.By("Creating a pod")
	pod, err := e2epod.CreateUnschedulablePod(ctx, client, namespace, nodeSelector, pvclaims, f.NamespacePodSecurityLevel, "")
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epod.DeletePodWithWait, client, pod)
}

func verifyPVCCreationFails(ctx context.Context, f *framework.Framework, scParameters map[string]string, zones []string, volumeBindingMode storagev1.VolumeBindingMode) error {
	client := f.ClientSet
	namespace := f.Namespace.Name
	storageclass, err := client.StorageV1().StorageClasses().Create(ctx, getVSphereStorageClassSpec("zone-sc", scParameters, zones, volumeBindingMode), metav1.CreateOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("Failed to create storage class with err: %v", err))
	ginkgo.DeferCleanup(client.StorageV1().StorageClasses().Delete, storageclass.Name, metav1.DeleteOptions{})

	ginkgo.By("Creating PVC using the Storage Class")
	pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, getVSphereClaimSpecWithStorageClass(namespace, "2Gi", storageclass))
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

	ginkgo.By("Waiting for claim to be in bound phase")
	err = e2epv.WaitForPersistentVolumeClaimPhase(ctx, v1.ClaimBound, client, pvclaim.Namespace, pvclaim.Name, framework.Poll, 2*time.Minute)
	framework.ExpectError(err)

	eventList, err := client.CoreV1().Events(pvclaim.Namespace).List(ctx, metav1.ListOptions{})
	framework.ExpectNoError(err)

	framework.Logf("Failure message : %+q", eventList.Items[0].Message)
	return fmt.Errorf("Failure message: %+q", eventList.Items[0].Message)
}

func verifyPVZoneLabels(ctx context.Context, f *framework.Framework, scParameters map[string]string, zones []string) {
	client := f.ClientSet
	namespace := f.Namespace.Name
	storageclass, err := client.StorageV1().StorageClasses().Create(ctx, getVSphereStorageClassSpec("zone-sc", nil, zones, ""), metav1.CreateOptions{})
	framework.ExpectNoError(err, fmt.Sprintf("Failed to create storage class with err: %v", err))
	ginkgo.DeferCleanup(client.StorageV1().StorageClasses().Delete, storageclass.Name, metav1.DeleteOptions{})

	ginkgo.By("Creating PVC using the storage class")
	pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, getVSphereClaimSpecWithStorageClass(namespace, "2Gi", storageclass))
	framework.ExpectNoError(err)
	ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

	var pvclaims []*v1.PersistentVolumeClaim
	pvclaims = append(pvclaims, pvclaim)
	ginkgo.By("Waiting for claim to be in bound phase")
	persistentvolumes, err := e2epv.WaitForPVClaimBoundPhase(ctx, client, pvclaims, f.Timeouts.ClaimProvision)
	framework.ExpectNoError(err)

	ginkgo.By("Verify zone information is present in the volume labels")
	for _, pv := range persistentvolumes {
		// Multiple zones are separated with "__"
		pvZoneLabels := strings.Split(pv.ObjectMeta.Labels[v1.LabelTopologyZone], "__")
		for _, zone := range zones {
			gomega.Expect(pvZoneLabels).Should(gomega.ContainElement(zone), "Incorrect or missing zone labels in pv.")
		}
	}
}
