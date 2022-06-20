package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

//PodsGetter is interface to get pods
type PodsGetter interface {
	Pods(namespace string) PodsInterface
}

//PodsInterface is pod interface
type PodsInterface interface {
	Create(*api.Pod) (*api.Pod, error)
	Update(*api.Pod) error
	Patch(name string, patchBytes []byte) (*api.Pod, error)
	Delete(name string, options metav1.DeleteOptions) error
	Get(name string) (*api.Pod, error)
}

type pods struct {
	namespace string
	send      SendInterface
}

// PodResp represents pod response from the api-server
type PodResp struct {
	Object *api.Pod
	Err    apierrors.StatusError
}

func newPods(namespace string, s SendInterface) *pods {
	return &pods{
		send:      s,
		namespace: namespace,
	}
}

func (c *pods) Create(cm *api.Pod) (*api.Pod, error) {
	return nil, nil
}

func (c *pods) Update(cm *api.Pod) error {
	return nil
}

func (c *pods) Delete(name string, options metav1.DeleteOptions) error {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePod, name)
	podOpt, _ := json.Marshal(options)
	podDeleteMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.DeleteOperation, podOpt)
	msg, err := c.send.SendSync(podDeleteMsg)
	if err != nil {
		return err
	}

	content, ok := msg.Content.(string)
	if ok && content == constants.MessageSuccessfulContent {
		errDB := dao.DeleteMetaByKey(resource)
		if errDB != nil {
			klog.Errorf("delete pod meta failed, %s, err: %v", resource, errDB)
		}

		podPatchKey := strings.Replace(resource, constants.ResourceSep+model.ResourceTypePod+constants.ResourceSep, constants.ResourceSep+model.ResourceTypePodPatch+constants.ResourceSep, 1)
		errDB = dao.DeleteMetaByKey(podPatchKey)
		if errDB != nil {
			klog.Errorf("delete podpatch meta failed, %s, err: %v", resource, errDB)
		}
		return nil
	}

	return err
}

func (c *pods) Get(name string) (*api.Pod, error) {
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

	return handlePodFromMetaDB(content)
}

func (c *pods) Patch(name string, patchBytes []byte) (*api.Pod, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypePodPatch, name)
	podMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.PatchOperation, patchBytes)
	resp, err := c.send.SendSync(podMsg)
	if err != nil {
		return nil, fmt.Errorf("update pod failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to pod failed, err: %v", err)
	}

	return handlePodResp(content)
}

func handlePodFromMetaDB(content []byte) (*api.Pod, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("pod length from meta db is %d", len(lists))
	}

	var pod *api.Pod
	err = json.Unmarshal([]byte(lists[0]), &pod)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod failed, err: %v", err)
	}
	return pod, nil
}

func handlePodFromMetaManager(content []byte) (*api.Pod, error) {
	var pod *api.Pod
	err := json.Unmarshal(content, &pod)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod failed, err: %v", err)
	}
	return pod, nil
}

func handlePodResp(content []byte) (*api.Pod, error) {
	var podResp *PodResp
	err := json.Unmarshal(content, &podResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to pod failed, err: %v", err)
	}

	if reflect.DeepEqual(podResp.Err, apierrors.StatusError{}) {
		return podResp.Object, nil
	}

	return podResp.Object, &podResp.Err
}
