package imitator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	metaserverconfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/watchhook"
	patchutil "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/util"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
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

// Inject transform the message to watch.event, save internal obj/objs to table meta_v2
// and trigger the corresponding hook to serve watch
func (s *imitator) Inject(msg model.Message) {
	for _, e := range s.Event(&msg) {
		// save to meta_v2
		var err error
		checkRV := true
		switch e.Type {
		case watch.Added, watch.Modified:
			err = s.InsertOrUpdateObj(context.TODO(), e.Object)
		case watch.Deleted:
			var removed, kept bool
			var contents []runtime.Object
			if e.Object, removed, kept, contents, err = s.deleteStored(e.Object); kept {
				continue
			}
			triggerDeleted(contents)
			// A DELETED that removed a row goes to every watch, whatever its resourceVersion:
			// the row was in each watch's list, and the resourceVersion can be below a watch
			// opened after the deletion in the cloud (a DELETE that waited in cloudhub behind an
			// un-ACKed update, or the stored object swapped in for a DELETE without
			// resourceVersion). Older watches see the RV go back, harmless while metaserver
			// ignores the RV a client resumes from (watcher.go forces 0).
			//
			// Any other DELETED is filtered by resourceVersion, as before. One that found no row
			// still reaches the watches opened before its resourceVersion, because a client may
			// hold the object from a list the cloud served; a watcher without an informer then
			// sees a second DELETED. One without resourceVersion reaches no watch (it parses to
			// 0): there is no stored object to send, and its own body has no labels or spec.
			checkRV = !removed
		}
		if err != nil {
			key := metaserver.KeyFunc(e.Object)
			klog.Errorf("failed to serve event {type:%v,key:%v}: %v", e.Type, key, err)
			continue
		}
		// TODO: move Trigger inside InsertOrUpdateObj and deleteStored
		watchhook.Trigger(e, checkRV)
	}
}

// deleteStored deletes obj's row, unless it holds another object (UID) under the same key, in
// one step under s.lock. It reports whether it removed the row or kept another object's, and
// returns the object a DELETED event carries: the stored one when obj has no resourceVersion
// (the DELETE synccontroller.gcOrphanedObjectSync sends, with no labels or spec), else obj.
// If the stored object does not decode, for example when its podpatch fails to merge, the row
// is still removed and the DELETED carries obj, so the object does not stay in the clients.
//
// A DELETE that names no UID deletes the row by key and goes out as before the UID check:
// with its own body, filtered by resourceVersion. edgecontroller sends one for a pod name
// whose pod is gone from the cloud or was recreated, and the row may hold the recreated pod,
// so watches must not be told that the stored object was deleted.
//
// When the removed row is a Namespace's, the rows of its objects go in the same step, and
// deleteStored returns those objects as contents.
func (s *imitator) deleteStored(obj runtime.Object) (_ runtime.Object, removed, kept bool, contents []runtime.Object, err error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return obj, false, false, nil, err
	}
	key, err := metaserver.KeyFuncObj(obj)
	if err != nil {
		return obj, false, false, nil, err
	}
	uid := string(accessor.GetUID())
	s.lock.Lock()
	defer s.lock.Unlock()
	if uid == "" {
		return obj, false, false, nil, s.deleteKey(key)
	}
	var stored runtime.Object
	if accessor.GetResourceVersion() == "" {
		stored = s.storedObj(key)
	}
	removed, kept, err = s.deleteKeyUID(key, uid)
	if err != nil {
		// The stored UID is unknown, so the DELETE goes out as it did before the UID check.
		klog.Errorf("[imitator] delete %s error: %v", key, err)
		return obj, false, false, nil, nil
	}
	if !removed {
		return obj, false, kept, nil, nil
	}
	if stored == nil {
		stored = obj
	}
	return stored, true, false, s.deleteNamespaceContents(obj, stored), nil
}

// podGVR names pods. This node's pods are left out of deleteNamespaceContents: edged manages them.
var podGVR = schema.GroupVersionResource{Version: "v1", Resource: "pods"}.String()

