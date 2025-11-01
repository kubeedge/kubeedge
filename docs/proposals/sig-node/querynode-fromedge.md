---
title: Query Node Resources from the Edge
authors:
- "@luomengY"
  approvers:
  creation-date: 2025-10-19
  last-updated: 2025-10-19
  status: implementable
---

# The retrieval of Node resources is performed directly at the edge, replacing the method of fetching from the cloud.


## Motivation

- In the underlying layer of edged, multiple modules need to retrieve Node resources from the API server, such as: nodestatusmanager, volumemanager, evictionmanager, and podmanager. Calls from these modules are typically made in a periodic polling manner. In edgecore, queries for Node resources from edged are handled through the metaclient, which forwards the requests via the cloud-edge channel through cloudcore to the API server. The Node resources obtained from the API server are then returned to the edge node's edgecore through the same cloud-edge channel.
- edged periodically updates patch node and patch pod status data to the API server via the cloud-edge channel. However, after a successful update, the newly updated Node and Pod resources are sent back to edged through the cloud-edge channel.

In summary, in weak network environments and scenarios with a large number of edge nodes, the transmission of Node resources from cloudcore to each edge node consumes significant network bandwidth. Therefore, it is necessary to reduce the volume of messages sent from cloudcore to edgecore.

### Goals

- When edged needs to obtain Node resources from the API server, instead of forwarding the request through the cloud-edge channel to the API server, it retrieves the data directly from the local SQLite database at the edge.
- After edged patches the status information of a node or pod to the API server, there is no need for the updated node or pod to be sent back to edged.
- To ensure that the node and pod data retrieved from the SQLite database remains consistent with the state in the API server, when edged receives confirmation that the patch to the API server was successful, the metamanager merges the patch data into the respective node or pod object and persists the updated state into SQLite.

## Background and challenges

- Edge nodes retrieve Node resources locally instead of fetching them from the cloud.
- After the cloudcore patches the pod and node status to the API server, it does not return the updated pod and node objects. Instead, the patch data is directly merged into the pod and node objects at the edge and then saved to SQLite


## Design Details

- Remove the condition `resType == model.ResourceTypeNode ||` from the `requireRemoteQuery` function in the file `edge/pkg/metamanager/process.go`, as follows:
```go
// is resource type require remote query
func requireRemoteQuery(resType string) bool {
	return resType == model.ResourceTypeConfigmap ||
		resType == model.ResourceTypeSecret ||
		resType == constants.ResourceTypePersistentVolume ||
		resType == constants.ResourceTypePersistentVolumeClaim ||
		resType == constants.ResourceTypeVolumeAttachment ||
		resType == model.ResourceTypeServiceAccountToken ||
		resType == model.ResourceTypeLease ||
		resType == model.ResourceTypeCSR ||
		resType == model.ResourceTypeK8sCA
}
```
- Add functionality to persist the Node to the SQLite database within the `handleNodeResp` function in `edge/pkg/metamanager/client/node.go`, with the goal of achieving persistence at the edge after a successful `CreateNode` operation.
```go
func handleNodeResp(resource string, content []byte) (*api.Node, error) {
	...
	if reflect.DeepEqual(nodeResp.Err, apierrors.StatusError{}) {
		if err = updateNodeDB(resource, nodeResp.Object); err != nil {
			return nil, fmt.Errorf("update node meta failed, err: %v", err)
		}
		return nodeResp.Object, nil
	}
   ...
}

func updateNodeDB(resource string, node *api.Node) error {
	node.APIVersion = "v1"
	node.Kind = "Node"
	nodeContent, err := json.Marshal(node)
	if err != nil {
		klog.Errorf("unmarshal resp node failed, err: %v", err)
		return err
	}
	nodeKey := strings.Replace(resource,
		constants.ResourceSep+model.ResourceTypeNodePatch+constants.ResourceSep,
		constants.ResourceSep+model.ResourceTypeNode+constants.ResourceSep, 1)

	meta := &dao.Meta{
		Key:   nodeKey,
		Type:  model.ResourceTypeNode,
		Value: string(nodeContent)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		return err
	}
	return nil
}
```

