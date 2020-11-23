package context

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// constants for channel context
const (
	ChannelSizeDefault = 1024

	MessageTimeoutDefault = 30 * time.Second

	TickerTimeoutDefault = 20 * time.Millisecond
)

// ChannelContext is object for Context channel
type ChannelContext struct {
	//ConfigFactory goarchaius.ConfigurationFactory
	channels     map[string]chan model.Message
	chsLock      sync.RWMutex
	typeChannels map[string]map[string]chan model.Message
	typeChsLock  sync.RWMutex
	anonChannels map[string]chan model.Message
	anonChsLock  sync.RWMutex
}

// NewChannelContext creates and returns object of new channel context
// TODO: Singleton
func NewChannelContext() *ChannelContext {
	channelMap := make(map[string]chan model.Message)
	moduleChannels := make(map[string]map[string]chan model.Message)
	anonChannels := make(map[string]chan model.Message)
	return &ChannelContext{
		channels:     channelMap,
		typeChannels: moduleChannels,
		anonChannels: anonChannels,
	}
}

// Cleanup close modules
func (ctx *ChannelContext) Cleanup(module string) {
	if channel := ctx.getChannel(module); channel != nil {
		ctx.delChannel(module)
		// decrease probable exception of channel closing
		time.Sleep(20 * time.Millisecond)
		close(channel)
	}
}

// Send send msg to a module. Todo: do not stuck
func (ctx *ChannelContext) Send(module string, message model.Message) {
	// avoid exception because of channel closing
	// TODO: need reconstruction
	defer func() {
		if exception := recover(); exception != nil {
			klog.Warningf("Recover when send message, exception: %+v", exception)
		}
	}()

	if channel := ctx.getChannel(module); channel != nil {
		channel <- message
		return
	}
	klog.Warningf("Get bad module name :%s when send message, do nothing", module)
}

// Receive msg from channel of module
func (ctx *ChannelContext) Receive(module string) (model.Message, error) {
	if channel := ctx.getChannel(module); channel != nil {
		content := <-channel
		return content, nil
	}

	klog.Warningf("Failed to get channel for module:%s when receive message", module)
	return model.Message{}, fmt.Errorf("failed to get channel for module(%s)", module)
}

func getAnonChannelName(msgID string) string {
	return msgID
}

// SendSync sends message in a sync way
func (ctx *ChannelContext) SendSync(module string, message model.Message, timeout time.Duration) (model.Message, error) {
	// avoid exception because of channel closing
	// TODO: need reconstruction
	defer func() {
		if exception := recover(); exception != nil {
			klog.Warningf("Recover when sendsync message, exception: %+v", exception)
		}
	}()

	if timeout <= 0 {
		timeout = MessageTimeoutDefault
	}
	deadline := time.Now().Add(timeout)

	// make sure to set sync flag
	message.Header.Sync = true

	// check req/resp channel
	reqChannel := ctx.getChannel(module)
	if reqChannel == nil {
		return model.Message{}, fmt.Errorf("bad request module name(%s)", module)
	}

	sendTimer := time.NewTimer(timeout)
	select {
	case reqChannel <- message:
	case <-sendTimer.C:
		return model.Message{}, errors.New("timeout to send message")
	}
	sendTimer.Stop()

	// new anonymous channel for response
	anonChan := make(chan model.Message)
	anonName := getAnonChannelName(message.GetID())
	ctx.anonChsLock.Lock()
	ctx.anonChannels[anonName] = anonChan
	ctx.anonChsLock.Unlock()
	defer func() {
		ctx.anonChsLock.Lock()
		delete(ctx.anonChannels, anonName)
		close(anonChan)
		ctx.anonChsLock.Unlock()
	}()

	var resp model.Message
	respTimer := time.NewTimer(time.Until(deadline))
	select {
	case resp = <-anonChan:
	case <-respTimer.C:
		return model.Message{}, errors.New("timeout to get response")
	}
	respTimer.Stop()

	return resp, nil
}

// SendResp send resp for this message when using sync mode
func (ctx *ChannelContext) SendResp(message model.Message) {
	anonName := getAnonChannelName(message.GetParentID())

	ctx.anonChsLock.RLock()
	defer ctx.anonChsLock.RUnlock()
	if channel, exist := ctx.anonChannels[anonName]; exist {
		channel <- message
		return
	}

	klog.V(4).Infof("Get bad anonName:%s when sendresp message, do nothing", anonName)
}

// SendToGroup send msg to modules. Todo: do not stuck
func (ctx *ChannelContext) SendToGroup(moduleType string, message model.Message) {
	// avoid exception because of channel closing
	// TODO: need reconstruction
	defer func() {
		if exception := recover(); exception != nil {
			klog.Warningf("Recover when sendToGroup message, exception: %+v", exception)
		}
	}()

	send := func(ch chan model.Message) {
		select {
		case ch <- message:
		default:
			klog.Warningf("the message channel is full, message: %+v", message)
			select {
			case ch <- message:
			}
		}
	}
	if channelList := ctx.getTypeChannel(moduleType); channelList != nil {
		for _, channel := range channelList {
			go send(channel)
		}
		return
	}
	klog.Warningf("Get bad module type:%s when sendToGroup message, do nothing", moduleType)
}

