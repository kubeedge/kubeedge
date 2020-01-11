package synccontroller

import (
	"k8s.io/klog"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/reliablesyncs/v1alpha1"
	edgectrconst "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	edgectrmessagelayer "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
)

func (sctl *SyncController) managePod(sync *v1alpha1.ObjectSync) {
	pod, err := sctl.podLister.Pods(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := strings.Split(sync.Name, "/")[0]
	//
	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypePod, sync.Spec.ObjectName, model.DeleteOperation, nil)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}

	if pod.ResourceVersion > sync.Status.ObjectResourceVersion {
		// trigger the update event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypePod, sync.Spec.ObjectName, model.UpdateOperation, pod)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}

func (sctl *SyncController) manageConfigMap(sync *v1alpha1.ObjectSync) {
	configmap, err := sctl.configMapLister.ConfigMaps(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := strings.Split(sync.Name, "/")[0]
	//
	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypeConfigmap, sync.Spec.ObjectName, model.DeleteOperation, nil)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}

	if configmap.ResourceVersion > sync.Status.ObjectResourceVersion {
		// trigger the update event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypeConfigmap, sync.Spec.ObjectName, model.UpdateOperation, configmap)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}
func (sctl *SyncController) manageSecret(sync *v1alpha1.ObjectSync) {
	secret, err := sctl.secretLister.Secrets(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := strings.Split(sync.Name, "/")[0]
	//
	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypeSecret, sync.Spec.ObjectName, model.DeleteOperation, nil)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}

	if secret.ResourceVersion > sync.Status.ObjectResourceVersion {
		// trigger the update event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypeSecret, sync.Spec.ObjectName, model.UpdateOperation, secret)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}

func (sctl *SyncController) manageService(sync *v1alpha1.ObjectSync) {
	service, err := sctl.serviceLister.Services(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := strings.Split(sync.Name, "/")[0]
	//
	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, commonconst.ResourceTypeService, sync.Spec.ObjectName, model.DeleteOperation, nil)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}

	if service.ResourceVersion > sync.Status.ObjectResourceVersion {
		// trigger the update event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, commonconst.ResourceTypeService, sync.Spec.ObjectName, model.UpdateOperation, service)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}

func (sctl *SyncController) manageEndpoint(sync *v1alpha1.ObjectSync) {
	endpoint, err := sctl.endpointLister.Endpoints(sync.Namespace).Get(sync.Spec.ObjectName)

	nodeName := strings.Split(sync.Name, "/")[0]
	//
	if err != nil && apierrors.IsNotFound(err) {
		//trigger the delete event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypePod, sync.Spec.ObjectName, model.DeleteOperation, nil)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}

	if endpoint.ResourceVersion > sync.Status.ObjectResourceVersion {
		// trigger the update event
		msg := buildEdgeControllerMessage(nodeName, sync.Namespace, model.ResourceTypePod, sync.Spec.ObjectName, model.UpdateOperation, endpoint)
		beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
	}
}

// todo: add events for devices
func (sctl *SyncController) manageDevice(sync *v1alpha1.ObjectSync) {
	//pod, err := sctl.deviceLister.Devices(sync.Namespace).Get(sync.Spec.ObjectName)

	//
	//if err != nil && apierrors.IsNotFound(err) {
	//trigger the delete event
	//}

	//if pod.ResourceVersion > sync.Status.ObjectResourceVersion {
	// trigger the update event
	//}
}

func buildEdgeControllerMessage(nodeName, namespace, resourceType, resourceName, operationType string, obj interface{}) *model.Message {
	msg := model.NewMessage("")
	resource, err := edgectrmessagelayer.BuildResource(nodeName, namespace, resourceType, resourceName)
	if err != nil {
		klog.Warningf("build message resource failed with error: %s", err)
		return nil
	}
	msg.BuildRouter(edgectrconst.EdgeControllerModuleName, edgectrconst.GroupResource, resource, operationType)
	msg.Content = obj
	return msg
}
