package imitator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
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
			if e.Object, removed, kept, err = s.deleteStored(e.Object); kept {
				continue
			}
			// The DELETED that removed a row goes to every watch, whatever resourceVersion it
			// carries: a DELETE that waited in cloudhub behind an un-ACKed update, or the stored
			// object swapped in for a DELETE without resourceVersion, is below a watch opened
			// after the deletion in the cloud, and the row it removed was in that watch's list.
			// Older watches see the RV go back, harmless while metaserver ignores the RV a client
			// resumes from (watcher.go forces 0). Any other DELETED is filtered by
			// resourceVersion, as before.
			checkRV = !removed
		}
		if err != nil {
			key := metaserver.KeyFunc(e.Object)
			klog.Errorf("failed to serve event {type:%v,key:%v}: %v", e.Type, key, err)
			continue
		}
		// TODO: move Trigger inside InsertOrUpdateObj and DeleteObj
		watchhook.Trigger(e, checkRV)
	}
}

// deleteStored deletes obj's row, unless it holds another object (UID) under the same key, in
// one step under s.lock. It reports whether it removed the row or kept another object's, and
// returns the object a DELETED event carries: the stored one when obj has no resourceVersion
// (the DELETE synccontroller.gcOrphanedObjectSync sends, with no labels or spec), else obj.
//
// A DELETE that names no UID deletes the row by key and goes out as before the UID check:
// with its own body, filtered by resourceVersion. edgecontroller sends one for a pod name
// whose pod is gone from the cloud or was recreated, and the row may hold the recreated pod,
// so watches must not be told that the stored object was deleted.
func (s *imitator) deleteStored(obj runtime.Object) (_ runtime.Object, removed, kept bool, err error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return obj, false, false, err
	}
	key, err := metaserver.KeyFuncObj(obj)
	if err != nil {
		return obj, false, false, err
	}
	uid := string(accessor.GetUID())
	s.lock.Lock()
	defer s.lock.Unlock()
	if uid == "" {
		return obj, false, false, s.deleteKey(key)
	}
	var stored runtime.Object
	if accessor.GetResourceVersion() == "" {
		stored = s.storedObj(key)
	}
	removed, kept, err = s.deleteKeyUID(key, uid)
	if err != nil {
		// The stored UID is unknown, so the DELETE goes out as it did before the UID check.
		klog.Errorf("[imitator] delete %s error: %v", key, err)
		return obj, false, false, nil
	}
	if !removed || stored == nil {
		return obj, removed, kept, nil
	}
	return stored, true, false, nil
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

	err := s.deleteKey(key)
	if err != nil {
		klog.Errorf("[imitator] delete error: %v", err)
	}
	return err
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
