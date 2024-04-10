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
	"time"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/kubernetes/test/e2e/storage/utils"
	admissionapi "k8s.io/pod-security-admission/api"
)

const (
	diskSizeSCName = "disksizesc"
)

/*
	Test to verify disk size specified in PVC is being rounded up correctly.

	Steps
	1. Create StorageClass.
	2. Create PVC with invalid disk size which uses the StorageClass created in step 1.
	3. Verify the provisioned PV size is correct.
*/

var _ = utils.SIGDescribe("Volume Disk Size [Feature:vsphere]", func() {
	f := framework.NewDefaultFramework("volume-disksize")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	var (
		client       clientset.Interface
		namespace    string
		scParameters map[string]string
		datastore    string
	)
	ginkgo.BeforeEach(func() {
		e2eskipper.SkipUnlessProviderIs("vsphere")
		Bootstrap(f)
		client = f.ClientSet
		namespace = f.Namespace.Name
		scParameters = make(map[string]string)
		datastore = GetAndExpectStringEnvVar(StorageClassDatastoreName)
	})

	ginkgo.It("verify dynamically provisioned pv has size rounded up correctly", func(ctx context.Context) {
		ginkgo.By("Invoking Test disk size")
		scParameters[Datastore] = datastore
		scParameters[DiskFormat] = ThinDisk
		diskSize := "1"
		expectedDiskSize := "1Mi"

		ginkgo.By("Creating Storage Class")
		storageclass, err := client.StorageV1().StorageClasses().Create(ctx, getVSphereStorageClassSpec(diskSizeSCName, scParameters, nil, ""), metav1.CreateOptions{})
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(framework.IgnoreNotFound(client.StorageV1().StorageClasses().Delete), storageclass.Name, metav1.DeleteOptions{})

		ginkgo.By("Creating PVC using the Storage Class")
		pvclaim, err := e2epv.CreatePVC(ctx, client, namespace, getVSphereClaimSpecWithStorageClass(namespace, diskSize, storageclass))
		framework.ExpectNoError(err)
		ginkgo.DeferCleanup(e2epv.DeletePersistentVolumeClaim, client, pvclaim.Name, namespace)

		ginkgo.By("Waiting for claim to be in bound phase")
		err = e2epv.WaitForPersistentVolumeClaimPhase(ctx, v1.ClaimBound, client, pvclaim.Namespace, pvclaim.Name, framework.Poll, 2*time.Minute)
		framework.ExpectNoError(err)

		ginkgo.By("Getting new copy of PVC")
		pvclaim, err = client.CoreV1().PersistentVolumeClaims(pvclaim.Namespace).Get(ctx, pvclaim.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Getting PV created")
		pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvclaim.Spec.VolumeName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Verifying if provisioned PV has the correct size")
		expectedCapacity := resource.MustParse(expectedDiskSize)
		pvCapacity := pv.Spec.Capacity[v1.ResourceName(v1.ResourceStorage)]
		framework.ExpectEqual(pvCapacity.Value(), expectedCapacity.Value())
	})
})
