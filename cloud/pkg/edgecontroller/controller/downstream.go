package controller

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/utils"
	common "github.com/kubeedge/kubeedge/common/constants"
)

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient   *kubernetes.Clientset
	messageLayer messagelayer.MessageLayer

	podManager *manager.PodManager

	configmapManager *manager.ConfigMapManager

	secretManager *manager.SecretManager

	nodeManager *manager.NodesManager

	serviceManager *manager.ServiceManager

	endpointsManager *manager.EndpointsManager

	lc *manager.LocationCache
}

func (dc *DownstreamController) syncPod() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncPod loop")
			return
		case e := <-dc.podManager.Events():
			pod, ok := e.Object.(*v1.Pod)
			if !ok {
				klog.Warningf("object type: %T unsupported", pod)
				continue
			}
			if !dc.lc.IsEdgeNode(pod.Spec.NodeName) {
				continue
			}
			msg := model.NewMessage("")
			msg.SetResourceVersion(pod.ResourceVersion)
			resource, err := messagelayer.BuildResource(pod.Spec.NodeName, pod.Namespace, model.ResourceTypePod, pod.Name)
			if err != nil {
				klog.Warningf("built message resource failed with error: %s", err)
				continue
			}
			msg.Content = pod
			switch e.Type {
			case watch.Added:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
				dc.lc.AddOrUpdatePod(*pod)
			case watch.Deleted:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
			case watch.Modified:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
				dc.lc.AddOrUpdatePod(*pod)
			default:
				klog.Warningf("pod event type: %s unsupported", e.Type)
			}
			if err := dc.messageLayer.Send(*msg); err != nil {
				klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
			} else {
				klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
			}
		}
	}
}

func (dc *DownstreamController) syncConfigMap() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncConfigMap loop")
			return
		case e := <-dc.configmapManager.Events():
			configMap, ok := e.Object.(*v1.ConfigMap)
			if !ok {
				klog.Warningf("object type: %T unsupported", configMap)
				continue
			}
			var operation string
			switch e.Type {
			case watch.Added:
				operation = model.InsertOperation
			case watch.Modified:
				operation = model.UpdateOperation
			case watch.Deleted:
				operation = model.DeleteOperation
			default:
				// unsupported operation, no need to send to any node
				klog.Warningf("config map event type: %s unsupported", e.Type)
				continue // continue to next select
			}

			nodes := dc.lc.ConfigMapNodes(configMap.Namespace, configMap.Name)
			if e.Type == watch.Deleted {
				dc.lc.DeleteConfigMap(configMap.Namespace, configMap.Name)
			}
			klog.V(4).Infof("there are %d nodes need to sync config map, operation: %s", len(nodes), e.Type)
			for _, n := range nodes {
				msg := model.NewMessage("")
				msg.SetResourceVersion(configMap.ResourceVersion)
				resource, err := messagelayer.BuildResource(n, configMap.Namespace, model.ResourceTypeConfigmap, configMap.Name)
				if err != nil {
					klog.Warningf("build message resource failed with error: %s", err)
					continue
				}
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, operation)
				msg.Content = configMap
				err = dc.messageLayer.Send(*msg)
				if err != nil {
					klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
			}
		}
	}
}

