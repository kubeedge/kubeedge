package controller

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgov1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/manager"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	routerv1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient kubernetes.Interface

	messageLayer messagelayer.MessageLayer

	podManager *manager.PodManager

	configmapManager *manager.ConfigMapManager

	secretManager *manager.SecretManager

	nodeManager *manager.NodesManager

	rulesManager *manager.RuleManager

	ruleEndpointsManager *manager.RuleEndpointManager

	lc *manager.LocationCache

	podLister clientgov1.PodLister
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
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			if !dc.lc.IsEdgeNode(pod.Spec.NodeName) {
				continue
			}
			resource, err := messagelayer.BuildResource(pod.Spec.NodeName, pod.Namespace, model.ResourceTypePod, pod.Name)
			if err != nil {
				klog.Warningf("built message resource failed with error: %s", err)
				continue
			}
			msg := model.NewMessage("").
				SetResourceVersion(pod.ResourceVersion).
				FillBody(pod)
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
				continue
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
				klog.Warningf("object type: %T unsupported", e.Object)
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
				resource, err := messagelayer.BuildResource(n, configMap.Namespace, model.ResourceTypeConfigmap, configMap.Name)
				if err != nil {
					klog.Warningf("build message resource failed with error: %s", err)
					continue
				}
				msg := model.NewMessage("").
					SetResourceVersion(configMap.ResourceVersion).
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, operation).
					FillBody(configMap)
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
				klog.Warningf("object type: %T unsupported", e.Object)
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
				resource, err := messagelayer.BuildResource(n, secret.Namespace, model.ResourceTypeSecret, secret.Name)
				if err != nil {
					klog.Warningf("build message resource failed with error: %s", err)
					continue
				}
				msg := model.NewMessage("").
					SetResourceVersion(secret.ResourceVersion).
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, operation).
					FillBody(secret)
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
				klog.Warningf("Object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				// update local cache
				dc.lc.UpdateEdgeNode(node.ObjectMeta.Name)
			case watch.Deleted:
				dc.lc.DeleteNode(node.ObjectMeta.Name)

				resource, err := messagelayer.BuildResource(node.Name, "namespace", constants.ResourceNode, node.Name)
				if err != nil {
					klog.Warningf("Built message resource failed with error: %s", err)
					break
				}
				msg := model.NewMessage("").
					BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
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

func (dc *DownstreamController) syncRule() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncRule loop")
			return
		case e := <-dc.rulesManager.Events():
			klog.V(4).Infof("Get rule events: event type: %s.", e.Type)
			rule, ok := e.Object.(*routerv1.Rule)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			klog.V(4).Infof("Get rule events: rule object: %+v.", rule)

			resource, err := messagelayer.BuildResourceForRouter(model.ResourceTypeRule, rule.Name)
			if err != nil {
				klog.Warningf("built message resource failed with error: %s", err)
				continue
			}
			msg := model.NewMessage("").
				SetResourceVersion(rule.ResourceVersion).
				FillBody(rule)
			switch e.Type {
			case watch.Added:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
			case watch.Deleted:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
			case watch.Modified:
				klog.Warningf("rule event type: %s unsupported", e.Type)
				continue
			default:
				klog.Warningf("rule event type: %s unsupported", e.Type)
				continue
			}
			if err := dc.messageLayer.Send(*msg); err != nil {
				klog.Warningf("send message failed with error: %s, operation: %s, resource: %s. Reason: %v", err, msg.GetOperation(), msg.GetResource(), err)
			} else {
				klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
			}
		}
	}
}

