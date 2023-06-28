package client

import (
	"errors"
	"strings"
	"testing"

	coordinationV1 "k8s.io/api/coordination/v1"
	api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

const TestNamespace = "test"

var metaclient CoreInterface

// ObjectResp is the object that api-server response
type ObjectResp struct {
	Object metaV1.Object
	Err    apierrors.StatusError
}

var objecttest = map[string]struct {
	object metaV1.Object
	err    apierrors.StatusError
	want   string
}{
	"operateErr": {
		object: nil,
		err:    *apierrors.NewBadRequest("bad request"),
		want:   "bad request",
	},
	"operateNodeSuccess": {
		object: &api.Node{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "operateNodeSuccess",
				Namespace: TestNamespace,
			},
		},
		err:  apierrors.StatusError{},
		want: "operateNodeSuccess",
	},
	"operateLeaseSuccess": {
		object: &coordinationV1.Lease{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "operateLeaseSuccess",
				Namespace: TestNamespace,
			},
		},
		err:  apierrors.StatusError{},
		want: "operateLeaseSuccess",
	},
	"operatePodSuccess": {
		object: &api.Pod{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "operatePodSuccess",
				Namespace: TestNamespace,
			},
		},
		err:  apierrors.StatusError{},
		want: "operatePodSuccess",
	},
}

