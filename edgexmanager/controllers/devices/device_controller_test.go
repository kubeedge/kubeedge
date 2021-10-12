package devices

import (
	"bytes"
	"context"
	"fmt"
	devicesv1alpha3 "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = FDescribe("Device controller", func() {

	//Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		DeviceNamespace   = "default"
		DeviceName        = "random-boolean-device"
		DeviceProfileName = "random-boolean-device"
		DeviceServiceName = "device-virtual"
		DeviceVisitorName = "random-boolean-device-visitor"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	deviceProfile := &devicesv1alpha3.DeviceModel{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "devices.kubeedge.io/v1alpha3",
			Kind:       "DeviceModel",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: DeviceProfileName,
		},
		Spec: devicesv1alpha3.DeviceModelSpec{
			DeviceProperties: []devicesv1alpha3.DeviceProperties{
				{
					Name:        "EnableRandomization_Bool",
					Description: "used to decide whether to re-generate a random value",
					ProfileProperty: devicesv1alpha3.ProfileProperty{
						Mutable:      true,
						Type:         "Bool",
						ReadWrite:    "W",
						DefaultValue: "true",
					},
				},
				{
					Name:        "Bool",
					Description: "Generate random boolean value",
					ProfileProperty: devicesv1alpha3.ProfileProperty{
						Mutable:      true,
						Type:         "Bool",
						ReadWrite:    "RW",
						DefaultValue: "true",
					},
				},
			},
		},
	}
	//s := &unstructured.Unstructured{}
	//s.SetUnstructuredContent(map[string]interface{}{"t": "t"})
	//deviceVisitor := &devicesv1alpha3.DeviceAccess{
	//	TypeMeta: metav1.TypeMeta{
	//		APIVersion: "devices.kubeedge.io/v1alpha3",
	//		Kind:       "DeviceAccess",
	//	},
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: DeviceVisitorName,
	//	},
	//	Spec: devicesv1alpha3.DeviceAccessSpec{
	//
	//	},
	//}
	device := &devicesv1alpha3.Device{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "devices.kubeedge.io/v1alpha3",
			Kind:       "Device",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeviceName,
		},
		Spec: devicesv1alpha3.DeviceSpec{
			ModelRef:      DeviceProfileName,
			DeviceAccessRef: &corev1.ObjectReference{
				APIVersion: "devices.kubeedge.io/v1alpha3",
				Kind:       "DeviceAccess",
				Name:       DeviceVisitorName,
			},
			Protocol: devicesv1alpha3.DeviceProtocol{
				Name:    "other",
				Type:    "300",
				Address: "device-virtual-bool-01",
				//Args: nil,
			},
			//DeviceCommands: []devicesv1alpha3.DeviceCommand{
			//	{Name: "Bool", Value: "false"},
			//},
		},
	}

	Context("When create Device CR", func() {
		It("Should create Device on Edgex.", func() {
			By("By creating a new Device")
			ctx := context.Background()
			//namespace := corev1.Namespace{}
			//namespace.Name= NAMESPACE
			//Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())
			//deviceService := &devicesv1alpha3.DeviceService{}
			//Expect(createByYaml(ctx, k8sClient, "../../config/samples/devices_v1alpha3_deviceservice.yaml", deviceService)).Should(Succeed())
			//edgexManager := &devicesv1alpha3.EdgeXManager{}
			//Expect(createByYaml(ctx, k8sClient, "../../config/samples/devices_v1alpha3_edgexmanager.yaml", edgexManager)).Should(Succeed())

			time.Sleep(time.Second * 3)

			Expect(k8sClient.Create(ctx, deviceProfile)).Should(Succeed())
			deviceVisitor := &devicesv1alpha3.DeviceAccess{}
			Expect(createByYaml(ctx, k8sClient, "../../config/samples/devices_v1alpha3_devicevisitor.yaml", deviceVisitor)).Should(Succeed())
			Expect(k8sClient.Create(ctx, device)).Should(Succeed())

			//Expect(k8sClient.Patch(ctx, device, )).Should(Succeed())
			defer func() {
				By("By deleting a new Device")
				//Expect(k8sClient.Delete(ctx, device)).Should(Succeed())
				//Expect(k8sClient.Delete(ctx, deviceProfile)).Should(Succeed())
				//Expect(k8sClient.Delete(ctx, deviceVisitor)).Should(Succeed())
			}()

			time.Sleep(time.Second * 2)
			//reportedDevice := &devicesv1alpha3.Device{}
			//Eventually(func() bool {
			//	err := k8sClient.Get(ctx, types.NamespacedName{Name: DeviceName, Namespace: DeviceNamespace}, reportedDevice)
			//	if err != nil {
			//		return false
			//	}
			//	return true
			//}, timeout, interval).Should(BeTrue())
			//j, _ := json.Marshal(reportedDevice)
			//By(string(j))

		})
	})
})


func createByYaml(ctx context.Context, client client.Client, filePath string, obj client.Object) error {

	filebytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(filebytes), 100)
	//var rawObj runtime.RawExtension
	//if err = decoder.Decode(&rawObj); err != nil {
	//	return err
	//}
	//
	//obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	//unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	//if err != nil {
	//	//log.Fatal(err)
	//	fmt.Printf("%v\n", err)
	//	return err
	//}
	//
	//unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	//deviceService := &v1alpha3.DeviceService{}
	if err = decoder.Decode(obj); err != nil {
		return err
	}

	fmt.Printf("deviceService: %+v", obj)

	return client.Create(ctx, obj)



}