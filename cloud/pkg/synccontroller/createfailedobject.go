package synccontroller

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	edgemgr "github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/manager"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
)

// Compare the objects in K8s with the objects that have been persisted to the edge,
// If the objects fail to persisted to the edge for the first time, it will be recreated here.
// Just focus on the pod, service, endpoint, because if the pod is passed down, and the configmap and secret
// cannot be found then it will be queried to the cloud.
func (sctl *SyncController) manageCreateFailedObject() {
	sctl.manageCreateFailedCoreObject()
	// TODO: checking for devices
	// sctl.manageCreateFailedDevice()
}

func (sctl *SyncController) manageCreateFailedCoreObject() {
	allPods, err := sctl.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Filed to list all the pods: %v", err)
		return
	}

	set := labels.Set{edgemgr.NodeRoleKey: edgemgr.NodeRoleValue}
	selector := labels.SelectorFromSet(set)
	allEdgeNodes, err := sctl.nodeLister.List(selector)
	if err != nil {
		klog.Errorf("Filed to list all the edge nodes: %v", err)
		return
	}

	for _, pod := range allPods {
		if !isFromEdgeNode(allEdgeNodes, pod.Spec.NodeName) {
			continue
		}
		// Check whether the pod is successfully persisted to edge
		_, err := sctl.objectSyncLister.ObjectSyncs(pod.Namespace).Get(BuildObjectSyncName(pod.Spec.NodeName, string(pod.UID)))
		if err != nil && apierrors.IsNotFound(err) {
			msg := buildEdgeControllerMessage(pod.Spec.NodeName, pod.Namespace, model.ResourceTypePod, pod.Name, model.InsertOperation, pod)
			beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
		}

		// TODO: add send check for service and endpoint
		/*
			services, err := sctl.serviceLister.GetPodServices(pod)
			if err != nil {
				klog.Errorf("Filed to list all the services for pod %s: %v", pod.Name, err)
			}

			for _, svc := range services {
				// Check whether the service related to pod is successfully persisted to edge
				_, err := sctl.objectSyncLister.ObjectSyncs(svc.Namespace).Get(buildObjectSyncName(pod.Spec.NodeName, string(svc.UID)))
				if err != nil && apierrors.IsNotFound(err) {
					msg := buildEdgeControllerMessage(pod.Spec.NodeName, svc.Namespace, commonconst.ResourceTypeService, svc.Name, model.InsertOperation, svc)
					beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
				}

				selector := labels.SelectorFromSet(svc.Spec.Selector)
				endpoints, err := sctl.endpointLister.Endpoints(svc.Namespace).List(selector)
				if err != nil {
					klog.Errorf("Filed to list all the endpoints for svc %s: %v", svc.Name, err)
				}

				for _, endpoint := range endpoints {
					// Check whether the endpoint related to service is successfully persisted to edge
					_, err := sctl.objectSyncLister.ObjectSyncs(endpoint.Namespace).Get(buildObjectSyncName(pod.Spec.NodeName, string(endpoint.UID)))
					if err != nil && apierrors.IsNotFound(err) {
						msg := buildEdgeControllerMessage(pod.Spec.NodeName, endpoint.Namespace, commonconst.ResourceTypeEndpoints, endpoint.Name, model.InsertOperation, endpoint)
						beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
					}
				}
			}
		*/
	}
}

func (sctl *SyncController) manageCreateFailedDevice() {
	allDevices, err := sctl.deviceLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Filed to list all the devices: %v", err)
		return
	}

	for _, device := range allDevices {
		// Check whether the device is successfully persisted to edge
		// TODO: refactor the nodeselector of the device
		nodeName := device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		_, err := sctl.objectSyncLister.ObjectSyncs(device.Namespace).Get(BuildObjectSyncName(nodeName, string(device.UID)))
		if err != nil && apierrors.IsNotFound(err) {
			msg := buildEdgeControllerMessage(nodeName, device.Namespace, commonconst.ResourceTypeService, device.Name, model.InsertOperation, device)
			beehiveContext.Send(commonconst.DefaultContextSendModuleName, *msg)
		}
	}
}