func init() {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	meta := &common.ModuleInfo{
		ModuleName: modules.MetaManagerModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(meta)

	runMeta()
}

func runMeta() {
	go func() {
		for {
			select {
			case <-beehiveContext.Done():
				klog.Warning("MetaManager main loop stop")
				return
			default:
			}
			msg, err := beehiveContext.Receive(modules.MetaManagerModuleName)
			if err != nil {
				klog.Errorf("get a message %+v: %v", msg, err)
				continue
			}
			klog.V(2).Infof("get a message %+v", msg)
			process(msg)
		}
	}()
}

// Resource format: <namespace>/<restype>[/resid]
// return <namespace, restype, resid>
func parseResource(resource string) (string, string, string) {
	tokens := strings.Split(resource, constants.ResourceSep)
	resType := ""
	resID := ""
	switch len(tokens) {
	case 2:
		resType = tokens[len(tokens)-1]
	case 3:
		resType = tokens[len(tokens)-2]
		resID = tokens[len(tokens)-1]
	default:
	}
	return resource, resType, resID
}

func process(msg model.Message) {
	_, resType, resID := parseResource(msg.GetResource())
	if resID == "getRespContentErr" {
		resp := model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody(make(chan int))
		beehiveContext.SendResp(*resp)
		return
	}
	switch resType {
	case model.ResourceTypeNode, model.ResourceTypeNodePatch, model.ResourceTypeLease:
		processNode(resID, msg)
	case model.ResourceTypePod, model.ResourceTypePodPatch:
		processPod(resID, msg)
	case model.ResourceTypeConfigmap, model.ResourceTypeSecret:
		processQueryResource(resID, msg)
	}

}

func processNode(resID string, msg model.Message) {
	objectResp := ObjectResp{
		Object: objecttest[resID].object,
		Err:    objecttest[resID].err,
	}
	resp := model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody(&objectResp)
	beehiveContext.SendResp(*resp)
}

func processPod(resID string, msg model.Message) {
	var resp *model.Message
	if msg.GetOperation() == model.DeleteOperation {
		if resID == "deletePodErr" {
			resp = model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody(errors.New("deletePodErr"))
		} else {
			resp = model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody(constants.MessageSuccessfulContent)
		}
		beehiveContext.SendResp(*resp)
		return
	}
	objectResp := ObjectResp{
		Object: objecttest[resID].object,
		Err:    objecttest[resID].err,
	}
	resp = model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody(&objectResp)
	beehiveContext.SendResp(*resp)
}

func processQueryResource(resID string, msg model.Message) {
	var resp *model.Message
	if resID == "getResourceErr" {
		resp = model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody(errors.New("getResourceErr"))
		beehiveContext.SendResp(*resp)
	}

	if resID == "unmarshalContentToStrListErr" {
		resp = model.NewMessage(msg.GetID()).BuildRouter(modules.MetaManagerModuleName, modules.MetaGroup, msg.GetResource(), model.ResponseOperation).FillBody("unmarshalContentToStrListErr")
		beehiveContext.SendResp(*resp)
	}

	if resID == "strListExtentLengthErr" {
		respStr := []string{"test1", "test2"}
		resp = model.NewMessage(msg.GetID()).BuildRouter(modules.MetaManagerModuleName, modules.MetaGroup, msg.GetResource(), model.ResponseOperation).FillBody(respStr)
		beehiveContext.SendResp(*resp)
	}

	if resID == "unmarshalStrListToResourceErr" {
		respStr := []string{"test1"}
		resp = model.NewMessage(msg.GetID()).BuildRouter(modules.MetaManagerModuleName, modules.MetaGroup, msg.GetResource(), model.ResponseOperation).FillBody(respStr)
		beehiveContext.SendResp(*resp)
	}

	if resID == "unmarshalContentToResourceErr" {
		resp = model.NewMessage(msg.GetID()).BuildRouter(modules.EdgedModuleName, modules.MetaGroup, msg.GetResource(), msg.GetOperation()).FillBody("unmarshalContentToResourceErr")
		beehiveContext.SendResp(*resp)
	}
}

func TestMetaClient(t *testing.T) {
	metaclient = New()

	edged := &common.ModuleInfo{
		ModuleName: modules.EdgedModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.AddModule(edged)
}

func TestNode(t *testing.T) {
	nodeReq := api.Node{}

	// GetCreateNodeContentError
	nodeReq.Name = "getRespContentErr"
	_, err := metaclient.Nodes(TestNamespace).Create(&nodeReq)
	t.Run("GetCreateNodeContentError", func(t *testing.T) {
		want := "parse message to node failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// CreateNodeError
	nodeReq.Name = "operateErr"
	_, err = metaclient.Nodes(TestNamespace).Create(&nodeReq)
	t.Run("CreateNodeError", func(t *testing.T) {
		want := objecttest["operateErr"].want
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// CreateNodeSuccess
	nodeReq.Name = "operateNodeSuccess"
	resp, _ := metaclient.Nodes(TestNamespace).Create(&nodeReq)
	t.Run("CreateNodeSuccess", func(t *testing.T) {
		want := objecttest["operateNodeSuccess"].want
		if resp.Name != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})

	// GetPatchNodeContentError
	nodeReq.Name = "getRespContentErr"
	_, err = metaclient.Nodes(TestNamespace).Patch("getRespContentErr", nil)
	t.Run("GetPatchNodeContentError", func(t *testing.T) {
		want := "parse message to node failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// PatchNodeError
	_, err = metaclient.Nodes(TestNamespace).Patch("operateErr", nil)
	t.Run("PatchNodeError", func(t *testing.T) {
		want := objecttest["operateErr"].want
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// PatchNodeSuccess
	resp, _ = metaclient.Nodes(TestNamespace).Patch("operateNodeSuccess", nil)
	t.Run("PatchNodeSuccess", func(t *testing.T) {
		want := objecttest["operateNodeSuccess"].want
		if resp.Name != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})

	// GetQueryNodeContentError
	nodeReq.Name = "getRespContentErr"
	_, err = metaclient.Nodes(TestNamespace).Create(&nodeReq)
	t.Run("GetQueryNodeContentError", func(t *testing.T) {
		want := "parse message to node failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// TestDeleteNode
	err = metaclient.Nodes(TestNamespace).Delete("deleteNode")
	t.Run("TestDeleteNode", func(t *testing.T) {
		if err != nil {
			t.Errorf("Wrong Error message received : want nil and Got %v", err)
		}
	})
}

func TestLease(t *testing.T) {
	leaseReq := coordinationV1.Lease{}

	// GetCreateLeaseContentError
	leaseReq.Name = "getRespContentErr"
	_, err := metaclient.Leases(TestNamespace).Create(&leaseReq)
	t.Run("GetCreateLeaseContentError", func(t *testing.T) {
		want := "parse message to lease failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// CreateLeaseError
	leaseReq.Name = "operateErr"
	_, err = metaclient.Leases(TestNamespace).Create(&leaseReq)
	t.Run("CreateLeaseError", func(t *testing.T) {
		want := objecttest["operateErr"].want
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// CreateLeaseSuccess
	leaseReq.Name = "operateLeaseSuccess"
	resp, _ := metaclient.Leases(TestNamespace).Create(&leaseReq)
	t.Run("CreateLeaseSuccess", func(t *testing.T) {
		want := objecttest["operateLeaseSuccess"].want
		if resp.Name != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})

	// GetUpdateLeaseContentError
	leaseReq.Name = "getRespContentErr"
	_, err = metaclient.Leases(TestNamespace).Update(&leaseReq)
	t.Run("GetUpdateLeaseContentError", func(t *testing.T) {
		want := "parse message to lease failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UpdateLeaseError
	leaseReq.Name = "operateErr"
	_, err = metaclient.Leases(TestNamespace).Update(&leaseReq)
	t.Run("UpdateLeaseError", func(t *testing.T) {
		want := objecttest["operateErr"].want
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UpdateNodeSuccess
	leaseReq.Name = "operateLeaseSuccess"
	resp, _ = metaclient.Leases(TestNamespace).Update(&leaseReq)
	t.Run("UpdateNodeSuccess", func(t *testing.T) {
		want := objecttest["operateLeaseSuccess"].want
		if resp.Name != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})

	// GetQueryLeaseContentError
	leaseReq.Name = "getRespContentErr"
	_, err = metaclient.Leases(TestNamespace).Create(&leaseReq)
	t.Run("GetQueryLeaseContentError", func(t *testing.T) {
		want := "parse message to lease failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// QueryLeaseError
	_, err = metaclient.Leases(TestNamespace).Get("operateErr")
	t.Run("QueryLeaseError", func(t *testing.T) {
		want := objecttest["operateErr"].want
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// QueryLeaseSuccess
	resp, _ = metaclient.Leases(TestNamespace).Get("operateLeaseSuccess")
	t.Run("QueryLeaseSuccess", func(t *testing.T) {
		want := objecttest["operateLeaseSuccess"].want
		if resp.Name != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})
}

func TestPod(t *testing.T) {
	// GetPatchPodContentError
	_, err := metaclient.Pods(TestNamespace).Patch("getRespContentErr", nil)
	t.Run("GetPatchPodContentError", func(t *testing.T) {
		want := "parse message to pod failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// PatchPodError
	_, err = metaclient.Pods(TestNamespace).Patch("operateErr", nil)
	t.Run("PatchPodError", func(t *testing.T) {
		want := objecttest["operateErr"].want
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// PatchPodSuccess
	resp, _ := metaclient.Pods(TestNamespace).Patch("operatePodSuccess", nil)
	t.Run("PatchPodSuccess", func(t *testing.T) {
		want := objecttest["operatePodSuccess"].want
		if resp.Name != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})

	// DeletePodErr
	err = metaclient.Pods(TestNamespace).Delete("deletePodErr", metaV1.DeleteOptions{})
	t.Run("DeletePodErr", func(t *testing.T) {
		want := "deletePodErr"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, resp)
		}
	})

	// DeletePodSuccess
	err = metaclient.Pods(TestNamespace).Delete("deletePodSuccess", metaV1.DeleteOptions{})
	t.Run("DeletePodSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("Wrong Error message received: want nil and Got %v", err)
		}
	})

	// GetQueryPodContentError
	_, err = metaclient.Pods(TestNamespace).Get("getRespContentErr")
	t.Run("GetQueryPodContentError", func(t *testing.T) {
		want := "parse message to pod failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// TestCreatePod
	_, err = metaclient.Pods(TestNamespace).Create(&api.Pod{})
	t.Run("TestCreatePod", func(t *testing.T) {
		if err != nil {
			t.Errorf("Wrong Error message received : want nil and Got %v", err)
		}
	})

	// TestUpdatePod
	err = metaclient.Pods(TestNamespace).Update(&api.Pod{})
	t.Run("TestUpdatePod", func(t *testing.T) {
		if err != nil {
			t.Errorf("Wrong Error message received : want nil and Got %v", err)
		}
	})
}

func TestConfigMap(t *testing.T) {
	// GetQueryConfigMapContentError
	_, err := metaclient.ConfigMaps(TestNamespace).Get("getRespContentErr")
	t.Run("GetQueryConfigMapContentError", func(t *testing.T) {
		want := "parse message to configmap failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// GetConfigMapErr
	_, err = metaclient.ConfigMaps(TestNamespace).Get("getResourceErr")
	t.Run("GetConfigMapErr", func(t *testing.T) {
		want := "getResourceErr"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UnmarshalContentToStrListErr
	_, err = metaclient.ConfigMaps(TestNamespace).Get("unmarshalContentToStrListErr")
	t.Run("UnmarshalContentToStrListErr", func(t *testing.T) {
		want := "unmarshal message to ConfigMap list from db failed, err: invalid character 'u' looking for beginning of value"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// strListExtentLengthErr
	_, err = metaclient.ConfigMaps(TestNamespace).Get("strListExtentLengthErr")
	t.Run("StrListExtentLengthErr", func(t *testing.T) {
		want := "ConfigMap length from meta db is 2"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UnmarshalStrListToResourceErr
	_, err = metaclient.ConfigMaps(TestNamespace).Get("unmarshalStrListToResourceErr")
	t.Run("UnmarshalStrListToResourceErr", func(t *testing.T) {
		want := "unmarshal message to ConfigMap from db failed, err: invalid character 'e' in literal true (expecting 'r')"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UnmarshalContentToResourceErr
	_, err = metaclient.ConfigMaps(TestNamespace).Get("unmarshalContentToResourceErr")
	t.Run("UnmarshalContentToResourceErr", func(t *testing.T) {
		want := "unmarshal message to ConfigMap failed, err: invalid character 'u' looking for beginning of value"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})
}

func TestSecret(t *testing.T) {
	// GetQuerySecretContentError
	_, err := metaclient.Secrets(TestNamespace).Get("getRespContentErr")
	t.Run("GetQuerySecretContentError", func(t *testing.T) {
		want := "parse message to secret failed, err: marshal message content failed: json: unsupported type: chan int"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// GetSecretErr
	_, err = metaclient.Secrets(TestNamespace).Get("getResourceErr")
	t.Run("GetSecretErr", func(t *testing.T) {
		want := "getResourceErr"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UnmarshalContentToStrListErr
	_, err = metaclient.Secrets(TestNamespace).Get("unmarshalContentToStrListErr")
	t.Run("UnmarshalContentToStrListErr", func(t *testing.T) {
		want := "unmarshal message to secret list from db failed, err: invalid character 'u' looking for beginning of value"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// strListExtentLengthErr
	_, err = metaclient.Secrets(TestNamespace).Get("strListExtentLengthErr")
	t.Run("StrListExtentLengthErr", func(t *testing.T) {
		want := "secret length from meta db is 2"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UnmarshalStrListToResourceErr
	_, err = metaclient.Secrets(TestNamespace).Get("unmarshalStrListToResourceErr")
	t.Run("UnmarshalStrListToResourceErr", func(t *testing.T) {
		want := "unmarshal message to secret from db failed, err: invalid character 'e' in literal true (expecting 'r')"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})

	// UnmarshalContentToResourceErr
	_, err = metaclient.Secrets(TestNamespace).Get("unmarshalContentToResourceErr")
	t.Run("UnmarshalContentToResourceErr", func(t *testing.T) {
		want := "unmarshal message to secret failed, err: invalid character 'u' looking for beginning of value"
		if err.Error() != want {
			t.Errorf("Wrong Error message received : want %v and Got %v", want, err.Error())
		}
	})
}