// deleteNamespaceContents removes the rows of the objects in the deleted Namespace, and returns
// the objects so that watches can be told. The cloud loses those deletes when the namespace's
// only edge watcher ran in it, or the edge was offline, because their ObjectSyncs go with the
// namespace. body is the DELETE's own object and obj the one the DELETED event carries, which
// is the stored Namespace when body has no resourceVersion. The caller holds s.lock.
//
// The deletionTimestamp of body, else of obj, tells the rows of the deleted namespace from those
// of a namespace recreated under its name: a terminating namespace takes no new objects, so the
// recreated namespace's objects are all created after it. The edge can hold them before this
// DELETE arrives when it never got the recreated Namespace. synccontroller.gcOrphanedClusterObjectSync
// stamps a deletionTimestamp on every DELETE it resends. Without one (an older cloudcore),
// every row goes. Timestamps are whole seconds, so an object created in the same second as the
// cutoff can land on the wrong side; that needs delete, finalize and recreate within one second.
func (s *imitator) deleteNamespaceContents(body, obj runtime.Object) []runtime.Object {
	ns := namespaceName(body)
	if ns == "" {
		return nil
	}
	var deleted *metav1.Time
	for _, o := range []runtime.Object{body, obj} {
		if accessor, err := meta.Accessor(o); err == nil && accessor.GetDeletionTimestamp() != nil {
			deleted = accessor.GetDeletionTimestamp()
			break
		}
	}
	return s.deleteNamespaced(ns, deleted)
}

// deleteRecreatedNamespaceContents removes the rows of the objects in the namespace that obj, a
// recreated Namespace, replaces under key, and returns the objects so that watches can be told.
// The edge can get the recreated Namespace before the DELETE of the old one, or instead of it,
// and that DELETE then finds another UID and removes nothing. The objects created before obj
// belong to the old namespace: obj was created after the old namespace was gone, and timestamps
// are in whole seconds. The caller holds s.lock.
func (s *imitator) deleteRecreatedNamespaceContents(key string, obj *unstructured.Unstructured) []runtime.Object {
	ns := namespaceName(obj)
	created := obj.GetCreationTimestamp()
	if ns == "" || obj.GetUID() == "" || created.IsZero() {
		return nil
	}
	stored, err := meta.Accessor(s.storedObj(key))
	if err != nil || stored.GetUID() == obj.GetUID() {
		return nil
	}
	deleted := metav1.NewTime(created.Add(-time.Second))
	return s.deleteNamespaced(ns, &deleted)
}

// namespaceName returns the name of obj if it is a Namespace, else "".
func namespaceName(obj runtime.Object) string {
	if gvk := obj.GetObjectKind().GroupVersionKind(); gvk.Group != "" || gvk.Kind != "Namespace" {
		return ""
	}
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}
	return accessor.GetName()
}

// triggerDeleted sends the DELETEDs of the objects removed with their namespace to every watch.
func triggerDeleted(objs []runtime.Object) {
	for _, o := range objs {
		watchhook.Trigger(watch.Event{Type: watch.Deleted, Object: o}, false)
	}
}

// deleteNamespaced deletes the rows of the objects in namespace ns, except this node's pods and
// those created after deleted when it is set, and returns the deleted objects that decode, so
// watches can be told. A row that does not decode is deleted all the same, unless it is a pod,
// which may be this node's. The caller holds s.lock.
func (s *imitator) deleteNamespaced(ns string, deleted *metav1.Time) []runtime.Object {
	metaV2 := dbclient.NewMetaV2Service()
	rows, err := metaV2.RawMetaByGVRNN(schema.GroupVersionResource{}, ns, "")
	if err != nil {
		klog.Errorf("[imitator] list rows of deleted namespace %s: %v", ns, err)
		return nil
	}
	var keys []string
	objs := make([]runtime.Object, 0, len(*rows))
	for _, row := range *rows {
		obj := new(unstructured.Unstructured)
		if err := patchutil.DecodeStored(s.codec, row.Value, obj); err != nil {
			klog.Errorf("[imitator] decode %s of deleted namespace %s: %v", row.Key, ns, err)
			if row.GroupVersionResource != podGVR {
				keys = append(keys, row.Key)
			}
			continue
		}
		if row.GroupVersionResource == podGVR {
			if node, _, _ := unstructured.NestedString(obj.Object, "spec", "nodeName"); node == metaserverconfig.Config.NodeName {
				continue
			}
		}
		if deleted != nil && obj.GetCreationTimestamp().After(deleted.Time) {
			continue
		}
		keys = append(keys, row.Key)
		objs = append(objs, obj)
	}
	// One transaction, so that a failure leaves every row and sends no DELETED.
	if err := metaV2.DeleteByKeys(keys); err != nil {
		klog.Errorf("[imitator] delete rows of deleted namespace %s: %v", ns, err)
		return nil
	}
	if len(keys) > 0 {
		klog.Infof("[imitator] namespace %s deleted: removed %d rows left in it, kept %d",
			ns, len(keys), len(*rows)-len(keys))
	}
	return objs
}

