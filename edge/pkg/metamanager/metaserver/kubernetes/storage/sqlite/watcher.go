package sqlite

import (
	"context"
	"errors"
	"fmt"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"
	"strings"
	"sync"
)

const (
	// We have set a buffer in order to reduce times of context switches.
	incomingBufSize = 100
	outgoingBufSize = 100
)

// errTestingDecode is the only error that testingDeferOnDecodeError catches during a panic
var errTestingDecode = errors.New("sentinel error only used during testing to indicate watch decoding error")

// testingDeferOnDecodeError is used during testing to recover from a panic caused by errTestingDecode, all other values continue to panic
func testingDeferOnDecodeError() {
	if r := recover(); r != nil && r != errTestingDecode {
		panic(r)
	}
}

type watcher struct {
	client imitator.Client
	codec  runtime.Codec
}

// watchChan implements watch.Interface.
type watchChan struct {
	watcher           *watcher
	key               string
	initialRev        int64
	recursive         bool
	internalPred      storage.SelectionPredicate
	ctx               context.Context
	cancel            context.CancelFunc
	incomingEventChan chan *watch.Event
	// added is map show an obj whether it has been added to watch chan befor
	added      map[string]bool
	resultChan chan watch.Event
	errChan    chan error
}

func NewWatcher(client imitator.Client, codec runtime.Codec) *watcher {
	return &watcher{
		client: client,
		codec:  codec,
	}
}

// To implement Receiver
func (wc *watchChan) Receive(e watch.Event) error {
	wc.sendEvent(&e)
	return nil
}

// Watch watches on a key and returns a watch.Interface that transfers relevant notifications.
// If rev is zero, it will return the existing object(s) and then start watching from
// the maximum revision+1 from returned objects.
// If rev is non-zero, it will watch events happened after given revision.
// If recursive is false, it watches on given key.
// If recursive is true, it watches any children and directories under the key, excluding the root key itself.
// pred must be non-nil. Only if pred matches the change, it will be returned.
func (w *watcher) Watch(ctx context.Context, key string, rev int64, recursive bool, pred storage.SelectionPredicate) (watch.Interface, error) {
	// TODO: support rev != 0
	if rev != 0 {
		klog.Warning("base storage now only support rev == 0, but get rev == %v, force set to 0!", rev)
		rev = 0
	}
	wc := w.createWatchChan(ctx, key, rev, recursive, pred)
	go wc.run()
	return wc, nil
}

func (w *watcher) createWatchChan(ctx context.Context, key string, rev int64, recursive bool, pred storage.SelectionPredicate) *watchChan {
	wc := &watchChan{
		watcher:           w,
		key:               key,
		initialRev:        rev,
		recursive:         recursive,
		internalPred:      pred,
		incomingEventChan: make(chan *watch.Event, incomingBufSize),
		resultChan:        make(chan watch.Event, outgoingBufSize),
		errChan:           make(chan error, 1),
		added:             make(map[string]bool),
	}
	if pred.Empty() {
		// The filter doesn't filter out any object.
		wc.internalPred = storage.Everything
	}
	wc.ctx, wc.cancel = context.WithCancel(ctx)
	return wc
}
func (wc *watchChan) run() {
	watchClosedCh := make(chan struct{})
	go wc.startWatching(watchClosedCh)

	var resultChanWG sync.WaitGroup
	resultChanWG.Add(1)
	go wc.processEvent(&resultChanWG)

	select {
	case err := <-wc.errChan:
		errResult := transformErrorToEvent(err)
		if errResult != nil {
			// error result is guaranteed to be received by user before closing ResultChan.
			select {
			case wc.resultChan <- *errResult:
			case <-wc.ctx.Done(): // user has given up all results
			}
		}
	case <-watchClosedCh:
	case <-wc.ctx.Done(): // user cancel
	}

	// We use wc.ctx to reap all goroutines. Under whatever condition, we should stop them all.
	// It's fine to double cancel.
	wc.cancel()

	// we need to wait until resultChan wouldn't be used anymore
	resultChanWG.Wait()
	close(wc.resultChan)
}

func (wc *watchChan) Stop() {
	wc.cancel()
}

func (wc *watchChan) ResultChan() <-chan watch.Event {
	return wc.resultChan
}

// sync tries to retrieve existing data and send them to process.
// The revision to watch will be set to the revision in response.
// All events sent will have isCreated=true
func (wc *watchChan) sync() error {
	switch wc.recursive {
	case true: /*list*/
		resp, err := wc.watcher.client.List(context.TODO(), wc.key)
		if err != nil {
			return err
		}
		for _, kv := range *resp.Kvs {
			wc.sendEvent(wc.parseMeta(&kv))
		}
		wc.initialRev = int64(resp.Revision)
	case false: /*get*/
		resp, err := wc.watcher.client.List(context.TODO(), wc.key)
		if err != nil {
			return err
		}
		if len(*resp.Kvs) > 1 {
			klog.Warningf("get %v obj in key %v", len(*resp.Kvs), wc.key)
		}
		for _, kv := range *resp.Kvs {
			wc.sendEvent(wc.parseMeta(&kv))
		}
		wc.initialRev = int64(resp.Revision)
	}
	klog.Infof("get storage revision:%v", wc.initialRev)
	return nil
}