func (dc *DownstreamController) syncSecret() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncSecret loop")
			return
		case e := <-dc.secretManager.Events():
			secret, ok := e.Object.(*v1.Secret)
			if !ok {
				klog.Warningf("object type: %T unsupported", secret)
				continue
			}
			var operation string
			switch e.Type {
			case watch.Added:
				// TODO: rollback when all edge upgrade to 2.1.6 or upper
				fallthrough
			case watch.Modified:
				operation = model.UpdateOperation
			case watch.Deleted:
				operation = model.DeleteOperation
			default:
				// unsupported operation, no need to send to any node
				klog.Warningf("secret event type: %s unsupported", e.Type)
				continue // continue to next select
			}

			nodes := dc.lc.SecretNodes(secret.Namespace, secret.Name)
			if e.Type == watch.Deleted {
				dc.lc.DeleteSecret(secret.Namespace, secret.Name)
			}
			klog.V(4).Infof("there are %d nodes need to sync secret, operation: %s", len(nodes), e.Type)
			for _, n := range nodes {
				msg := model.NewMessage("")
				msg.SetResourceVersion(secret.ResourceVersion)
				resource, err := messagelayer.BuildResource(n, secret.Namespace, model.ResourceTypeSecret, secret.Name)
				if err != nil {
					klog.Warningf("build message resource failed with error: %s", err)
					continue
				}
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, operation)
				msg.Content = secret
				err = dc.messageLayer.Send(*msg)
				if err != nil {
					klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
			}
		}
	}
}

func (dc *DownstreamController) syncEdgeNodes() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncEdgeNodes loop")
			return
		case e := <-dc.nodeManager.Events():
			node, ok := e.Object.(*v1.Node)
			if !ok {
				klog.Warningf("Object type: %T unsupported", node)
				continue
			}
			switch e.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				// When node comes to running, send all the service/endpoints/pods information to edge
				for _, nsc := range node.Status.Conditions {
					if nsc.Type == "Ready" {
						status, ok := dc.lc.GetNodeStatus(node.ObjectMeta.Name)
						dc.lc.UpdateEdgeNode(node.ObjectMeta.Name, string(nsc.Status))
						if nsc.Status == "True" && (!ok || status != "True") {
							// send all services to edge
							msg := model.NewMessage("")
							// TODO: what should in place of namespace and service when we are sending service list ?
							resource, err := messagelayer.BuildResource(node.Name, "namespace", common.ResourceTypeServiceList, "service")
							if err != nil {
								klog.Warningf("Built message resource failed with error: %s", err)
								break
							}
							msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
							svcs := dc.lc.GetAllServices()
							msg.Content = svcs
							if err := dc.messageLayer.Send(*msg); err != nil {
								klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
							} else {
								klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
							}

							for _, svc := range svcs {
								pods, ok := dc.lc.GetServicePods(fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
								if ok {
									msg := model.NewMessage("")
									resource, err := messagelayer.BuildResource(node.Name, svc.Namespace, model.ResourceTypePodlist, svc.Name)
									if err != nil {
										klog.Warningf("Built message resource failed with error: %v", err)
										continue
									}
									msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
									msg.Content = pods
									if err := dc.messageLayer.Send(*msg); err != nil {
										klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
									} else {
										klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
									}
								}
							}

							// send all endpoints to edge
							msg = model.NewMessage("")
							// TODO: what should in place of namespace and endpoints when we are sending endpoints list ?
							resource, err = messagelayer.BuildResource(node.Name, "namespace", common.ResourceTypeEndpointsList, "endpoints")
							if err != nil {
								klog.Warningf("Built message resource failed with error: %s", err)
								break
							}
							msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
							msg.Content = dc.lc.GetAllEndpoints()
							if err := dc.messageLayer.Send(*msg); err != nil {
								klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
							} else {
								klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
							}
						}
						break
					}
				}
			case watch.Deleted:
				dc.lc.DeleteNode(node.ObjectMeta.Name)

				msg := model.NewMessage("")
				resource, err := messagelayer.BuildResource(node.Name, "namespace", constants.ResourceNode, node.Name)
				if err != nil {
					klog.Warningf("Built message resource failed with error: %s", err)
					break
				}
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
				err = dc.messageLayer.Send(*msg)
				if err != nil {
					klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
			default:
				// unsupported operation, no need to send to any node
				klog.Warningf("Node event type: %s unsupported", e.Type)
			}
		}
	}
}