func (dc *DownstreamController) syncRuleEndpoint() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncRuleEndpoint loop")
			return
		case e := <-dc.ruleEndpointsManager.Events():
			klog.V(4).Infof("Get ruleEndpoint events: event type: %s.", e.Type)
			ruleEndpoint, ok := e.Object.(*routerv1.RuleEndpoint)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			klog.V(4).Infof("Get ruleEndpoint events: ruleEndpoint object: %+v.", ruleEndpoint)

			resource, err := messagelayer.BuildResourceForRouter(model.ResourceTypeRuleEndpoint, ruleEndpoint.Name)
			if err != nil {
				klog.Warningf("built message resource failed with error: %s", err)
				continue
			}
			msg := model.NewMessage("").
				SetResourceVersion(ruleEndpoint.ResourceVersion).
				FillBody(ruleEndpoint)
			switch e.Type {
			case watch.Added:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
			case watch.Deleted:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
			case watch.Modified:
				klog.Warningf("ruleEndpoint event type: %s unsupported", e.Type)
				continue
			default:
				klog.Warningf("ruleEndpoint event type: %s unsupported", e.Type)
				continue
			}
			if err := dc.messageLayer.Send(*msg); err != nil {
				klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
			} else {
				klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
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

	// rule
	go dc.syncRule()

	// ruleendpoint
	go dc.syncRuleEndpoint()

	return nil
}

// initLocating to know configmap and secret should send to which nodes
func (dc *DownstreamController) initLocating() error {
	set := labels.Set{manager.NodeRoleKey: manager.NodeRoleValue}
	selector := labels.SelectorFromSet(set)
	nodes, err := dc.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		dc.lc.UpdateEdgeNode(node.ObjectMeta.Name)
	}

	pods, err := dc.kubeClient.CoreV1().Pods(v1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
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
func NewDownstreamController(config *v1alpha1.EdgeController, k8sInformerFactory k8sinformers.SharedInformerFactory, keInformerFactory informers.KubeEdgeCustomInformer,
	crdInformerFactory crdinformers.SharedInformerFactory) (*DownstreamController, error) {
	lc := &manager.LocationCache{}

	podInformer := k8sInformerFactory.Core().V1().Pods()
	podManager, err := manager.NewPodManager(config, podInformer.Informer())
	if err != nil {
		klog.Warningf("create pod manager failed with error: %s", err)
		return nil, err
	}

	configMapInformer := k8sInformerFactory.Core().V1().ConfigMaps()
	configMapManager, err := manager.NewConfigMapManager(config, configMapInformer.Informer())
	if err != nil {
		klog.Warningf("create configmap manager failed with error: %s", err)
		return nil, err
	}

	secretInformer := k8sInformerFactory.Core().V1().Secrets()
	secretManager, err := manager.NewSecretManager(config, secretInformer.Informer())
	if err != nil {
		klog.Warningf("create secret manager failed with error: %s", err)
		return nil, err
	}
	nodeInformer := keInformerFactory.EdgeNode()
	nodesManager, err := manager.NewNodesManager(nodeInformer)
	if err != nil {
		klog.Warningf("Create nodes manager failed with error: %s", err)
		return nil, err
	}

	rulesInformer := crdInformerFactory.Rules().V1().Rules().Informer()
	rulesManager, err := manager.NewRuleManager(config, rulesInformer)
	if err != nil {
		klog.Warningf("Create rulesManager failed with error: %s", err)
		return nil, err
	}

	ruleEndpointsInformer := crdInformerFactory.Rules().V1().RuleEndpoints().Informer()
	ruleEndpointsManager, err := manager.NewRuleEndpointManager(config, ruleEndpointsInformer)
	if err != nil {
		klog.Warningf("Create ruleEndpointsManager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:           client.GetKubeClient(),
		podManager:           podManager,
		configmapManager:     configMapManager,
		secretManager:        secretManager,
		nodeManager:          nodesManager,
		messageLayer:         messagelayer.EdgeControllerMessageLayer(),
		lc:                   lc,
		podLister:            podInformer.Lister(),
		rulesManager:         rulesManager,
		ruleEndpointsManager: ruleEndpointsManager,
	}
	if err := dc.initLocating(); err != nil {
		return nil, err
	}

	return dc, nil
}
