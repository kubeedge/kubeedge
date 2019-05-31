package controller

import (
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/utils"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
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
				dc.lc.UpdateEdgeNode(node.ObjectMeta.Name)
			case watch.Modified:
				continue
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

	return nil
}

// Stop DownstreamController
func (dc *DownstreamController) Stop() error {
	log.LOGGER.Infof("stop downstream controller")
	dc.podStop <- struct{}{}
	dc.configMapStop <- struct{}{}
	dc.secretStop <- struct{}{}
	dc.nodesStop <- struct{}{}
	return nil
}

// initLocating to know configmap and secret should send to which nodes
func (dc *DownstreamController) initLocating() error {
	set := labels.Set{manager.NodeRoleKey: manager.NodeRoleValue}
	selector := labels.SelectorFromSet(set)
	nodes, err := dc.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		dc.lc.UpdateEdgeNode(node.ObjectMeta.Name)
	}
	pods, err := dc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{})
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

	podManager, err := manager.NewPodManager(cli, v1.NamespaceAll)
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

	ml, err := messagelayer.NewMessageLayer()
	if err != nil {
		log.LOGGER.Warnf("create message layer failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{kubeClient: cli, podManager: podManager, configmapManager: configMapManager, secretManager: secretManager, nodeManager: nodesManager, messageLayer: ml, lc: lc}
	if err := dc.initLocating(); err != nil {
		return nil, err
	}

	return dc, nil
}
