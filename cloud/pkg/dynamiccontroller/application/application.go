package application

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/util/pass-through"
)

type Center struct {
	HandlerCenter
	messageLayer  messagelayer.MessageLayer
	dynamicClient dynamic.Interface
	kubeClient    kubernetes.Interface
}

func NewApplicationCenter(dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory) *Center {
	a := &Center{
		HandlerCenter: NewHandlerCenter(dynamicSharedInformerFactory),
		dynamicClient: client.GetDynamicClient(),
		kubeClient:    client.GetKubeClient(),
		messageLayer:  messagelayer.DynamicControllerMessageLayer(),
	}
	return a
}

// Process translate msg to application , process and send resp to edge
// TODO: upgrade to parallel process
func (c *Center) Process(msg model.Message) {
	if strings.HasSuffix(msg.GetResource(), metaserver.WatchAppSync) {
		if err := c.ProcessWatchSync(msg); err != nil {
			klog.Errorf("failed to ProcessWatchSync: %v", err)
		}
		return
	}

	app, err := metaserver.MsgToApplication(msg)
	if err != nil {
		klog.Errorf("failed to translate msg to Application: %v", err)
		return
	}

	klog.Infof("[metaserver/ApplicationCenter] get a Application %v", app.String())

	if passthrough.IsPassThroughPath(app.Key, string(app.Verb)) {
		resp, err := c.passThroughRequest(app)
		if err != nil {
			c.Response(app, msg.GetID(), metaserver.Rejected, err, nil)
			klog.Errorf("[metaserver/passThrough]failed to process Application(%+v), %v", app, err)
			return
		}
		c.Response(app, msg.GetID(), metaserver.Approved, nil, resp)
		return
	}

	resp, err := c.ProcessApplication(app)
	if err != nil {
		c.Response(app, msg.GetID(), metaserver.Rejected, err, nil)
		klog.Errorf("[metaserver/applicationCenter]failed to process Application(%+v), %v", app, err)
		return
	}
	c.Response(app, msg.GetID(), metaserver.Approved, nil, resp)
	klog.Infof("[metaserver/applicationCenter]successfully to process Application(%+v)", app)
}

// ProcessApplication processes application by re-translating it to kube-api request with kube client,
// which will be processed and responded by apiserver eventually.
// Specially if app.verb == watch, it transforms app to a listener and register it to HandlerCenter, rather
// than request to apiserver directly. Listener will then continuously listen kube-api change events and
// push them to edge node.
func (c *Center) ProcessApplication(app *metaserver.Application) (interface{}, error) {
	app.Status = metaserver.InProcessing
	gvr, ns, name := metaserver.ParseKey(app.Key)

	switch app.Verb {
	case metaserver.List:
		var option = new(metav1.ListOptions)
		if err := app.OptionTo(option); err != nil {
			return nil, err
		}
		list, err := c.dynamicClient.Resource(gvr).Namespace(ns).List(context.TODO(), *option)
		if err != nil {
			return nil, fmt.Errorf("get current list error: %v", err)
		}
		return list, nil
	case metaserver.Watch:
		listener, err := applicationToListener(app)
		if err != nil {
			return nil, err
		}

		if err := c.HandlerCenter.AddListener(listener); err != nil {
			return nil, fmt.Errorf("failed to add listener, %v", err)
		}
		return nil, nil
	case metaserver.Get:
		var option = new(metav1.GetOptions)
		if err := app.OptionTo(option); err != nil {
			return nil, err
		}
		retObj, err := c.dynamicClient.Resource(gvr).Namespace(ns).Get(context.TODO(), name, *option)
		if err != nil {
			return nil, err
		}
		return retObj, nil
	case metaserver.Create:
		var option = new(metav1.CreateOptions)
		if err := app.OptionTo(option); err != nil {
			return nil, err
		}
		var obj = new(unstructured.Unstructured)
		if err := app.ReqBodyTo(obj); err != nil {
			return nil, err
		}
		var retObj interface{}
		var err error
		if app.Subresource == "" {
			retObj, err = c.dynamicClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, *option)
		} else {
			retObj, err = c.dynamicClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, *option, app.Subresource)
		}
		if err != nil {
			return nil, err
		}
		return retObj, err
	case metaserver.Delete:
		var option = new(metav1.DeleteOptions)
		if err := app.OptionTo(&option); err != nil {
			return nil, err
		}
		if err := c.dynamicClient.Resource(gvr).Namespace(ns).Delete(context.TODO(), name, *option); err != nil {
			return nil, err
		}
		return nil, nil
	case metaserver.Update:
		var option = new(metav1.UpdateOptions)
		if err := app.OptionTo(option); err != nil {
			return nil, err
		}
		var obj = new(unstructured.Unstructured)
		if err := app.ReqBodyTo(obj); err != nil {
			return nil, err
		}
		var retObj interface{}
		var err error
		if app.Subresource == "" {
			retObj, err = c.dynamicClient.Resource(gvr).Namespace(ns).Update(context.TODO(), obj, *option)
		} else {
			retObj, err = c.dynamicClient.Resource(gvr).Namespace(ns).Update(context.TODO(), obj, *option, app.Subresource)
		}
		if err != nil {
			return nil, err
		}
		return retObj, nil
	case metaserver.UpdateStatus:
		var option = new(metav1.UpdateOptions)
		if err := app.OptionTo(option); err != nil {
			return nil, err
		}
		var obj = new(unstructured.Unstructured)
		if err := app.ReqBodyTo(obj); err != nil {
			return nil, err
		}
		retObj, err := c.dynamicClient.Resource(gvr).Namespace(ns).UpdateStatus(context.TODO(), obj, *option)
		if err != nil {
			return nil, err
		}
		return retObj, nil
	case metaserver.Patch:
		var pi = new(metaserver.PatchInfo)
		if err := app.OptionTo(pi); err != nil {
			return nil, err
		}
		retObj, err := c.dynamicClient.Resource(gvr).Namespace(ns).Patch(context.TODO(), pi.Name, pi.PatchType, pi.Data, pi.Options, pi.Subresources...)
		if err != nil {
			return nil, err
		}
		return retObj, nil
	default:
		return nil, fmt.Errorf("unsupported Application Verb type :%v", app.Verb)
	}
}

