package smgr

import (
	"fmt"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/comm"
	"github.com/lucas-clemente/quic-go"
)

const (
	NumStreamsMax = 99

	PoolStreamMinDefault = 2
	ThresholdDefault     = 0.8
)

// get stream from session
// AcceptStream or OpenStreamXX
type GetFuncEx func(api.UseType, bool) (*Stream, error)

type StreamManager struct {
	NumStreamsMax int
	Session       *Session

	messagePool PoolManager
	binaryPool  PoolManager
	lock        sync.RWMutex
}

type PoolManager struct {
	idlePool streamPool
	busyPool streamPool
	lock     sync.RWMutex
	cond     sync.Cond
	autoFree bool
}

type streamPool struct {
	streamMap  map[quic.StreamID]*Stream
	streamFiFo []*Stream
	lock       sync.RWMutex
}

func (pool *streamPool) addStream(stream *Stream) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	pool.streamMap[stream.Stream.StreamID()] = stream
}

func (pool *streamPool) len() int {
	pool.lock.RLock()
	defer pool.lock.RUnlock()
	return len(pool.streamMap)
}

func (pool *streamPool) delStream(stream quic.Stream) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	delete(pool.streamMap, stream.StreamID())
}

func (pool *streamPool) getStream() *Stream {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	// get one stream randomly
	var stream *Stream
	for _, stream = range pool.streamMap {
		break
	}
	if stream != nil {
		delete(pool.streamMap, stream.Stream.StreamID())
	}
	return stream
}

// free stream by stream id
func (pool *streamPool) freeStream(s quic.Stream) bool {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	for streamID := range pool.streamMap {
		if streamID == s.StreamID() {
			delete(pool.streamMap, s.StreamID())
			s.CancelRead(quic.ErrorCode(comm.StatusCodeFreeStream))
			s.Close()
			s.CancelWrite(quic.ErrorCode(comm.StatusCodeFreeStream))
			return true
		}
	}
	return false
}

func (pool *streamPool) destroyStreams() {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	for _, stream := range pool.streamMap {
		stream.Stream.CancelRead(quic.ErrorCode(comm.StatusCodeFreeStream))
		stream.Stream.Close()
		stream.Stream.CancelWrite(quic.ErrorCode(comm.StatusCodeFreeStream))
	}
	pool.streamMap = make(map[quic.StreamID]*Stream)
}

func (mgr *PoolManager) len() int {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	return mgr.idlePool.len() + mgr.busyPool.len()
}

func (mgr *PoolManager) acquireStream() (*Stream, error) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	// get a stream from idle pool
	stream := mgr.idlePool.getStream()
	if stream != nil {
		// add into busy pool
		mgr.busyPool.addStream(stream)
		return stream, nil
	}
	return nil, fmt.Errorf("no stream existing")
}

func (mgr *PoolManager) addBusyStream(stream *Stream) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	mgr.busyPool.addStream(stream)
}

func (mgr *PoolManager) addIdleStream(stream *Stream) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	mgr.idlePool.addStream(stream)
	mgr.cond.Signal()
}

func (mgr *PoolManager) availableOrWait() {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	if mgr.idlePool.len() <= 0 {
		mgr.cond.Wait()
	}
}

func (mgr *PoolManager) freeStream(stream quic.Stream) {
	if freed := mgr.idlePool.freeStream(stream); !freed {
		mgr.busyPool.freeStream(stream)
	}
}

// check and free the idle streams when the idle number
// is more 80% when releaseStream called
func (mgr *PoolManager) checkThreshold() bool {
	totalLen := mgr.idlePool.len() + mgr.busyPool.len()
	klog.Infof("total: %v, idle: %v, busy: %v", totalLen, mgr.idlePool.len(), mgr.busyPool.len())
	if totalLen >= PoolStreamMinDefault &&
		mgr.idlePool.len() >= (int(float64(totalLen)*ThresholdDefault)+1) {
		return true
	}
	return false
}

func (mgr *PoolManager) releaseStream(stream quic.Stream) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// delete the stream from busy pool
	mgr.busyPool.delStream(stream)

	// add into idle pool
	mgr.idlePool.addStream(&Stream{
		Stream: stream,
	})

	if mgr.autoFree {
		overrun := mgr.checkThreshold()
		if overrun {
			klog.Info("start to free idle streams")
			mgr.idlePool.destroyStreams()
		}
	}

	mgr.cond.Signal()
}