func (dc *DownstreamController) syncService() {
	var operation string
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncService loop")
			return
		case e := <-dc.serviceManager.Events():
			svc, ok := e.Object.(*v1.Service)
			if !ok {
				klog.Warningf("Object type: %T unsupported", svc)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.lc.AddOrUpdateService(*svc)
				operation = model.InsertOperation
			case watch.Modified:
				dc.lc.AddOrUpdateService(*svc)
				operation = model.UpdateOperation
			case watch.Deleted:
				dc.lc.DeleteService(*svc)
				operation = model.DeleteOperation
			default:
				// unsupported operation, no need to send to any node
				klog.Warningf("Service event type: %s unsupported", e.Type)
				continue
			}

			// send to all nodes
			dc.lc.EdgeNodes.Range(func(key interface{}, value interface{}) bool {
				nodeName, ok := key.(string)
				if !ok {
					klog.Warning("Failed to assert key to sting")
					return true
				}
				msg := model.NewMessage("")
				msg.SetResourceVersion(svc.ResourceVersion)
				resource, err := messagelayer.BuildResource(nodeName, svc.Namespace, common.ResourceTypeService, svc.Name)
				if err != nil {
					klog.Warningf("Built message resource failed with error: %v", err)
					return true
				}
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, operation)
				msg.Content = svc
				if err := dc.messageLayer.Send(*msg); err != nil {
					klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
				return true
			})
		}
	}
}

func (dc *DownstreamController) syncEndpoints() {
	var operation string
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncEndpoints loop")
			return
		case e := <-dc.endpointsManager.Events():
			eps, ok := e.Object.(*v1.Endpoints)
			if !ok {
				klog.Warningf("Object type: %T unsupported", eps)
				continue
			}

			ok = true
			switch e.Type {
			case watch.Added:
				dc.lc.AddOrUpdateEndpoints(*eps)
				operation = model.InsertOperation
			case watch.Modified:
				ok = dc.lc.IsEndpointsUpdated(*eps)
				dc.lc.AddOrUpdateEndpoints(*eps)
				operation = model.UpdateOperation
			case watch.Deleted:
				dc.lc.DeleteEndpoints(*eps)
				dc.lc.DeleteServicePods(*eps)
				operation = model.DeleteOperation
			default:
				// unsupported operation, no need to send to any node
				klog.Warningf("endpoints event type: %s unsupported", e.Type)
				continue
			}
			// send to all nodes
			if ok {
				var listOptions metav1.ListOptions
				var pods *v1.PodList
				var err error
				svc, ok := dc.lc.GetService(fmt.Sprintf("%s/%s", eps.Namespace, eps.Name))
				if ok {
					labelSelectorString := ""
					for key, value := range svc.Spec.Selector {
						labelSelectorString = labelSelectorString + key + "=" + value + ","
					}
					labelSelectorString = strings.TrimSuffix(labelSelectorString, ",")
					listOptions = metav1.ListOptions{
						LabelSelector: labelSelectorString,
						Limit:         100,
					}
					pods, err = dc.kubeClient.CoreV1().Pods(svc.Namespace).List(context.Background(), listOptions)
					if err == nil {
						dc.lc.AddOrUpdateServicePods(fmt.Sprintf("%s/%s", svc.Namespace, svc.Name), pods.Items)
					}
				}
				dc.lc.EdgeNodes.Range(func(key interface{}, value interface{}) bool {
					nodeName, check := key.(string)
					if !check {
						klog.Warning("Failed to assert key to sting")
						return true
					}
					msg := model.NewMessage("")
					msg.SetResourceVersion(eps.ResourceVersion)
					resource, err := messagelayer.BuildResource(nodeName, eps.Namespace, common.ResourceTypeEndpoints, eps.Name)
					if err != nil {
						klog.Warningf("Built message resource failed with error: %s", err)
						return true
					}
					msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, operation)
					msg.Content = eps
					if err := dc.messageLayer.Send(*msg); err != nil {
						klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
					} else {
						klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
					}
					if operation != model.DeleteOperation && ok {
						msg := model.NewMessage("")
						resource, err := messagelayer.BuildResource(nodeName, svc.Namespace, model.ResourceTypePodlist, svc.Name)
						if err != nil {
							klog.Warningf("Built message resource failed with error: %v", err)
							return true
						}
						msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
						msg.Content = pods.Items
						if err := dc.messageLayer.Send(*msg); err != nil {
							klog.Warningf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
						} else {
							klog.V(4).Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
						}
					}
					return true
				})
			}
		}
	}
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("start downstream controller")
	// pod
	go dc.syncPod()

	// configmap
	go dc.syncConfigMap()

	// secret
	go dc.syncSecret()

	// nodes
	go dc.syncEdgeNodes()

	// service
	go dc.syncService()

	// endpoints
	go dc.syncEndpoints()

	return nil
}

