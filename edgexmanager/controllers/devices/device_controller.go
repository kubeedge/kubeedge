/*
Copyright 2021.

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

package devices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kubeedge/kubeedge/edgexmanager/pkg/util"
	"github.com/kubeedge/kubeedge/edgexmanager/pkg/util/patch"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	devicesv1alpha3 "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha3"
	"github.com/kubeedge/kubeedge/edgexmanager/pkg/edgex"
)

// DeviceReconciler reconciles a Device object
type DeviceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const deviceFinalizerName = "device.device.kubeedge.io"
const NAMESPACE = "kubeedge"

//+kubebuilder:rbac:groups=devices.kubeedge.io,resources=devices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devices.kubeedge.io,resources=devices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devices.kubeedge.io,resources=devices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Device object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	device := &devicesv1alpha3.Device{}
	if err := r.Get(ctx, req.NamespacedName, device); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "device is not available yet")
		return ctrl.Result{}, err
	}
	r.Log.Info("device controler reconciling")
	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(device, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		fmt.Printf("reconcile %+v \n", device)
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, device, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()
	// your logic here
	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(device, deviceFinalizerName) {
		controllerutil.AddFinalizer(device, deviceFinalizerName)
	}

	if !device.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, device)
	}

	return r.reconcileNormal(ctx, device)
}

func createOrUpdateDevice(ctx context.Context, client client.Client, device *devicesv1alpha3.Device, model *devicesv1alpha3.DeviceModel, visitor *unstructured.Unstructured) (string, error) {
	typedVisitor := &devicesv1alpha3.DeviceAccess{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(visitor.UnstructuredContent(), typedVisitor); err != nil {
		return "", err
	}
	metadataUrl, commandUrl, err := getServiceAddress(ctx, client, device)
	if err != nil {
		return "", err
	}

	edgexClient := edgex.NewEdgeXClient(metadataUrl, commandUrl)
	profileID, err := edgexClient.GetDeviceProfileByName(ctx, model.Name)
	if err != nil {
		return "", err
	} else if profileID == "" {
		deviceProfileDto, err := edgex.ConvertToDeviceProfileDTO(model, typedVisitor)
		if err != nil {
			return "", err
		}
		_, err = edgexClient.AddDeviceProfile(ctx, deviceProfileDto)
		if err != nil {
			fmt.Println("add device model fail")
			return "", err
		}
	} else {
		deviceProfileDto, err := edgex.ConvertToDeviceProfileDTO(model, typedVisitor)
		if err != nil {
			return "", err
		}
		err = edgexClient.UpdateDeviceProfile(ctx, deviceProfileDto)
		if err != nil {
			fmt.Println("update device model fail")
			return "", err
		}
	}

	deviceID, err := edgexClient.GetDeviceByName(ctx, generateDeviceName(device.Namespace, device.Name))
	if err != nil {
		return "", err
	} else if deviceID == "" {
		deviceDto, err := edgex.ConvertToDeviceDTO(device)
		if err != nil {
			return "", err
		}
		deviceID, err = edgexClient.AddDevice(ctx, deviceDto)
		if err != nil {
			return "", err
		}
	} else {
		deviceDto, err := edgex.ConvertToUpdateDeviceDTO(device)
		if err != nil {
			return "", err
		}
		err = edgexClient.UpdateDevice(ctx, deviceDto)
		if err != nil {
			return "", err
		}
	}
	return deviceID, nil
}

func deleteDevice(ctx context.Context, client client.Client, device *devicesv1alpha3.Device) error {
	metadataUrl, commandUrl, err := getServiceAddress(ctx, client, device)
	if err != nil {
		return err
	}
	edgexClient := edgex.NewEdgeXClient(metadataUrl, commandUrl)
	if err := edgexClient.DeleteDeviceByName(ctx, generateDeviceName(device.Namespace, device.Name)); err != nil {
		fmt.Println("delete device fail")
		return err
	}
	fmt.Println("deleteing profile ...")
	if err := edgexClient.DeleteDeviceProfileByName(ctx, device.Spec.ModelRef); err != nil {
		fmt.Println("delete device profile fail")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapFunc := func(a client.Object) []reconcile.Request {
		profile := a.(*devicesv1alpha3.DeviceModel)
		r := make([]reconcile.Request, len(profile.Status.DeviceRefs))
		for i, n := range profile.Status.DeviceRefs {
			r[i].Name = n.Name
			r[i].Namespace = n.Namespace
		}
		return r
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&devicesv1alpha3.Device{}).
		Watches(&source.Kind{Type: &devicesv1alpha3.DeviceModel{}}, handler.EnqueueRequestsFromMapFunc(mapFunc)).
		//WithEventFilter(predicate.AnnotationChangedPredicate{}).
		Complete(r)
}

func (r *DeviceReconciler) reconcileDelete(ctx context.Context, device *devicesv1alpha3.Device) (ctrl.Result, error) {
	deleteDevice(ctx, r.Client, device)
	controllerutil.RemoveFinalizer(device, deviceFinalizerName)
	return ctrl.Result{}, nil
}

func (r *DeviceReconciler) reconcileNormal(ctx context.Context, device *devicesv1alpha3.Device) (ctrl.Result, error) {

	model := &devicesv1alpha3.DeviceModel{}
	if err := r.Get(ctx, types.NamespacedName{Name: device.Spec.ModelRef}, model); err != nil {
		r.Log.Error(err, "device model does not exist")
		return ctrl.Result{}, err
	}

	if err := r.reconcileModel(ctx, device, model); err != nil {
		return ctrl.Result{}, err
	}
	device.OwnerReferences = util.EnsureOwnerRef(device.OwnerReferences, v1.OwnerReference{
		APIVersion: devicesv1alpha3.GroupVersion.String(),
		Kind:       "DeviceModel",
		Name:       model.Name,
		UID:        model.UID,
	})

	var visitor *unstructured.Unstructured
	if device.Spec.DeviceAccessRef != nil {
		visitor = &unstructured.Unstructured{}
		visitor.SetGroupVersionKind(schema.FromAPIVersionAndKind(device.Spec.DeviceAccessRef.APIVersion, device.Spec.DeviceAccessRef.Kind))
		if err := r.Get(ctx, types.NamespacedName{
			Name: device.Spec.DeviceAccessRef.Name,
		}, visitor); err != nil {
			r.Log.Error(err, "device access does not exist")
			return ctrl.Result{}, err
		}
	}

	deviceID, err := createOrUpdateDevice(ctx, r.Client, device, model, visitor)
	if err != nil {
		r.Log.Error(err, "create device failed")
		return ctrl.Result{}, err
	}
	if !v1.HasAnnotation(device.ObjectMeta, "device.kubeedge.io/deviceID") {
		v1.SetMetaDataAnnotation(&device.ObjectMeta, "device.kubeedge.io/deviceID", deviceID)
	}

	device.Status.Ready = true

	if err := r.reconcileDeviceProperties(ctx, device, model); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DeviceReconciler) reconcileModel(ctx context.Context, device *devicesv1alpha3.Device, profile *devicesv1alpha3.DeviceModel) error {
	for _, deviceRef := range profile.Status.DeviceRefs {
		if deviceRef.Name == device.Name && deviceRef.Namespace == device.Namespace {
			return nil
		}
	}
	profile.Status.DeviceRefs = append(profile.Status.DeviceRefs, devicesv1alpha3.NamespaceName{Namespace: device.Namespace, Name: device.Name})
	return r.Status().Update(ctx, profile)
}

func (r *DeviceReconciler) reconcileDeviceProperties(ctx context.Context, device *devicesv1alpha3.Device, model *devicesv1alpha3.DeviceModel) error {
	fmt.Println("update device status")

	if len(device.Status.DeviceCommands) == 0 {
		device.Status.DeviceCommands = make([]devicesv1alpha3.DeviceCommandStatus, 0, len(device.Spec.DeviceCommands))
	}
	writeIdx := make([]int, 0, len(device.Spec.DeviceCommands))
	readIdx := make([]int, 0, len(device.Spec.DeviceCommands))
	for i, prop := range device.Spec.DeviceCommands {
		res := getDevcieResourceByName(model.Spec.DeviceProperties, prop.Name)
		if len(device.Status.DeviceCommands) < i+1 {
			device.Status.DeviceCommands = append(device.Status.DeviceCommands, devicesv1alpha3.DeviceCommandStatus{
				DeviceCommand: devicesv1alpha3.DeviceCommand{Name: prop.Name},
				ReadWrite:     res.ReadWrite})
		} else if device.Status.DeviceCommands[i].Name != prop.Name {
			device.Status.DeviceCommands = append(device.Status.DeviceCommands[:i], device.Status.DeviceCommands[i+1:]...)
		} else {
			device.Status.DeviceCommands[i].ReadWrite = res.ReadWrite
			if device.Status.DeviceCommands[i].Value != prop.Value {
				writeIdx = append(writeIdx, i)
			} else if res.ReadWrite == "R" || res.ReadWrite == "RW" {
				readIdx = append(readIdx, i)
			}
		}
		p := device.Status.DeviceCommands[i]
		switch res.ReadWrite {
		case "R":
			readIdx = append(readIdx, i)
		case "W":
			if p.Value != prop.Value {
				writeIdx = append(writeIdx, i)
			}
		case "RW":
			if p.Value != prop.Value {
				writeIdx = append(writeIdx, i)
			}
			readIdx = append(readIdx, i)
		}
	}
	if len(device.Spec.DeviceCommands) < len(device.Status.DeviceCommands) {
		device.Status.DeviceCommands = device.Status.DeviceCommands[:len(device.Spec.DeviceCommands)]
	}

	setDeviceProperties(ctx, r.Client, device, writeIdx)

	getDeviceProperties(ctx, r.Client, device, readIdx)

	return nil
}

func getDeviceProperties(ctx context.Context, client client.Client, device *devicesv1alpha3.Device, indexes []int) {
	metadataUrl, commandUrl, err := getServiceAddress(ctx, client, device)
	if err != nil {
		return
	}
	edgexClient := edgex.NewEdgeXClient(metadataUrl, commandUrl)
	deviceName := generateDeviceName(device.Namespace, device.Name)
	for _, idx := range indexes {
		prop := &device.Status.DeviceCommands[idx]
		v, err := edgexClient.GetDeviceResourceByName(ctx, deviceName, prop.Name)
		if err != nil {
			fmt.Println(err)
		}
		prop.Value = v
	}
}

func setDeviceProperties(ctx context.Context, client client.Client, device *devicesv1alpha3.Device, indexes []int) {

	metadataUrl, commandUrl, err := getServiceAddress(ctx, client, device)
	if err != nil {
		return
	}
	edgexClient := edgex.NewEdgeXClient(metadataUrl, commandUrl)
	deviceName := generateDeviceName(device.Namespace, device.Name)
	for _, idx := range indexes {
		prop := device.Spec.DeviceCommands[idx]
		err := edgexClient.SetDeviceResourceByName(ctx, deviceName, prop.Name, map[string]string{prop.Name: prop.Value})
		if err != nil {
			fmt.Println(err)
		}

	}
}

func getDevcieResourceByName(resources []devicesv1alpha3.DeviceProperties, name string) *devicesv1alpha3.DeviceProperties {
	for _, r := range resources {
		if r.Name == name {
			return &r
		}
	}
	return nil
}

func getServiceAddress(ctx context.Context, client client.Client, device *devicesv1alpha3.Device) (metadataAddr string,
	commandAddr string, err error) {
	metadataSvc, exist := device.Spec.DeviceService["metadata"]
	if !exist {
		err = errors.New("not exist metadata")
		return
	}
	var metadataName string
	if err = json.Unmarshal(metadataSvc.Raw, &metadataName); err != nil {
		return
	}
	svc := &corev1.Service{}
	if err = client.Get(ctx, types.NamespacedName{Name: metadataName, Namespace: NAMESPACE}, svc); err != nil {
		err = errors.New("not exist metadata service")
		return
	}
	metadataAddr = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace,
		svc.Spec.Ports[0].Port)

	commandSvc, exist := device.Spec.DeviceService["command"]
	if !exist {
		err = errors.New("not exist command")
		return
	}
	var commandName string
	if err = json.Unmarshal(commandSvc.Raw, &commandName); err != nil {
		return
	}
	svc = &corev1.Service{}
	if err = client.Get(ctx, types.NamespacedName{Name: commandName, Namespace: NAMESPACE}, svc); err != nil {
		err = errors.New("not exist command service")
		return
	}
	commandAddr = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace,
		svc.Spec.Ports[0].Port)

	return
}

type ResourceLabelChangedPredicate struct {
	predicate.Funcs
}

func generateDeviceName(namespace string, name string) string {
	return namespace + "-" + name
}
