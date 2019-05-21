package controller

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/utils"
	common "github.com/kubeedge/kubeedge/common/constants"
)

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient   *kubernetes.Clientset
	messageLayer messagelayer.MessageLayer

	podManager *manager.PodManager
	podStop    chan struct{}

	configmapManager *manager.ConfigMapManager
	configMapStop    chan struct{}

	secretManager *manager.SecretManager
	secretStop    chan struct{}

	nodeManager *manager.NodesManager
	nodesStop   chan struct{}

	serviceManager *manager.ServiceManager
	serviceStop    chan struct{}

	endpointsManager *manager.EndpointsManager
	endpointsStop    chan struct{}

	lc *manager.LocationCache
}

func (dc *DownstreamController) syncPod(stop chan struct{}) {
	running := true
	for running {
		select {
		case e := <-dc.podManager.Events():
			pod, ok := e.Object.(*v1.Pod)
			if !ok {
				log.LOGGER.Warnf("object type: %T unsupported", pod)
				continue
			}
			if !dc.lc.IsEdgeNode(pod.Spec.NodeName) {
				continue
			}
			msg := model.NewMessage("")
			resource, err := messagelayer.BuildResource(pod.Spec.NodeName, pod.Namespace, model.ResourceTypePod, pod.Name)
			if err != nil {
				log.LOGGER.Warnf("built message resource failed with error: %s", err)
				continue
			}
			msg.Content = pod
			switch e.Type {
			case watch.Added:
				msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
				dc.lc.AddOrUpdatePod(*pod)
			case watch.Deleted:
				msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
			case watch.Modified:
				msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
				dc.lc.AddOrUpdatePod(*pod)
			default:
				log.LOGGER.Warnf("pod event type: %s unsupported", e.Type)
			}
			if err := dc.messageLayer.Send(*msg); err != nil {
				log.LOGGER.Warnf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
			} else {
				log.LOGGER.Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
			}
		case <-stop:
			log.LOGGER.Infof("stop syncPod")
			running = false
		}
	}
}