func (c *Center) passThroughRequest(app *metaserver.Application) (interface{}, error) {
	kubeClient, ok := c.kubeClient.(*kubernetes.Clientset)
	if !ok {
		return nil, fmt.Errorf("converting kubeClient to *kubernetes.Clientset type failed")
	}
	verb := strings.ToUpper(string(app.Verb))
	return kubeClient.RESTClient().Verb(verb).AbsPath(app.Key).Body(app.ReqBody).Do(context.TODO()).Raw()
}

// Response update application, generate and send resp message to edge
func (c *Center) Response(app *metaserver.Application, parentID string, status metaserver.ApplicationStatus, err error, respContent interface{}) {
	app.Status = status
	if err != nil {
		apierr, ok := err.(apierrors.APIStatus)
		if ok {
			app.Error = apierrors.StatusError{ErrStatus: apierr.Status()}
		} else {
			app.Reason = err.Error()
		}
	}
	if respContent != nil {
		if app.Verb == metaserver.List || app.Verb == metaserver.Get {
			filter.MessageFilter(respContent, app.Nodename)
		}
		app.RespBody = metaserver.ToBytes(respContent)
	}

	resource, err := messagelayer.BuildResource(app.Nodename, metaserver.Ignore, metaserver.ApplicationResource, metaserver.Ignore)
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return
	}
	msg := model.NewMessage(parentID).
		BuildRouter(modules.DynamicControllerModuleName, message.ResourceGroupName, resource, metaserver.ApplicationResp).
		FillBody(app)

	if err := c.messageLayer.Response(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
		return
	}
	klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
}

// ProcessWatchSync process watch sync message
func (c *Center) ProcessWatchSync(msg model.Message) error {
	nodeID, err := messagelayer.GetNodeID(msg)
	if err != nil {
		return err
	}

	applications, err := metaserver.MsgToApplications(msg)
	if err != nil {
		return fmt.Errorf("failed translate msg to Applications: %v", err)
	}

	addedWatchApp, removedListeners := c.getWatchDiff(applications, nodeID)

	// gc already removed listeners
	for _, listener := range removedListeners {
		c.HandlerCenter.DeleteListener(listener)
	}

	failedWatchApp := make(map[string]metaserver.Application)

	// add listener for new added watch app
	for _, watchApp := range addedWatchApp {
		err := c.processWatchApp(&watchApp)
		if err != nil {
			watchApp.Status = metaserver.Rejected
			apiErr, ok := err.(apierrors.APIStatus)
			if ok {
				watchApp.Error = apierrors.StatusError{ErrStatus: apiErr.Status()}
			} else {
				watchApp.Reason = err.Error()
			}
			failedWatchApp[watchApp.ID] = watchApp
			klog.Errorf("processWatchApp %s err: %v", watchApp.String(), err)
		}
	}

	respMsg := model.NewMessage(msg.GetID()).
		BuildRouter(modules.DynamicControllerModuleName, message.ResourceGroupName, msg.GetResource(), metaserver.ApplicationResp).
		FillBody(failedWatchApp)

	if err := c.messageLayer.Response(*respMsg); err != nil {
		klog.Warningf("send message failed error: %s, operation: %s, resource: %s", err, respMsg.GetOperation(), respMsg.GetResource())
		return err
	}

	return nil
}

func (c *Center) getWatchDiff(allWatchAppInEdge map[string]metaserver.Application,
	nodeID string) ([]metaserver.Application, []*SelectorListener) {
	listenerInCloud := c.HandlerCenter.GetListenersForNode(nodeID)

	addedWatchApp := make([]metaserver.Application, 0)
	for ID, app := range allWatchAppInEdge {
		if _, exist := listenerInCloud[ID]; !exist {
			addedWatchApp = append(addedWatchApp, app)
			klog.Infof("added watch app %s", app.String())
		}
	}

	removedListeners := make([]*SelectorListener, 0)
	for ID, listener := range listenerInCloud {
		if _, exist := allWatchAppInEdge[ID]; !exist {
			removedListeners = append(removedListeners, listener)
			klog.Infof("need removed listener %s", listener.id)
		}
	}

	return addedWatchApp, removedListeners
}

func (c *Center) processWatchApp(watchApp *metaserver.Application) error {
	watchApp.Status = metaserver.InProcessing
	listener, err := applicationToListener(watchApp)
	if err != nil {
		return err
	}

	if err := c.HandlerCenter.AddListener(listener); err != nil {
		return fmt.Errorf("failed to add listener, %v", err)
	}

	return nil
}

func applicationToListener(app *metaserver.Application) (*SelectorListener, error) {
	var option = new(metav1.ListOptions)
	if err := app.OptionTo(option); err != nil {
		return nil, err
	}

	gvr, namespace, _ := metaserver.ParseKey(app.Key)
	selector := NewSelector(option.LabelSelector, option.FieldSelector)
	if namespace != "" {
		selector.Field = fields.AndSelectors(selector.Field, fields.OneTermEqualSelector("metadata.namespace", namespace))
	}

	return NewSelectorListener(app.ID, app.Nodename, gvr, selector), nil
}