func (mgr *PoolManager) Destroy() {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	mgr.idlePool.destroyStreams()
	mgr.busyPool.destroyStreams()
}

func NewStreamManager(streamMax int, autoFree bool, session quic.Session) *StreamManager {
	if streamMax <= 0 {
		streamMax = NumStreamsMax
	}

	streamMgr := &StreamManager{
		NumStreamsMax: streamMax,
		Session:       &Session{session},
		messagePool: PoolManager{
			idlePool: streamPool{
				streamMap: make(map[quic.StreamID]*Stream),
			},
			busyPool: streamPool{
				streamMap: make(map[quic.StreamID]*Stream),
			},
			autoFree: autoFree,
		},
		binaryPool: PoolManager{
			idlePool: streamPool{
				streamMap: make(map[quic.StreamID]*Stream),
			},
			busyPool: streamPool{
				streamMap: make(map[quic.StreamID]*Stream),
			},
			autoFree: autoFree,
		},
	}
	streamMgr.messagePool.cond.L = &streamMgr.messagePool.lock
	streamMgr.binaryPool.cond.L = &streamMgr.binaryPool.lock
	return streamMgr
}

func (mgr *StreamManager) getPoolManager(useType api.UseType) *PoolManager {
	var poolMgr *PoolManager
	switch useType {
	case api.UseTypeMessage:
		poolMgr = &mgr.messagePool
	case api.UseTypeStream:
		poolMgr = &mgr.binaryPool
	default:
		klog.Errorf("bad stream use type(%s)%s, ", useType, api.UseTypeMessage)
	}
	return poolMgr
}

func (mgr *StreamManager) ReleaseStream(useType api.UseType, stream quic.Stream) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// try to get stream from pool
	poolMgr := mgr.getPoolManager(useType)
	if poolMgr == nil {
		return
	}
	poolMgr.releaseStream(stream)
}

func (mgr *StreamManager) AddStream(stream *Stream) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// try to get stream from pool
	poolMgr := mgr.getPoolManager(stream.UseType)
	if poolMgr == nil {
		return
	}
	poolMgr.addIdleStream(stream)
}

func (mgr *StreamManager) GetStream(useType api.UseType, autoDispatch bool, getFuncEx GetFuncEx) (quic.Stream, error) {
	mgr.lock.Lock()
	// try to get stream from pool
	poolMgr := mgr.getPoolManager(useType)
	if poolMgr == nil {
		mgr.lock.Unlock()
		return nil, fmt.Errorf("bad stream use type(%s)", useType)
	}

	// get stream from stream pool
	if getFuncEx == nil {
		// wait if have no idle stream
		mgr.lock.Unlock()
		poolMgr.availableOrWait()
		mgr.lock.Lock()
	} else {
		// check the max number of streams
		total := mgr.binaryPool.len() + mgr.messagePool.len()
		if total >= mgr.NumStreamsMax {
			// if no available idle stream, block and wait
			klog.Info("wait for idle stream")
			mgr.lock.Unlock()
			// check it has a available stream or wait for a stream
			poolMgr.availableOrWait()
			mgr.lock.Lock()
		}
	}

	// acquire stream from current stream pool
	stream, err := poolMgr.acquireStream()
	if err == nil {
		mgr.lock.Unlock()
		return stream.Stream, err
	}
	mgr.lock.Unlock()

	// failed to acquire stream
	// return err if just want get stream from pool
	if getFuncEx == nil {
		return nil, fmt.Errorf("failed to get stream, error: %+v", err)
	}

	// try to get stream from session
	stream, err = getFuncEx(useType, autoDispatch)
	if err != nil {
		klog.Warningf("get stream error(%+v)", err)
		return nil, err
	}

	// add the new stream into pools
	mgr.lock.Lock()
	poolMgr.addBusyStream(stream)
	mgr.lock.Unlock()
	return stream.Stream, nil
}

func (mgr *StreamManager) FreeStream(stream *Stream) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// try to get stream from pool
	poolMgr := mgr.getPoolManager(stream.UseType)
	if poolMgr == nil {
		return
	}
	poolMgr.freeStream(stream.Stream)
}

func (mgr *StreamManager) Destroy() {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	mgr.messagePool.Destroy()
	mgr.binaryPool.Destroy()
}