func (dc *DownstreamController) syncConfigMap(stop chan struct{}) {
	running := true
	for running {
		select {
		case e := <-dc.configmapManager.Events():
			configMap, ok := e.Object.(*v1.ConfigMap)
			if !ok {
				log.LOGGER.Warnf("object type: %T unsupported", configMap)
				continue
			}
			nodes := dc.lc.ConfigMapNodes(configMap.Namespace, configMap.Name)
			log.LOGGER.Infof("there are %d nodes need to sync config map, operation: %s", len(nodes), e.Type)
			for _, n := range nodes {
				msg := model.NewMessage("")
				resource, err := messagelayer.BuildResource(n, configMap.Namespace, model.ResourceTypeConfigmap, configMap.Name)
				if err != nil {
					log.LOGGER.Warnf("build message resource failed with error: %s", err)
				}
				switch e.Type {
				case watch.Added:
					msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
				case watch.Modified:
					msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
				case watch.Deleted:
					msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
					dc.lc.DeleteConfigMap(configMap.Namespace, configMap.Name)
				default:
					// unsupported operation, no need to send to any node
					log.LOGGER.Warnf("config map event type: %s unsupported", e.Type)
					break
				}
				msg.Content = configMap
				err = dc.messageLayer.Send(*msg)
				if err != nil {
					log.LOGGER.Warnf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					log.LOGGER.Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
			}
		case <-stop:
			log.LOGGER.Infof("stop syncConfigMap")
			running = false
		}
	}
}

func (dc *DownstreamController) syncSecret(stop chan struct{}) {
	running := true
	for running {
		select {
		case e := <-dc.secretManager.Events():
			secret, ok := e.Object.(*v1.Secret)
			if !ok {
				log.LOGGER.Warnf("object type: %T unsupported", secret)
				continue
			}
			nodes := dc.lc.SecretNodes(secret.Namespace, secret.Name)
			log.LOGGER.Infof("there are %d nodes need to sync secret, operation: %s", len(nodes), e.Type)
			for _, n := range nodes {
				msg := model.NewMessage("")
				resource, err := messagelayer.BuildResource(n, secret.Namespace, model.ResourceTypeSecret, secret.Name)
				if err != nil {
					log.LOGGER.Warnf("build message resource failed with error: %s", err)
				}
				switch e.Type {
				case watch.Added:
					// TODO: rollback when all edge upgrade to 2.1.6 or upper
					fallthrough
				case watch.Modified:
					msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
				case watch.Deleted:
					msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
					dc.lc.DeleteSecret(secret.Namespace, secret.Name)
				default:
					// unsupported operation, no need to send to any node
					log.LOGGER.Warnf("secret event type: %s unsupported", e.Type)
					break
				}
				msg.Content = secret
				err = dc.messageLayer.Send(*msg)
				if err != nil {
					log.LOGGER.Warnf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					log.LOGGER.Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
			}
		case <-stop:
			log.LOGGER.Infof("stop syncSecret")
			running = false
		}
	}
}

func (dc *DownstreamController) syncEdgeNodes(stop chan struct{}) {
	running := true
	for running {
		select {
		case e := <-dc.nodeManager.Events():
			node, ok := e.Object.(*v1.Node)
			if !ok {
				log.LOGGER.Warnf("Object type: %T unsupported", node)
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
								log.LOGGER.Warnf("Built message resource failed with error: %s", err)
								break
							}
							msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
							svcs := dc.lc.GetAllServices()
							msg.Content = svcs
							if err := dc.messageLayer.Send(*msg); err != nil {
								log.LOGGER.Warnf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
							} else {
								log.LOGGER.Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
							}

							for _, svc := range svcs {
								pods, ok := dc.lc.GetServicePods(fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
								if ok {
									msg := model.NewMessage("")
									resource, err := messagelayer.BuildResource(node.Name, svc.Namespace, model.ResourceTypePodlist, svc.Name)
									if err != nil {
										log.LOGGER.Warnf("Built message resource failed with error: %v", err)
										continue
									}
									msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
									msg.Content = pods
									if err := dc.messageLayer.Send(*msg); err != nil {
										log.LOGGER.Warnf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
									} else {
										log.LOGGER.Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
									}
								}
							}

							// send all endpoints to edge
							msg = model.NewMessage("")
							// TODO: what should in place of namespace and endpoints when we are sending endpoints list ?
							resource, err = messagelayer.BuildResource(node.Name, "namespace", common.ResourceTypeEndpointsList, "endpoints")
							if err != nil {
								log.LOGGER.Warnf("Built message resource failed with error: %s", err)
								break
							}
							msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
							msg.Content = dc.lc.GetAllEndpoints()
							if err := dc.messageLayer.Send(*msg); err != nil {
								log.LOGGER.Warnf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
							} else {
								log.LOGGER.Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
							}
						}
						break
					}
				}
			case watch.Deleted:
				dc.lc.DeleteNode(node.ObjectMeta.Name)
			default:
				// unsupported operation, no need to send to any node
				log.LOGGER.Warnf("Node event type: %s unsupported", e.Type)
				break
			}
		case <-stop:
			log.LOGGER.Infof("Stop syncNodes")
			running = false
		}
	}
}

func (dc *DownstreamController) syncService(stop chan struct{}) {
	running := true
	var operation string
	for running {
		select {
		case e := <-dc.serviceManager.Events():
			svc, ok := e.Object.(*v1.Service)
			if !ok {
				log.LOGGER.Warnf("Object type: %T unsupported", svc)
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
				log.LOGGER.Warnf("Service event type: %s unsupported", e.Type)
				continue
			}

			// send to all nodes
			dc.lc.EdgeNodes.Range(func(key interface{}, value interface{}) bool {
				nodeName, ok := key.(string)
				if !ok {
					log.LOGGER.Warnf("Failed to assert key to sting")
					return true
				}
				msg := model.NewMessage("")
				resource, err := messagelayer.BuildResource(nodeName, svc.Namespace, common.ResourceTypeService, svc.Name)
				if err != nil {
					log.LOGGER.Warnf("Built message resource failed with error: %v", err)
					return true
				}
				msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, operation)
				msg.Content = svc
				if err := dc.messageLayer.Send(*msg); err != nil {
					log.LOGGER.Warnf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
				} else {
					log.LOGGER.Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
				}
				return true
			})
		case <-stop:
			log.LOGGER.Infof("Stop sync services")
			running = false
		}
	}
}

