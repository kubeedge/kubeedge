package application

import (
	"context"
	"fmt"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	kefeatures "github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

type Center struct {
	HandlerCenter
	messageLayer messagelayer.MessageLayer
	authConfig   *rest.Config
}

func NewApplicationCenter(dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory) *Center {
	a := &Center{
		HandlerCenter: NewHandlerCenter(dynamicSharedInformerFactory),
		authConfig:    client.GetAuthConfig(),
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

	resp, err := c.ProcessApplication(app)
	if err != nil {
		c.Response(app, msg.GetID(), metaserver.Rejected, err, nil)
		klog.Errorf("[metaserver/applicationCenter]failed to process Application(%+v), %v", app, err)
		return
	}
	c.Response(app, msg.GetID(), metaserver.Approved, nil, resp)
	klog.Infof("[metaserver/applicationCenter]successfully to process Application(%+v)", app)
}

func (c *Center) generateNewConfig(raw string) (*rest.Config, error) {
	parts := strings.SplitN(raw, " ", 3)
	if len(parts) < 2 || strings.ToLower(parts[0]) != "bearer" || len(parts[1]) <= 0 {
		return nil, fmt.Errorf("invalid request token format or length: %v", len(parts))
	}
	authConfig := rest.CopyConfig(c.authConfig)
	authConfig.BearerToken = parts[1]
	return authConfig, nil
}

func (c *Center) createAuthClient(app *metaserver.Application) (authorizationv1client.AuthorizationV1Interface, error) {
	authConfig, err := c.generateNewConfig(app.Token)
	if err != nil {
		return nil, err
	}
	return authorizationv1client.NewForConfigOrDie(authConfig), nil
}

func (c *Center) createKubeClient(app *metaserver.Application) (dynamic.Interface, error) {
	if !kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		return client.GetDynamicClient(), nil
	}
	authConfig, err := c.generateNewConfig(app.Token)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfigOrDie(authConfig), nil
}

func (c *Center) authorizeApplication(app *metaserver.Application, gvr schema.GroupVersionResource, namespace string, name string) error {
	if !kefeatures.DefaultFeatureGate.Enabled(kefeatures.RequireAuthorization) {
		return nil
	}
	tmpAuthClient, err := c.createAuthClient(app)
	if err != nil {
		return err
	}
	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   namespace,
				Verb:        string(app.Verb),
				Group:       gvr.Group,
				Resource:    gvr.Resource,
				Name:        name,
				Subresource: app.Subresource,
			},
		},
	}
	response, err := tmpAuthClient.SelfSubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	if response.Status.Allowed {
		return nil
	}
	var errMsg = fmt.Sprintf("resource %v authorize failed.", gvr)
	if len(response.Status.Reason) > 0 {
		errMsg += fmt.Sprintf("reason: %v.", response.Status.Reason)
	}
	if len(response.Status.EvaluationError) > 0 {
		errMsg += fmt.Sprintf("evaluation error: %v.", response.Status.EvaluationError)
	}
	return fmt.Errorf(errMsg)
}

// ProcessApplication processes application by re-translating it to kube-api request with kube client,
// which will be processed and responded by apiserver eventually.
// Specially if app.verb == watch, it transforms app to a listener and register it to HandlerCenter, rather
// than request to apiserver directly. Listener will then continuously listen kube-api change events and
// push them to edge node.
func (c *Center) ProcessApplication(app *metaserver.Application) (interface{}, error) {
	app.Status = metaserver.InProcessing
	gvr, ns, name := metaserver.ParseKey(app.Key)
	var kubeClient dynamic.Interface
	var err error
	if app.Verb != metaserver.Watch {
		kubeClient, err = c.createKubeClient(app)
		if err != nil {
			klog.Errorf("create kube client error: %v", err)
			return nil, err
		}
	} else {
		err := c.authorizeApplication(app, gvr, ns, name)
		if err != nil {
			klog.Errorf("authorize application error: %v", err)
			return nil, err
		}
	}

	switch app.Verb {
	case metaserver.List:
		var option = new(metav1.ListOptions)
		if err := app.OptionTo(option); err != nil {
			return nil, err
		}
		list, err := kubeClient.Resource(gvr).Namespace(ns).List(context.TODO(), *option)
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
		retObj, err := kubeClient.Resource(gvr).Namespace(ns).Get(context.TODO(), name, *option)
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
			retObj, err = kubeClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, *option)
		} else {
			retObj, err = kubeClient.Resource(gvr).Namespace(ns).Create(context.TODO(), obj, *option, app.Subresource)
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
		if err := kubeClient.Resource(gvr).Namespace(ns).Delete(context.TODO(), name, *option); err != nil {
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
			retObj, err = kubeClient.Resource(gvr).Namespace(ns).Update(context.TODO(), obj, *option)
		} else {
			retObj, err = kubeClient.Resource(gvr).Namespace(ns).Update(context.TODO(), obj, *option, app.Subresource)
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
		retObj, err := kubeClient.Resource(gvr).Namespace(ns).UpdateStatus(context.TODO(), obj, *option)
		if err != nil {
			return nil, err
		}
		return retObj, nil
	case metaserver.Patch:
		var pi = new(metaserver.PatchInfo)
		if err := app.OptionTo(pi); err != nil {
			return nil, err
		}
		retObj, err := kubeClient.Resource(gvr).Namespace(ns).Patch(context.TODO(), pi.Name, pi.PatchType, pi.Data, pi.Options, pi.Subresources...)
		if err != nil {
			return nil, err
		}
		return retObj, nil
	default:
		return nil, fmt.Errorf("unsupported Application Verb type :%v", app.Verb)
	}
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
	gvr, ns, name := metaserver.ParseKey(watchApp.Key)

	err := c.authorizeApplication(watchApp, gvr, ns, name)
	if err != nil {
		return fmt.Errorf("authorize application error: %v", err)
	}

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