// storedObj returns the object under key as Get serves it, or nil if there is none or it
// does not decode. The caller holds s.lock.
func (s *imitator) storedObj(key string) runtime.Object {
	rows, err := dbclient.NewMetaV2Service().RawMetaByGVRNN(metaserver.ParseKey(key))
	if err != nil || len(*rows) != 1 {
		return nil
	}
	obj := new(unstructured.Unstructured)
	if err := patchutil.DecodeStored(s.codec, (*rows)[0].Value, obj); err != nil {
		klog.Errorf("failed to decode stored obj %s: %v", key, err)
		return nil
	}
	return obj
}

// deleteKeyUID deletes the row under key unless its object has a UID other than uid, and
// reports whether it removed the row or kept another object's. It reads the UID from the row
// itself: a pod's podpatch can name an older pod's UID. The caller holds s.lock.
func (s *imitator) deleteKeyUID(key, uid string) (removed, kept bool, err error) {
	metaV2 := dbclient.NewMetaV2Service()
	n, err := metaV2.DeleteByKeyAndUID(key, uid)
	if err != nil || n > 0 {
		return n > 0, false, err
	}
	rows, err := metaV2.CountByKey(key)
	if err != nil {
		return false, false, err
	}
	if rows > 0 {
		klog.V(2).Infof("[metaserver] skip delete of %s: the stored object is not uid %s", key, uid)
	}
	return false, rows > 0, nil
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
	var contents []runtime.Object
	err = func() error {
		s.lock.Lock()
		defer s.lock.Unlock()
		contents = s.deleteRecreatedNamespaceContents(key, unstr)
		return s.insertOrReplaceMetaV2Locked(m, objRv)
	}()
	triggerDeleted(contents)
	return err
}

func (s *imitator) insertOrReplaceMetaV2(m models.MetaV2, objRv uint64) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.insertOrReplaceMetaV2Locked(m, objRv)
}

// insertOrReplaceMetaV2Locked is insertOrReplaceMetaV2 for a caller that holds s.lock.
func (s *imitator) insertOrReplaceMetaV2Locked(m models.MetaV2, objRv uint64) error {
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

// deleteKey deletes the row under key. The caller holds s.lock.
func (s *imitator) deleteKey(key string) error {
	return dbclient.NewMetaV2Service().DeleteByKey(key)
}

func (s *imitator) InsertOrUpdatePassThroughObj(_ context.Context, obj []byte, key string) error {
	m := models.MetaV2{
		Key:   key,
		Value: string(obj),
	}
	return s.insertOrReplaceMetaV2(m, 0)
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
	var bytes []byte
	var err error
	var body = msg.GetContent()
	// convert body to bytes
	switch body := body.(type) {
	case []byte:
		bytes = body
	default:
		bytes, err = json.Marshal(body)
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
	//TODO: support array List like []obj
	obj := new(unstructured.Unstructured)
	err = runtime.DecodeInto(s.codec, bytes, obj)
	if err != nil {
		klog.Errorf("failed to unmarshal message content to unstructured obj: %+v", err)
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

// Resource format: <namespace>/<restype>[/resid]
// return <reskey, restype, resid>
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
