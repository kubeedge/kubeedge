package imitator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/watchhook"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// imitator is a storage based on metav2 that imitate the behavior of etcd
type imitator struct {
	lock sync.RWMutex
	// The Revision is the current revision of client
	// It is set when client inits or a bigger resourceversion obj was saved into meta_v2
	revision uint64
	// to parse obj resource version from string to int64
	versioner storage.Versioner
	// to co/decoder obj
	codec runtime.Codec
}

const (
	resourceTypeDevice = "device"
)

// Inject transform the message to watch.event, save internal obj/objs to table meta_v2
// and trigger the corresponding hook to serve watch
func (s *imitator) Inject(msg model.Message) {
	for _, e := range s.Event(&msg) {
		// save to meta_v2
		var err error
		switch e.Type {
		case watch.Added, watch.Modified:
			err = s.InsertOrUpdateObj(context.TODO(), e.Object)
		case watch.Deleted:
			err = s.DeleteObj(context.TODO(), e.Object)
		}
		if err != nil {
			key := metaserver.KeyFunc(e.Object)
			klog.Errorf("failed to serve event {type:%v,key:%v}", e.Type, key)
			continue
		}
		// TODO: move Trigger inside InsertOrUpdateObj and DeleteObj
		watchhook.Trigger(e)
	}
}

// TODO: filter out insert or update req that the obj's rev is smaller than the stored
func (s *imitator) InsertOrUpdateObj(_ context.Context, obj runtime.Object) error {
	key, err := metaserver.KeyFuncObj(obj)
	if err != nil {
		return err
	}
	gvr, ns, name := metaserver.ParseKey(key)
	unstr, isUnstr := obj.(*unstructured.Unstructured)
	if !isUnstr {
		return fmt.Errorf("obj is not unstructured type")
	}
	buf := bytes.NewBuffer(nil)
	err = s.codec.Encode(unstr, buf)
	if err != nil {
		return err
	}
	objRv, err := s.versioner.ObjectResourceVersion(obj)
	m := models.MetaV2{
		Key:                  key,
		GroupVersionResource: gvr.String(),
		Namespace:            ns,
		Name:                 name,
		ResourceVersion:      objRv,
		Value:                buf.String(),
	}
	return s.insertOrReplaceMetaV2(m, objRv)
}

func (s *imitator) insertOrReplaceMetaV2(m models.MetaV2, objRv uint64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := dbclient.NewMetaV2Service().RetryInsertOrReplaceMetaV2(&m, 3)
	if err != nil {
		klog.Errorf("failed to access database after retries: %v", err)
		return err
	}

	if objRv > s.GetRevision() {
		s.SetRevision(objRv)
	}
	klog.V(4).Infof("[metaserver]successfully insert or update obj:%v", m.Key)
	return nil
}

func (s *imitator) GetPassThroughObj(_ context.Context, key string) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	result, err := dbclient.NewMetaV2Service().GetByKey(key)
	if err != nil {
		return nil, err
	}
	klog.V(4).Infof("[metaserver]successfully queried obj:%v", key)
	return []byte(result.Value), nil
}

func (s *imitator) Delete(_ context.Context, key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := dbclient.NewMetaV2Service().DeleteByKey(key)
	if err != nil {
		klog.Errorf("[imitator] delete error: %v", err)
	}
	return err
}

func (s *imitator) InsertOrUpdatePassThroughObj(_ context.Context, obj []byte, key string) error {
	m := models.MetaV2{
		Key:   key,
		Value: string(obj),
	}
	return s.insertOrReplaceMetaV2(m, 0)
}

func (s *imitator) DeleteObj(_ context.Context, obj runtime.Object) error {
	key, err := metaserver.KeyFuncObj(obj)
	if err != nil {
		return err
	}
	err = s.Delete(context.TODO(), key)
	return err
}

