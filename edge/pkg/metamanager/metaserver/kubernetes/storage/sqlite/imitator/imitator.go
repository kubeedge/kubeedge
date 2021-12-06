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
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/watchhook"
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
		switch e.Type {
		case watch.Added, watch.Modified:
			err = s.InsertOrUpdateObj(context.TODO(), e.Object)
			utilruntime.Must(err)
		case watch.Deleted:
			err = s.DeleteObj(context.TODO(), e.Object)
			utilruntime.Must(err)
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

//TODO: filter out insert or update req that the obj's rev is smaller than the stored
func (s *imitator) InsertOrUpdateObj(ctx context.Context, obj runtime.Object) error {
	key, err := metaserver.KeyFuncObj(obj)
	utilruntime.Must(err)
	gvr, ns, name := metaserver.ParseKey(key)
	unstr, isUnstr := obj.(*unstructured.Unstructured)
	if !isUnstr {
		return fmt.Errorf("obj is not unstructrued type")
	}
	buf := bytes.NewBuffer(nil)
	err = s.codec.Encode(unstr, buf)
	utilruntime.Must(err)
	objRv, err := s.versioner.ObjectResourceVersion(obj)
	m := v2.MetaV2{
		Key:                  key,
		GroupVersionResource: gvr.String(),
		Namespace:            ns,
		Name:                 name,
		ResourceVersion:      objRv,
		Value:                buf.String(),
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	_, err = dbm.DBAccess.Raw("INSERT OR REPLACE INTO meta_v2 (key, groupversionresource, namespace,name,resourceversion,value) VALUES (?,?,?,?,?,?)", m.Key, m.GroupVersionResource, m.Namespace, m.Name, m.ResourceVersion, m.Value).Exec()
	var maxRetryTimes = 3
	for i := 1; err != nil; i++ {
		klog.Errorf("failed to access database:%v", err)
		if i == maxRetryTimes {
			return fmt.Errorf("failed to access database after %v times try", i)
		}
		_, err = dbm.DBAccess.Raw("INSERT OR REPLACE INTO meta_v2 (key, groupversionresource, namespace,name,resourceversion,value) VALUES (?,?,?,?,?,?)", m.Key, m.GroupVersionResource, m.Namespace, m.Name, m.ResourceVersion, m.Value).Exec()
	}
	if objRv > s.GetRevision() {
		s.SetRevision(objRv)
	}
	klog.V(4).Infof("[metaserver]successfully insert or update obj:%v", key)
	return nil
}
func (s *imitator) Delete(ctx context.Context, key string) error {
	m := v2.MetaV2{
		Key: key,
	}
	s.lock.Lock()
	_, err := dbm.DBAccess.Delete(&m)
	if err != nil {
		klog.Errorf("[imitator] delete error: %v", err)
	}
	s.lock.Unlock()
	return nil
}
func (s *imitator) DeleteObj(ctx context.Context, obj runtime.Object) error {
	key, err := metaserver.KeyFuncObj(obj)
	utilruntime.Must(err)
	err = s.Delete(context.TODO(), key)
	utilruntime.Must(err)
	return nil
}
func (s *imitator) Get(ctx context.Context, key string) (Resp, error) {
	var resp Resp
	s.lock.RLock()
	results, err := v2.RawMetaByGVRNN(metaserver.ParseKey(key))
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
func (s *imitator) List(ctx context.Context, key string) (Resp, error) {
	gvr, ns, name := metaserver.ParseKey(key)
	//if name != NullName {
	//	return Resp{}, fmt.Errorf("dao client list must not have resource name")
	//}
	klog.Infof("%v,%v,%v", gvr, ns, name)
	var resp Resp
	s.lock.RLock()
	results, err := v2.RawMetaByGVRNN(gvr, ns, name)
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
	wh := watchhook.NewWatchHook(key, rev, receiver)
	go func() {
		<-ctx.Done()
		wh.Stop()
	}()
	return wch
}

// Event transform the message to watch.event
func (s *imitator) Event(msg *model.Message) []watch.Event {
	klog.V(4).Infof("[metaserver] get a message from metamanager: %+v", msg)
	_, resType, _ := parseResource(msg.Router.Resource)
	//skip nodestatus and podstatus
	if strings.Contains(resType, "status") {
		klog.V(4).Infof("skip status messages")
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
		utilruntime.Must(err)
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
	var ret []watch.Event
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
		utilruntime.Must(err)
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