// SendToGroupSync : broadcast the message to echo module channel, the module send response back anon channel
// check timeout and the size of anon channel
func (ctx *ChannelContext) SendToGroupSync(moduleType string, message model.Message, timeout time.Duration) error {
	// avoid exception because of channel closing
	// TODO: need reconstruction
	defer func() {
		if exception := recover(); exception != nil {
			klog.Warningf("Recover when sendToGroupsync message, exception: %+v", exception)
		}
	}()

	if timeout <= 0 {
		timeout = MessageTimeoutDefault
	}
	deadline := time.Now().Add(timeout)

	channelList := ctx.getTypeChannel(moduleType)
	if channelList == nil {
		return fmt.Errorf("failed to get module type(%s) channel list", moduleType)
	}

	// echo module must sync a response,
	// let anonchan size be module number
	channelNumber := len(channelList)
	anonChan := make(chan model.Message, channelNumber)
	anonName := getAnonChannelName(message.GetID())
	ctx.anonChsLock.Lock()
	ctx.anonChannels[anonName] = anonChan
	ctx.anonChsLock.Unlock()

	cleanup := func() error {
		ctx.anonChsLock.Lock()
		delete(ctx.anonChannels, anonName)
		close(anonChan)
		ctx.anonChsLock.Unlock()

		var uninvitedGuests int
		// cleanup anonchan and check parentid for resp
		for resp := range anonChan {
			if resp.GetParentID() != message.GetID() {
				uninvitedGuests++
			}
		}
		if uninvitedGuests != 0 {
			klog.Errorf("Get some unexpected:%d resp when sendToGroupsync message", uninvitedGuests)
			return fmt.Errorf("got some unexpected(%d) resp", uninvitedGuests)
		}
		return nil
	}

	// make sure to set sync flag before sending
	message.Header.Sync = true

	var timeoutCounter int32
	send := func(ch chan model.Message) {
		sendTimer := time.NewTimer(time.Until(deadline))
		select {
		case ch <- message:
			sendTimer.Stop()
		case <-sendTimer.C:
			atomic.AddInt32(&timeoutCounter, 1)
		}
	}
	for _, channel := range channelList {
		go send(channel)
	}

	sendTimer := time.NewTimer(time.Until(deadline))
	ticker := time.NewTicker(TickerTimeoutDefault)
	for {
		// annonChan is full
		if len(anonChan) == channelNumber {
			break
		}
		select {
		case <-ticker.C:
		case <-sendTimer.C:
			cleanup()
			if timeoutCounter != 0 {
				errInfo := fmt.Sprintf("timeout to send message, several %d timeout when send", timeoutCounter)
				return fmt.Errorf(errInfo)
			}
			klog.Error("Timeout to sendToGroupsync message")
			return fmt.Errorf("Timeout to send message")
		}
	}

	return cleanup()
}

// New Channel
func (ctx *ChannelContext) newChannel() chan model.Message {
	channel := make(chan model.Message, ChannelSizeDefault)
	return channel
}

// getChannel return chan
func (ctx *ChannelContext) getChannel(module string) chan model.Message {
	ctx.chsLock.RLock()
	defer ctx.chsLock.RUnlock()

	if _, exist := ctx.channels[module]; exist {
		return ctx.channels[module]
	}

	klog.Warningf("Failed to get channel, type:%s", module)
	return nil
}

// addChannel return chan
func (ctx *ChannelContext) addChannel(module string, moduleCh chan model.Message) {
	ctx.chsLock.Lock()
	defer ctx.chsLock.Unlock()

	ctx.channels[module] = moduleCh
}

// deleteChannel by module name
func (ctx *ChannelContext) delChannel(module string) {
	// delete module channel from channels map
	ctx.chsLock.Lock()
	_, exist := ctx.channels[module]
	if !exist {
		klog.Warningf("Failed to get channel, module:%s", module)
		return
	}
	delete(ctx.channels, module)

	ctx.chsLock.Unlock()

	// delete module channel from typechannels map
	ctx.typeChsLock.Lock()
	for _, moduleMap := range ctx.typeChannels {
		if _, exist := moduleMap[module]; exist {
			delete(moduleMap, module)
			break
		}
	}
	ctx.typeChsLock.Unlock()
}

// getTypeChannel return chan
func (ctx *ChannelContext) getTypeChannel(moduleType string) map[string]chan model.Message {
	ctx.typeChsLock.RLock()
	defer ctx.typeChsLock.RUnlock()

	if _, exist := ctx.typeChannels[moduleType]; exist {
		return ctx.typeChannels[moduleType]
	}

	klog.Warningf("Failed to get type channel, type:%s", moduleType)
	return nil
}

func (ctx *ChannelContext) getModuleByChannel(ch chan model.Message) string {
	ctx.chsLock.RLock()
	defer ctx.chsLock.RUnlock()

	for module, channel := range ctx.channels {
		if channel == ch {
			return module
		}
	}

	klog.Warning("Failed to get module by channel")
	return ""
}

// addTypeChannel put modules into moduleType map
func (ctx *ChannelContext) addTypeChannel(module, group string, moduleCh chan model.Message) {
	ctx.typeChsLock.Lock()
	defer ctx.typeChsLock.Unlock()

	if _, exist := ctx.typeChannels[group]; !exist {
		ctx.typeChannels[group] = make(map[string]chan model.Message)
	}
	ctx.typeChannels[group][module] = moduleCh
}

// AddModule adds module into module context
func (ctx *ChannelContext) AddModule(module string) {
	channel := ctx.newChannel()
	ctx.addChannel(module, channel)
}

// AddModuleGroup adds modules into module context group
func (ctx *ChannelContext) AddModuleGroup(module, group string) {
	if channel := ctx.getChannel(module); channel != nil {
		ctx.addTypeChannel(module, group, channel)
		return
	}
	klog.Warningf("Get bad module name %s when addmodulegroup", module)
}