func (s *imitator) Get(_ context.Context, key string) (Resp, error) {
	var resp Resp
	s.lock.RLock()
	results, err := dbclient.NewMetaV2Service().RawMetaByGVRNN(metaserver.ParseKey(key))
	resp.Revision = s.revision
	s.lock.RUnlock()
	if err != nil {
		return Resp{}, err
	}
	switch {
	case len(*results) == 1:
		resp.Kvs = results
		return resp, nil
	default:
		return Resp{}, fmt.Errorf("the server could not find the requested resource")
	}
}
func (s *imitator) List(_ context.Context, key string) (Resp, error) {
	gvr, ns, name := metaserver.ParseKey(key)
	//if name != NullName {
	//	return Resp{}, fmt.Errorf("dao client list must not have resource name")
	//}
	klog.Infof("%v,%v,%v", gvr, ns, name)
	var resp Resp
	s.lock.RLock()
	results, err := dbclient.NewMetaV2Service().RawMetaByGVRNN(gvr, ns, name)
	resp.Revision = s.revision

	s.lock.RUnlock()
	if err != nil {
		return Resp{}, err
	}
	resp.Kvs = results
	return resp, nil
}

func (s *imitator) GetRevision() uint64 {
	return s.revision
}

func (s *imitator) SetRevision(version interface{}) {
	switch v := version.(type) {
	case int64:
		s.revision = uint64(v)
	case uint64:
		s.revision = v
	case string:
		rv, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			klog.Error(err)
			return
		}
		s.revision = rv
	default:
		klog.Error("unsupported type when parse version")
	}
}

func (s *imitator) Watch(ctx context.Context, key string, rev uint64) <-chan watch.Event {
	wch := make(chan watch.Event)
	receiver := watchhook.NewChanReceiver(wch)
	wh, err := watchhook.NewWatchHook(key, rev, receiver)
	if err != nil {
		klog.Errorf("add hook for %s failed, %v", key, err)
		return nil
	}

	go func() {
		<-ctx.Done()
		wh.Stop()
		close(wch)
	}()
	return wch
}

// Event transform the message to watch.event
func (s *imitator) Event(msg *model.Message) []watch.Event {
	klog.V(4).Infof("[metaserver] get a message from metamanager: %+v", msg)
	var ret []watch.Event
	_, resType, _ := parseResource(msg.Router.Resource)
	//skip nodestatus, podstatus and node-lease
	if strings.Contains(resType, "status") || (strings.Contains(resType, "lease") && msg.GetSource() == modules.EdgedModuleName) {
		klog.V(4).Infof("skip status or node-lease messages")
		return []watch.Event{}
	}
	var contentBytes []byte
	var err error
	var body = msg.GetContent()
	// convert body to bytes
	switch body := body.(type) {
	case []byte:
		contentBytes = body
	default:
		contentBytes, err = json.Marshal(body)
		if err != nil {
			klog.Errorf("failed to marshal msg content, err: %+v", err)
			return ret
		}
	}
	var op watch.EventType
	switch msg.Router.Operation {
	case model.InsertOperation:
		op = watch.Added
	case model.UpdateOperation:
		op = watch.Modified
	case model.DeleteOperation:
		op = watch.Deleted
	}
	if op == watch.Deleted && isDeleteOptionsPayload(contentBytes) {
		klog.V(4).Infof("[metaserver]skip delete options payload for resource %q", msg.Router.Resource)
		return ret
	}
	if msg.Router.Group == "resource" {
		contentBytes = patchMissingKindForJSONObject(contentBytes, resType)
	}

	//TODO: support array List like []obj
	obj := new(unstructured.Unstructured)
	err = runtime.DecodeInto(s.codec, contentBytes, obj)
	if err != nil && isMissingTypeMetaError(err) {
		patched := patchMissingKindForJSONObject(contentBytes, resType)
		if !bytes.Equal(contentBytes, patched) {
			if retryErr := runtime.DecodeInto(s.codec, patched, obj); retryErr == nil {
				err = nil
			} else {
				err = retryErr
			}
		}
	}
	if err != nil {
		klog.Errorf("failed to unmarshal message content to unstructured obj: resource=%q group=%q operation=%q err=%+v", msg.Router.Resource, msg.Router.Group, msg.Router.Operation, err)
		return ret
	}
	if obj.IsList() {
		fn := func(object runtime.Object) error {
			event := watch.Event{
				Type:   op,
				Object: object,
			}
			ret = append(ret, event)
			return nil
		}
		err := obj.EachListItem(fn)
		if err != nil {
			klog.Errorf("failed to get ret list, err: %+v", err)
			return ret
		}
	} else {
		ret = append(ret, watch.Event{Type: op, Object: obj})
	}
	return ret
}