// parseMeta converts meta data to watch.Event
// and is only called in sync()
func (wc *watchChan) parseMeta(kv *v2.MetaV2) *watch.Event {
	obj, err := runtime.Decode(wc.watcher.codec, []byte(kv.Value))
	utilruntime.Must(err)
	return &watch.Event{
		Type:   watch.Added,
		Object: obj,
	}
}

// logWatchChannelErr checks whether the error is about mvcc revision compaction which is regarded as warning
func logWatchChannelErr(err error) {
	if !strings.Contains(err.Error(), "mvcc: required revision has been compacted") {
		klog.Errorf("watch chan error: %v", err)
	} else {
		klog.Warningf("watch chan error: %v", err)
	}
}

// startWatching does:
// - get current objects if initialRev=0; set initialRev to current rev
// - watch on given key and send events to process.
func (wc *watchChan) startWatching(watchClosedCh chan struct{}) {
	klog.Infof("start watching, rev:%v", wc.initialRev)
	if wc.initialRev == 0 {
		if err := wc.sync(); err != nil {
			klog.Errorf("failed to sync with latest state: %v", err)
			wc.sendError(err)
			return
		}
	}
	wch := wc.watcher.client.Watch(wc.ctx, wc.key, uint64(wc.initialRev+1))
	for wres := range wch {
		wc.sendEvent(&wres)
	}
	wc.sendError(fmt.Errorf("stop to watch sqlite/meta_v2"))
	close(watchClosedCh)
}

// processEvent processes events from etcd watcher and sends results to resultChan.
func (wc *watchChan) processEvent(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case e := <-wc.incomingEventChan:
			var res = e
			key, err := metaserver.KeyFuncObj(e.Object)
			if err != nil {
				klog.Errorf("failed to get key from obj:%v", err)
				continue
			}
			hasBeenAdded := wc.added[key]
			matched := wc.filter(e.Object)
			switch {
			case hasBeenAdded && !matched:
				// stop to watch this obj because it's field or label are no longer meet the internalPred
				res = &watch.Event{
					Type:   watch.Deleted,
					Object: e.Object,
				}
			case hasBeenAdded && matched:
				//添加过，且没被过滤，继续
			case !hasBeenAdded && !matched:
				//没添加过，被过滤了
				klog.V(4).Infof("[apiservelite-watchChan]filter event: %v", e)
				continue
			case !hasBeenAdded && matched && res.Type == watch.Modified:
				//没添加过，没被过滤，但是事件时modified
				res = &watch.Event{
					Type:   watch.Added,
					Object: e.Object,
				}
			}
			if len(wc.resultChan) == outgoingBufSize {
				klog.V(3).InfoS("Fast watcher, slow processing. Probably caused by slow dispatching events to watchers", "outgoingEvents", outgoingBufSize)
			}
			// If user couldn't receive results fast enough, we also block incoming events from watcher.
			// Because storing events in local will cause more memory usage.
			// The worst case would be closing the fast watcher.
			select {
			case wc.resultChan <- *res:
				if res.Type == watch.Deleted {
					wc.added[key] = false
				} else {
					wc.added[key] = true
				}
			case <-wc.ctx.Done():
				return
			}
		case <-wc.ctx.Done():
			return
		}
	}
}

func (wc *watchChan) filter(obj runtime.Object) bool {
	if wc.internalPred.Empty() {
		return true
	}
	matched, err := wc.internalPred.Matches(obj)
	return err == nil && matched
}

func (wc *watchChan) acceptAll() bool {
	return wc.internalPred.Empty()
}

func transformErrorToEvent(err error) *watch.Event {
	status := apierrors.NewInternalError(err).Status()
	return &watch.Event{
		Type:   watch.Error,
		Object: &status,
	}
}

func (wc *watchChan) sendError(err error) {
	select {
	case wc.errChan <- err:
	case <-wc.ctx.Done():
	}
}

func (wc *watchChan) sendEvent(e *watch.Event) {
	if len(wc.incomingEventChan) == incomingBufSize {
		klog.V(3).InfoS("Fast watcher, slow processing. Probably caused by slow decoding, user not receiving fast, or other processing logic", "incomingEvents", incomingBufSize)
	}
	select {
	case wc.incomingEventChan <- e:
	case <-wc.ctx.Done():
	}
}
