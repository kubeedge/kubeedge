package keeper

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
)

type SyncKeeper struct {
	keeper sync.Map
}

func NewSyncKeeper() *SyncKeeper {
	return &SyncKeeper{}
}

func (k *SyncKeeper) sendToKeepChannel(msg model.Message) error {
	obj, exist := k.keeper.Load(msg.GetParentID())
	if !exist {
		klog.Errorf("failed to get sync keeper channel, message id:%s", msg.GetID())
		return fmt.Errorf("failed to get sync keeper channel")
	}

	channel := obj.(chan model.Message)
	select {
	case channel <- msg:
	default:
		klog.Error("keeper channel is full")
		return fmt.Errorf("keeper channel is full")
	}
	return nil
}

func (k *SyncKeeper) addKeepChannel(msgID string) chan model.Message {
	channel := make(chan model.Message)
	k.keeper.Store(msgID, channel)
	return channel
}

func (k *SyncKeeper) deleteKeepChannel(msgID string) {
	k.keeper.Delete(msgID)
}

func (k *SyncKeeper) Match(msg model.Message) bool {
	_, exist := k.keeper.Load(msg.GetParentID())
	return exist
}

func (k *SyncKeeper) MatchAndNotify(msg model.Message) bool {
	if matched := k.Match(msg); !matched {
		return false
	}

	if err := k.sendToKeepChannel(msg); err != nil {
		klog.Errorf("failed to send to keep channel, error:%+v", err)
	}
	return true
}

func (k *SyncKeeper) WaitResponse(msg *model.Message, deadline time.Time) (model.Message, error) {
	msgID := msg.GetID()
	channel := k.addKeepChannel(msgID)
	timer := time.NewTimer(deadline.Sub(time.Now()))
	select {
	case resp := <-channel:
		timer.Stop()
		k.keeper.Delete(msgID)
		return resp, nil
	case <-timer.C:
		klog.Warningf("wait response timeout, message id:%s", msgID)
		k.keeper.Delete(msgID)
		return model.Message{}, fmt.Errorf("wait response timeout, message id:%s", msgID)
	}
}