func patchMissingKindForJSONObject(raw []byte, resourceType string) []byte {
	if len(raw) == 0 || resourceType == "" {
		return raw
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return raw
	}

	obj := make(map[string]interface{})
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return raw
	}

	if !looksLikeKubeObject(obj) {
		return raw
	}

	kind, _ := obj["kind"].(string)
	if kind == "" {
		kind = inferKindFromResourceObject(obj, resourceType)
		if kind == "" {
			return raw
		}
		obj["kind"] = kind
	}

	if _, ok := obj["apiVersion"].(string); !ok || obj["apiVersion"] == "" {
		if apiVersion := inferAPIVersion(obj, resourceType, kind); apiVersion != "" {
			obj["apiVersion"] = apiVersion
		}
	}

	patched, err := json.Marshal(obj)
	if err != nil {
		return raw
	}

	klog.V(4).Infof("[metaserver]patched typemeta for resource %q to kind=%q apiVersion=%q", resourceType, kind, obj["apiVersion"])
	return patched
}

func isMissingTypeMetaError(err error) bool {
	if err == nil {
		return false
	}
	errText := err.Error()
	return strings.Contains(errText, "Object 'Kind' is missing") || strings.Contains(errText, "Object 'apiVersion' is missing")
}

func looksLikeKubeObject(obj map[string]interface{}) bool {
	if len(obj) == 0 {
		return false
	}
	if _, ok := obj["metadata"]; ok {
		return true
	}
	if _, ok := obj["spec"]; ok {
		return true
	}
	if _, ok := obj["status"]; ok {
		return true
	}
	if _, ok := obj["items"]; ok {
		return true
	}
	return isDeleteOptionsObject(obj)
}

func inferKindFromResourceObject(obj map[string]interface{}, resourceType string) string {
	if isDeleteOptionsObject(obj) {
		return "DeleteOptions"
	}
	return util.UnsafeResourceToKind(strings.ToLower(resourceType))
}

func inferAPIVersion(obj map[string]interface{}, resourceType, kind string) string {
	if apiVersion := apiVersionFromManagedFields(obj); apiVersion != "" {
		return apiVersion
	}

	resType := strings.ToLower(resourceType)
	switch resType {
	case "node", "nodes",
		"nodestatus", "nodepatch",
		"pod", "pods",
		"podstatus", "podpatch",
		"service", "services",
		"namespace", "namespaces",
		"configmap", "configmaps",
		"secret", "secrets",
		"event", "events",
		"endpoints":
		return "v1"
	case "endpointslice", "endpointslices":
		return "discovery.k8s.io/v1"
	case "lease", "leases":
		return "coordination.k8s.io/v1"
	case "csr", "certificatesigningrequest", "certificatesigningrequests":
		return "certificates.k8s.io/v1"
	case resourceTypeDevice, "devices", "devicemodel", "devicemodels", "devicestatus", "devicestatuses":
		return "devices.kubeedge.io/v1beta1"
	case "objectsync", "objectsyncs", "clusterobjectsync", "clusterobjectsyncs":
		return "reliablesyncs.kubeedge.io/v1alpha1"
	}

	if strings.EqualFold(kind, "DeleteOptions") {
		return "v1"
	}
	return ""
}

func apiVersionFromManagedFields(obj map[string]interface{}) string {
	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		return ""
	}
	managedFields, ok := metadata["managedFields"].([]interface{})
	if !ok {
		return ""
	}
	for _, field := range managedFields {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}
		if apiVersion, ok := fieldMap["apiVersion"].(string); ok && apiVersion != "" {
			return apiVersion
		}
	}
	return ""
}

func isDeleteOptionsObject(obj map[string]interface{}) bool {
	_, hasGracePeriod := obj["gracePeriodSeconds"]
	_, hasPreconditions := obj["preconditions"]
	if !hasGracePeriod && !hasPreconditions {
		return false
	}
	_, hasMetadata := obj["metadata"]
	return !hasMetadata
}

func isDeleteOptionsPayload(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}

	obj := make(map[string]interface{})
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return false
	}
	return isDeleteOptionsObject(obj)
}

// Resource format: <namespace>/<restype>[/resid]
// return <reskey, restype, resid>
func parseResource(resource string) (string, string, string) {
	resType, resID := util.ParseResourcePath(resource)
	return resource, resType, resID
}
