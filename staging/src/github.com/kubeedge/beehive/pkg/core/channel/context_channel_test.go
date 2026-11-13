package channel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core/model"
)

func newTestContext() *Context {
	return &Context{
		channels:     make(map[string]chan model.Message),
		typeChannels: make(map[string]map[string]chan model.Message),
		anonChannels: make(map[string]chan model.Message),
	}
}

func TestSendSyncEarlyResponse(t *testing.T) {
	const (
		moduleName = "test-receiver"
		iterations = 20
		timeout    = 2 * time.Second
	)

	for i := 0; i < iterations; i++ {
		ctx := newTestContext()
		ctx.addChannel(moduleName, make(chan model.Message))

		reqMsg := model.NewMessage("")
		reqMsg.FillBody("ping")

		replied := make(chan struct{})
		go func() {
			defer close(replied)
			recvMsg, err := ctx.Receive(moduleName)
			if err != nil {
				t.Errorf("receive request: %v", err)
				return
			}
			resp := model.NewMessage(recvMsg.GetID())
			resp.FillBody("pong")
			ctx.SendResp(*resp)
		}()

		resp, err := ctx.SendSync(moduleName, *reqMsg, timeout)
		require.NoError(t, err, "iteration %d", i)
		assert.Equal(t, "pong", resp.GetContent())
		assert.Equal(t, reqMsg.GetID(), resp.GetParentID())
		<-replied
	}
}

func TestSendSyncResponseBeforeReceive(t *testing.T) {
	const (
		moduleName = "test-receiver-early"
		timeout    = 2 * time.Second
	)

	ctx := newTestContext()
	reqCh := make(chan model.Message)
	ctx.addChannel(moduleName, reqCh)

	reqMsg := model.NewMessage("")
	reqMsg.FillBody("ping")

	type result struct {
		resp model.Message
		err  error
	}
	resCh := make(chan result, 1)
	go func() {
		resp, err := ctx.SendSync(moduleName, *reqMsg, timeout)
		resCh <- result{resp: resp, err: err}
	}()

	registered := func() bool {
		ctx.anonChsLock.RLock()
		defer ctx.anonChsLock.RUnlock()
		_, ok := ctx.anonChannels[getAnonChannelName(reqMsg.GetID())]
		return ok
	}
	deadline := time.Now().Add(time.Second)
	for !registered() {
		if time.Now().After(deadline) {
			t.Fatal("SendSync did not register its response channel")
		}
		time.Sleep(time.Millisecond)
	}

	resp := model.NewMessage(reqMsg.GetID())
	resp.FillBody("pong")
	ctx.SendResp(*resp)

	select {
	case got := <-reqCh:
		assert.Equal(t, reqMsg.GetID(), got.GetID())
	case <-time.After(time.Second):
		t.Fatal("request was never handed to the module")
	}

	select {
	case res := <-resCh:
		require.NoError(t, res.err)
		assert.Equal(t, "pong", res.resp.GetContent())
		assert.Equal(t, reqMsg.GetID(), res.resp.GetParentID())
	case <-time.After(timeout + time.Second):
		t.Fatal("SendSync did not return")
	}
}

func TestSendSyncLateResponse(t *testing.T) {
	const moduleName = "test-receiver-late"

	ctx := newTestContext()
	ctx.AddModule(&common.ModuleInfo{ModuleName: moduleName})

	reqMsg := model.NewMessage("")
	reqMsg.FillBody("hello")

	replied := make(chan struct{})
	go func() {
		defer close(replied)
		recvMsg, err := ctx.Receive(moduleName)
		if err != nil {
			t.Errorf("receive request: %v", err)
			return
		}
		time.Sleep(20 * time.Millisecond)
		resp := model.NewMessage(recvMsg.GetID())
		resp.FillBody("world")
		ctx.SendResp(*resp)
	}()

	resp, err := ctx.SendSync(moduleName, *reqMsg, 5*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "world", resp.GetContent())
	<-replied
}

func TestSendSyncTimeout(t *testing.T) {
	const moduleName = "test-receiver-timeout"

	ctx := newTestContext()
	ctx.AddModule(&common.ModuleInfo{ModuleName: moduleName})

	reqMsg := model.NewMessage("")
	reqMsg.FillBody("request")

	go func() {
		_, _ = ctx.Receive(moduleName)
	}()

	_, err := ctx.SendSync(moduleName, *reqMsg, 100*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestSendSyncBadModule(t *testing.T) {
	ctx := newTestContext()

	reqMsg := model.NewMessage("")
	reqMsg.FillBody("test")

	_, err := ctx.SendSync("nonexistent-module", *reqMsg, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad request module name")
}