func (dc *DownstreamController) syncEndpoints(stop chan struct{}) {
	running := true
	var operation string
	for running {
		select {
		case e := <-dc.endpointsManager.Events():
			eps, ok := e.Object.(*v1.Endpoints)
			if !ok {
				log.LOGGER.Warnf("Object type: %T unsupported", eps)
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
				log.LOGGER.Warnf("endpoints event type: %s unsupported", e.Type)
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
					pods, err = dc.kubeClient.CoreV1().Pods(svc.Namespace).List(listOptions)
					if err == nil {
						dc.lc.AddOrUpdateServicePods(fmt.Sprintf("%s/%s", svc.Namespace, svc.Name), pods.Items)
					}
				}
				dc.lc.EdgeNodes.Range(func(key interface{}, value interface{}) bool {
					nodeName, check := key.(string)
					if !check {
						log.LOGGER.Warnf("Failed to assert key to sting")
						return true
					}
					msg := model.NewMessage("")
					resource, err := messagelayer.BuildResource(nodeName, eps.Namespace, common.ResourceTypeEndpoints, eps.Name)
					if err != nil {
						log.LOGGER.Warnf("Built message resource failed with error: %s", err)
						return true
					}
					msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, operation)
					msg.Content = eps
					if err := dc.messageLayer.Send(*msg); err != nil {
						log.LOGGER.Warnf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
					} else {
						log.LOGGER.Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
					}
					if operation != model.DeleteOperation && ok {
						msg := model.NewMessage("")
						resource, err := messagelayer.BuildResource(nodeName, svc.Namespace, model.ResourceTypePodlist, svc.Name)
						if err != nil {
							log.LOGGER.Warnf("Built message resource failed with error: %v", err)
							return true
						}
						msg.BuildRouter(constants.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
						msg.Content = pods.Items
						if err := dc.messageLayer.Send(*msg); err != nil {
							log.LOGGER.Warnf("Send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
						} else {
							log.LOGGER.Infof("Send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
						}
					}
					return true
				})
			}
		case <-stop:
			log.LOGGER.Infof("Stop sync endpoints")
			running = false
		}
	}
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	log.LOGGER.Infof("start downstream controller")
	// pod
	dc.podStop = make(chan struct{})
	go dc.syncPod(dc.podStop)

	// configmap
	dc.configMapStop = make(chan struct{})
	go dc.syncConfigMap(dc.configMapStop)

	// secret
	dc.secretStop = make(chan struct{})
	go dc.syncSecret(dc.secretStop)

	// nodes
	dc.nodesStop = make(chan struct{})
	go dc.syncEdgeNodes(dc.nodesStop)

	// service
	dc.serviceStop = make(chan struct{})
	go dc.syncService(dc.serviceStop)

	// endpoints
	dc.endpointsStop = make(chan struct{})
	go dc.syncEndpoints(dc.endpointsStop)

	return nil
}

// Stop DownstreamController
func (dc *DownstreamController) Stop() error {
	log.LOGGER.Infof("stop downstream controller")
	dc.podStop <- struct{}{}
	dc.configMapStop <- struct{}{}
	dc.secretStop <- struct{}{}
	dc.nodesStop <- struct{}{}
	dc.serviceStop <- struct{}{}
	dc.endpointsStop <- struct{}{}
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
	nodes, err := dc.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector.String()})
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

	if !config.EdgeSiteEnabled {
		pods, err = dc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{})
	} else {
		selector := fields.OneTermEqualSelector("spec.nodeName", config.KubeNodeName).String()
		pods, err = dc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{FieldSelector: selector})
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
		log.LOGGER.Warnf("create kube client failed with error: %s", err)
		return nil, err
	}

	var nodeName = ""
	if config.EdgeSiteEnabled {
		if config.KubeNodeName == "" {
			return nil, fmt.Errorf("kubeEdge node name is not provided in edgesite controller configuration")
		}
		nodeName = config.KubeNodeName
	}

	podManager, err := manager.NewPodManager(cli, v1.NamespaceAll, nodeName)
	if err != nil {
		log.LOGGER.Warnf("create pod manager failed with error: %s", err)
		return nil, err
	}

	configMapManager, err := manager.NewConfigMapManager(cli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("create configmap manager failed with error: %s", err)
		return nil, err
	}

	secretManager, err := manager.NewSecretManager(cli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("create secret manager failed with error: %s", err)
		return nil, err
	}

	nodesManager, err := manager.NewNodesManager(cli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("Create nodes manager failed with error: %s", err)
		return nil, err
	}

	serviceManager, err := manager.NewServiceManager(cli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("Create service manager failed with error: %s", err)
		return nil, err
	}

	endpointsManager, err := manager.NewEndpointsManager(cli, v1.NamespaceAll)
	if err != nil {
		log.LOGGER.Warnf("Create endpoints manager failed with error: %s", err)
		return nil, err
	}

	ml, err := messagelayer.NewMessageLayer()
	if err != nil {
		log.LOGGER.Warnf("create message layer failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{kubeClient: cli, podManager: podManager, configmapManager: configMapManager, secretManager: secretManager, nodeManager: nodesManager, serviceManager: serviceManager, endpointsManager: endpointsManager, messageLayer: ml, lc: lc}
	if err := dc.initLocating(); err != nil {
		return nil, err
	}

	return dc, nil
}
