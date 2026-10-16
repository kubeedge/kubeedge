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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
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
	// The key is computed under the lock: a mapping change, and the migration
	// of the objects it moves, happen under the same lock, see initRESTMapper.
	s.lock.Lock()
	defer s.lock.Unlock()

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
	return s.insertOrReplaceMetaV2Locked(m, objRv)
}

func (s *imitator) insertOrReplaceMetaV2(m models.MetaV2, objRv uint64) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.insertOrReplaceMetaV2Locked(m, objRv)
}

// insertOrReplaceMetaV2Locked must be called with s.lock held.
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

// beforeMigrate is called by migrateResourceLocked before it reads the objects
// to move. It lets tests interleave other operations with a migration.
var beforeMigrate = func() {}

// migrateResourceLocked moves the objects of gvk saved under oldGVR to newGVR.
// It is called when the authoritative resource of gvk becomes known, so that
// objects saved under a guessed resource, e.g. gatewaies for Gateway, are
// served again. It runs with s.lock held, as the change locker of the
// RESTMapper, so that no object is written or deleted between the mapping
// change and the move.
func (s *imitator) migrateResourceLocked(gvk schema.GroupVersionKind, oldGVR, newGVR schema.GroupVersionResource) {
	beforeMigrate()

	service := dbclient.NewMetaV2Service()
	results, err := service.RawMetaByGVRNN(oldGVR, models.NullNamespace, models.NullName)
	if err != nil {
		klog.Errorf("[metaserver] failed to query objects of %s to move them to %s: %v", oldGVR.String(), newGVR.String(), err)
		return
	}

	var oldKeys []string
	var moved []models.MetaV2
	for _, result := range *results {
		obj := new(unstructured.Unstructured)
		if err := runtime.DecodeInto(s.codec, []byte(result.Value), obj); err != nil {
			klog.Warningf("[metaserver] failed to decode object %s: %v", result.Key, err)
			continue
		}
		if obj.GroupVersionKind() != gvk {
			continue
		}
		key, err := metaserver.KeyFuncObj(obj)
		if err != nil || key == result.Key {
			continue
		}
		oldKeys = append(oldKeys, result.Key)
		result.Key = key
		result.GroupVersionResource = newGVR.String()
		moved = append(moved, result)
	}
	if len(moved) == 0 {
		return
	}
	if err := service.MoveMetaV2(oldKeys, moved); err != nil {
		klog.Errorf("[metaserver] failed to move objects of %s from %s to %s: %v", gvk.String(), oldGVR.String(), newGVR.String(), err)
		return
	}
	klog.Infof("[metaserver] moved %d objects of %s from %s to %s", len(moved), gvk.String(), oldGVR.String(), newGVR.String())
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
	// The key is computed under the lock, like in InsertOrUpdateObj, so that a
	// delete concurrent with a mapping change cannot miss the object and let
	// the migration move it to the new resource.
	s.lock.Lock()
	defer s.lock.Unlock()

	key, err := metaserver.KeyFuncObj(obj)
	if err != nil {
		return err
	}
	if err := dbclient.NewMetaV2Service().DeleteByKey(key); err != nil {
		klog.Errorf("[imitator] delete error: %v", err)
		return err
	}
	return nil
}

// listKindKeyPrefix is the key prefix of the list kinds saved in meta_v2. It is
// not an API path, so it never collides with a pass-through request.
const listKindKeyPrefix = "/kubeedge.io/metaserver/listkinds/"

// saveListKind saves the list kind of gvr, so that it is known when listing
// from the local storage after a restart, before the cloud returns a list of
// gvr again. Only list kinds that differ from <kind>List need to be saved.
func (s *imitator) saveListKind(mapper *util.RESTMapper) util.ListKindLearnedFunc {
	return func(gvr schema.GroupVersionResource, listKind string) {
		if listKind == mapper.KindFor(gvr).Kind+"List" {
			return
		}
		m := models.MetaV2{
			Key:   listKindKeyPrefix + gvr.Group + "/" + gvr.Version + "/" + gvr.Resource,
			Value: listKind,
		}
		if err := s.insertOrReplaceMetaV2(m, 0); err != nil {
			klog.Warningf("[metaserver] failed to save list kind %s of %s: %v", listKind, gvr.String(), err)
		}
	}
}

// loadListKinds teaches mapper the list kinds saved by saveListKind.
func loadListKinds(mapper *util.RESTMapper) int {
	saved, err := dbclient.NewMetaV2Service().ListMetaV2ByKeyPrefix(listKindKeyPrefix)
	if err != nil {
		klog.Errorf("failed to load saved list kinds: %v", err)
		return 0
	}
	loaded := 0
	for _, m := range *saved {
		parts := strings.SplitN(strings.TrimPrefix(m.Key, listKindKeyPrefix), "/", 3)
		if len(parts) != 3 {
			continue
		}
		gvr := schema.GroupVersionResource{Group: parts[0], Version: parts[1], Resource: parts[2]}
		if mapper.RegisterListKind(gvr, m.Value) {
			loaded++
		}
	}
	return loaded
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