// initLocating to know configmap and secret should send to which nodes
func (dc *DownstreamController) initLocating() error {
	var (
		pods *v1.PodList
		err  error
	)

	set := labels.Set{manager.NodeRoleKey: manager.NodeRoleValue}
	selector := labels.SelectorFromSet(set)
	nodes, err := dc.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	var status string
	for _, node := range nodes.Items {
		for _, nsc := range node.Status.Conditions {
			if nsc.Type == "Ready" {
				status = string(nsc.Status)
				break
			}
		}
		dc.lc.UpdateEdgeNode(node.ObjectMeta.Name, status)
	}

	if !config.Config.EdgeSiteEnable {
		pods, err = dc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	} else {
		selector := fields.OneTermEqualSelector("spec.nodeName", config.Config.NodeName).String()
		pods, err = dc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{FieldSelector: selector})
	}
	if err != nil {
		return err
	}
	for _, p := range pods.Items {
		if dc.lc.IsEdgeNode(p.Spec.NodeName) {
			dc.lc.AddOrUpdatePod(p)
		}
	}

	return nil
}

// NewDownstreamController create a DownstreamController from config
func NewDownstreamController() (*DownstreamController, error) {
	lc := &manager.LocationCache{}

	cli, err := utils.KubeClient()
	if err != nil {
		klog.Warningf("create kube client failed with error: %s", err)
		return nil, err
	}

	var nodeName = ""
	if config.Config.EdgeSiteEnable {
		if config.Config.NodeName == "" {
			return nil, fmt.Errorf("kubeEdge node name is not provided in edgesite controller configuration")
		}
		nodeName = config.Config.NodeName
	}

	podManager, err := manager.NewPodManager(cli, v1.NamespaceAll, nodeName)
	if err != nil {
		klog.Warningf("create pod manager failed with error: %s", err)
		return nil, err
	}

	configMapManager, err := manager.NewConfigMapManager(cli, v1.NamespaceAll)
	if err != nil {
		klog.Warningf("create configmap manager failed with error: %s", err)
		return nil, err
	}

	secretManager, err := manager.NewSecretManager(cli, v1.NamespaceAll)
	if err != nil {
		klog.Warningf("create secret manager failed with error: %s", err)
		return nil, err
	}

	nodesManager, err := manager.NewNodesManager(cli, v1.NamespaceAll)
	if err != nil {
		klog.Warningf("Create nodes manager failed with error: %s", err)
		return nil, err
	}

	serviceManager, err := manager.NewServiceManager(cli, v1.NamespaceAll)
	if err != nil {
		klog.Warningf("Create service manager failed with error: %s", err)
		return nil, err
	}

	endpointsManager, err := manager.NewEndpointsManager(cli, v1.NamespaceAll)
	if err != nil {
		klog.Warningf("Create endpoints manager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:       cli,
		podManager:       podManager,
		configmapManager: configMapManager,
		secretManager:    secretManager,
		nodeManager:      nodesManager,
		serviceManager:   serviceManager,
		endpointsManager: endpointsManager,
		messageLayer:     messagelayer.NewContextMessageLayer(),
		lc:               lc,
	}
	if err := dc.initLocating(); err != nil {
		return nil, err
	}

	return dc, nil
}