- In the `patchPod` and `patchNode` functions within `cloudcore/cloud/pkg/edgecontroller/controller/upstream.go`, set the Object to nil in the returned message after successfully patching to the API server, thereby avoiding the return of the updated pod and node.
```go
func (uc *UpstreamController) patchNode() {
	...
	resMsg := model.NewMessage(msg.GetID()).
	SetResourceVersion(node.ResourceVersion).
	FillBody(&edgeapi.ObjectResp{Object: nil, Err: err}).
	BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
	if err = uc.messageLayer.Response(*resMsg); err != nil {
	klog.Warningf("Message: %s process failure, response failed with error: %v", msg.GetID(), err)
	continue
	}
	...
}

func (uc *UpstreamController) patchPod() {
	...
	resMsg := model.NewMessage(msg.GetID()).
	SetResourceVersion(updatedPod.ResourceVersion).
	FillBody(&edgeapi.ObjectResp{Object: nil, Err: err}).
	BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, msg.GetResource(), model.ResponseOperation)
	if err = uc.messageLayer.Response(*resMsg); err != nil {
	klog.Errorf("Message: %s process failure, response failed with error: %v", msg.GetID(), err)
	continue
	}
	...
}
```
- In the `Patch` function within both `edge/pkg/metamanager/client/node.go` and `edge/pkg/metamanager/client/pod.go`, after receiving the response from patching the status to the API server, if the update is successful, merge the patch data into the node and pod objects respectively, and then persist them to the SQLite database.
```go


//edge/pkg/metamanager/client/node.go
func (c *nodes) Patch(name string, data []byte) (*api.Node, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeNodePatch, name)
	nodeMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, string(data))
	resp, err := c.send.SendSync(nodeMsg)
	if err != nil {
		return nil, fmt.Errorf("update node failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to node failed, err: %v", err)
	}
	if resp.Router.Operation == model.ResponseErrorOperation {
		return nil, errors.New(string(content))
	}
	var nodeResp NodeResp
	err = json.Unmarshal(content, &nodeResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node failed, err: %v", err)
	}

	if reflect.DeepEqual(nodeResp.Err, apierrors.StatusError{}) {
		node, err := c.Get(name)
		if err != nil {
			return nil, err
		}
		toUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(node)
		if err != nil {
			return nil, err
		}
		originalObj := &unstructured.Unstructured{Object: toUnstructured}
		defaultScheme := scheme.Scheme
		defaulter := runtime.ObjectDefaulter(defaultScheme)
		updatedResource := new(unstructured.Unstructured)
		GroupVersionKind := originalObj.GroupVersionKind()
		schemaReferenceObj, err := defaultScheme.New(GroupVersionKind)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema reference object, err: %+v", err)
		}
		ctx := context.Background()
		if err = patchutil.StrategicPatchObject(ctx, defaulter, originalObj, data, updatedResource, schemaReferenceObj, ""); err != nil {
			return nil, err
		}
		updatedNode := &api.Node{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedResource.UnstructuredContent(), updatedNode); err != nil {
			return nil, err
		}
		if err = updateNodeDB(resource, updatedNode); err != nil {
			return nil, fmt.Errorf("update node meta failed, err: %v", err)
		}
		return updatedNode, nil
	}
	return nil, &nodeResp.Err
}


//edge/pkg/metamanager/client/pod.go
func (c *pods) Patch(name string, patchBytes []byte) (*corev1.Pod, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePodPatch, name)
	podMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, string(patchBytes))
	resp, err := c.send.SendSync(podMsg)
	if err != nil {
		return nil, fmt.Errorf("update pod failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to pod failed, err: %v", err)
	}

	if resp.Router.Operation == model.ResponseErrorOperation {
		return nil, errors.New(string(content))
	}

	var podResp PodResp
	err = json.Unmarshal(content, &podResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod failed, err: %v", err)
	}

	if reflect.DeepEqual(podResp.Err, apierrors.StatusError{}) {
		pod, err := c.Get(name)
		if err != nil {
			return nil, err
		}

		toUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
		if err != nil {
			return nil, err
		}
		originalObj := &unstructured.Unstructured{Object: toUnstructured}
		defaultScheme := scheme.Scheme
		defaulter := runtime.ObjectDefaulter(defaultScheme)
		updatedResource := new(unstructured.Unstructured)
		GroupVersionKind := originalObj.GroupVersionKind()
		schemaReferenceObj, err := defaultScheme.New(GroupVersionKind)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema reference object, err: %+v", err)
		}
		ctx := context.Background()
		if err = patchutil.StrategicPatchObject(ctx, defaulter, originalObj, patchBytes, updatedResource, schemaReferenceObj, ""); err != nil {
			return nil, err
		}
		updatedPod := &corev1.Pod{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedResource.UnstructuredContent(), updatedPod); err != nil {
			return nil, err
		}
		if err = updatePodDB(resource, updatedPod); err != nil {
			return nil, fmt.Errorf("update pod meta failed, err: %v", err)
		}
		return updatedPod, nil
	}
	return nil, &podResp.Err
}
```