package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

// PodsGetter is interface to get pods
type PodsGetter interface {
	Pods(namespace string) PodsInterface
}

// PodsInterface is pod interface
type PodsInterface interface {
	Create(*corev1.Pod) (*corev1.Pod, error)
	Update(*corev1.Pod) error
	Patch(name string, patchBytes []byte) (*corev1.Pod, error)
	Delete(name string, options metav1.DeleteOptions) error
	Get(name string) (*corev1.Pod, error)
}

type pods struct {
	namespace string
	send      SendInterface
}

// PodResp represents pod response from the api-server
type PodResp struct {
	Object *corev1.Pod
	Err    apierrors.StatusError
}

func newPods(namespace string, s SendInterface) *pods {
	return &pods{
		send:      s,
		namespace: namespace,
	}
}

func (c *pods) Create(cm *corev1.Pod) (*corev1.Pod, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePod, cm.Name)
	podMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, *cm)
	resp, err := c.send.SendSync(podMsg)
	if err != nil {
		return nil, fmt.Errorf("create pod failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to pod failed, err: %v", err)
	}

	return handlePodResp(resource, content)
}

func (c *pods) Update(*corev1.Pod) error {
	return nil
}

func (c *pods) Delete(name string, options metav1.DeleteOptions) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePod, name)
	podDeleteMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.DeleteOperation, options)
	msg, err := c.send.SendSync(podDeleteMsg)
	if err != nil {
		return err
	}

	content, ok := msg.Content.(string)
	if ok && content == constants.MessageSuccessfulContent {
		return nil
	}

	err, ok = msg.Content.(error)
	if ok {
		return err
	}

	return fmt.Errorf("delete pod failed, response content type unsupported")
}

func (c *pods) Get(name string) (*corev1.Pod, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePod, name)
	podMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(podMsg)
	if err != nil {
		return nil, fmt.Errorf("get pod failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to pod failed, err: %v", err)
	}

	return handlePodFromMetaDB(name, content)
}

func (c *pods) Patch(name string, patchBytes []byte) (*corev1.Pod, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePodPatch, name)

	// FIXME: cleanup this code when the static pod mqtt broker no longer needs to be compatible
	if name == constants.DefaultMosquittoContainerName {
		return handleMqttMeta()
	}

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

	return handlePodResp(resource, content)
}

func handlePodFromMetaDB(name string, content []byte) (*corev1.Pod, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod list from db failed, err: %v", err)
	}

	if len(lists) == 0 {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "",
			Resource: "pod",
		}, name)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("pod length from meta db is %d", len(lists))
	}

	var pod *corev1.Pod
	err = json.Unmarshal([]byte(lists[0]), &pod)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod failed, err: %v", err)
	}
	return pod, nil
}

func handlePodResp(resource string, content []byte) (*corev1.Pod, error) {
	var podResp PodResp
	err := json.Unmarshal(content, &podResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod failed, err: %v", err)
	}

	if reflect.DeepEqual(podResp.Err, apierrors.StatusError{}) {
		if err = updatePodDB(resource, podResp.Object); err != nil {
			return nil, fmt.Errorf("update pod meta failed, err: %v", err)
		}
		return podResp.Object, nil
	}
	return podResp.Object, &podResp.Err
}

// updatePodDB update pod meta when patch pod successful
func updatePodDB(resource string, pod *corev1.Pod) error {
	pod.APIVersion = "v1"
	pod.Kind = "Pod"
	podContent, err := json.Marshal(pod)
	if err != nil {
		klog.Errorf("unmarshal resp pod failed, err: %v", err)
		return err
	}

	podKey := strings.Replace(resource,
		constants.ResourceSep+model.ResourceTypePodPatch+constants.ResourceSep,
		constants.ResourceSep+model.ResourceTypePod+constants.ResourceSep, 1)
	meta := &dao.Meta{
		Key:   podKey,
		Type:  model.ResourceTypePod,
		Value: string(podContent)}
	return dao.InsertOrUpdate(meta)
}

// FIXME: cleanup this code when the static pod mqtt broker no longer needs to be compatible
func handleMqttMeta() (*corev1.Pod, error) {
	var pod corev1.Pod
	metas, err := dao.QueryMeta("key", fmt.Sprintf("default/pod/%s", constants.DefaultMosquittoContainerName))
	if err != nil || len(*metas) != 1 {
		return nil, fmt.Errorf("get mqtt meta failed, err: %v", err)
	}

	err = json.Unmarshal([]byte((*metas)[0]), &pod)
	if err != nil {
		return nil, fmt.Errorf("unmarshal mqtt meta failed, err: %v", err)
	}
	return &pod, nil
}
