package context

import (
	gocontext "context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core/channel"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/beehive/pkg/core/socket"
)

// BeehiveContext is global context: only use for local cache to dispatch message
type BeehiveContext struct {
	// module -> ModuleInfo
	moduleInfo map[string]*common.ModuleInfo
	moduleLock sync.RWMutex
	// group -> ModuleInfo
	groupInfo map[string]*common.ModuleInfo
	groupLock sync.RWMutex

	// type(socket/channel) -> context
	moduleHandler  map[string]ModuleContext
	messageHandler map[string]MessageContext

	ctx    gocontext.Context
	cancel gocontext.CancelFunc
}

func init() {
	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	globalContext = &BeehiveContext{
		moduleInfo:     make(map[string]*common.ModuleInfo),
		groupInfo:      make(map[string]*common.ModuleInfo),
		moduleHandler:  make(map[string]ModuleContext),
		messageHandler: make(map[string]MessageContext),
		ctx:            ctx,
		cancel:         cancel,
	}
}

var (
	// singleton
	globalContext *BeehiveContext
	once          sync.Once
)

// InitContext gets global context instance
func InitContext(contextTypes []string) {
	for _, contextType := range contextTypes {
		switch contextType {
		case common.MsgCtxTypeChannel:
			channelContext := channel.NewChannelContext()
			globalContext.moduleHandler[contextType] = channelContext
			globalContext.messageHandler[contextType] = channelContext
		case common.MsgCtxTypeUS:
			socketContext := socket.InitSocketContext()
			globalContext.moduleHandler[common.MsgCtxTypeUS] = socketContext
			globalContext.messageHandler[common.MsgCtxTypeUS] = socketContext
		default:
			klog.Fatalf("Do not support context type:%s", contextType)
		}
	}
}

func GetContext() gocontext.Context {
	return globalContext.ctx
}

func Done() <-chan struct{} {
	return globalContext.ctx.Done()
}

// AddModule adds module into module context
func AddModule(module common.ModuleInfo) {
	globalContext.AddModule(&module)
}

// AddModuleGroup adds module into module context group
func AddModuleGroup(module, group string) {
	globalContext.AddModuleGroup(module, group)
}

// Cancel function
func Cancel() {
	globalContext.cancel()
}

// Cleanup cleans up module
func Cleanup(module string) {
	globalContext.Cleanup(module)
}

// Send the message
func Send(module string, message model.Message) {
	globalContext.Send(module, message)
}

// Receive the message
// module : local module name
func Receive(module string) (model.Message, error) {
	return globalContext.Receive(module)
}

// SendSync sends message in sync mode
// module: the destination of the message
// timeout: if <= 0 using default value(30s)
func SendSync(module string,
	message model.Message, timeout time.Duration) (model.Message, error) {
	return globalContext.SendSync(module, message, timeout)
}

// SendResp sends response
// please get resp message using model.NewRespByMessage
func SendResp(resp model.Message) {
	globalContext.SendResp(resp)
}

// SendToGroup broadcasts the message to all of group members
func SendToGroup(moduleType string, message model.Message) {
	globalContext.SendToGroup(moduleType, message)
}

// SendToGroupSync broadcasts the message to all of group members in sync mode
func SendToGroupSync(moduleType string, message model.Message, timeout time.Duration) error {
	return globalContext.sendToGroupSync(moduleType, message, timeout)
}

// AddModule add module into module context
func (ctx *BeehiveContext) AddModule(module *common.ModuleInfo) {
	storeModuleInfo(module)

	handler, err := getModuleHandler(module.ModuleName)
	if err != nil {
		klog.Errorf("failed to get module handler, err is %v", err)
		return
	}

	handler.AddModule(module)
}

// AddModuleGroup add module into module context group
func (ctx *BeehiveContext) AddModuleGroup(module, group string) {
	handler, err := getModuleHandler(module)
	if err != nil {
		klog.Errorf("failed to get module handler, err is %v", err)
		return
	}
	handler.AddModuleGroup(module, group)

	// store group info
	storeGroupInfo(module, group)
	return
}

// Cleanup clean up module
func (ctx *BeehiveContext) Cleanup(module string) {
	handler, err := getModuleHandler(module)
	if err != nil {
		klog.Errorf("failed to get module handler, err is %v", err)
		return
	}

	handler.Cleanup(module)

	removeHandler(module)
	return
}

// Send send the message
func (ctx *BeehiveContext) Send(module string, message model.Message) {
	handler, err := getMessageHandler(module)
	if err != nil {
		return
	}

	handler.Send(module, message)
	return
}

// Receive receive the message
// module : local module name
func (ctx *BeehiveContext) Receive(module string) (model.Message, error) {
	handler, err := getMessageHandler(module)
	if err != nil {
		return model.Message{}, err
	}
	return handler.Receive(module)
}

// SendSync send message in sync mode
// module: the destination of the message
// timeout: if <= 0 using default value(30s)
func (ctx *BeehiveContext) SendSync(module string,
	message model.Message, timeout time.Duration) (model.Message, error) {
	handler, err := getMessageHandler(module)
	if err != nil {
		return model.Message{}, err
	}
	return handler.SendSync(module, message, timeout)
}

// SendResp send response to local
// please get resp message using model.NewRespByMessage
// opts[0]: module name
func (ctx *BeehiveContext) SendResp(resp model.Message, opts ...string) {
	messageType := resp.GetType()
	if messageType == "" {
		messageType = common.MsgCtxTypeChannel
	}

	if messageType == common.MsgCtxTypeChannel {
		moduleContext := channel.GetChannelContext()
		moduleContext.SendResp(resp)
		return
	}

	handler, err := getMessageHandler(resp.GetSource())
	if err != nil {
		return
	}
	handler.SendResp(resp)
	return
}

// SendToGroup broadcast the message to all of group members
func (ctx *BeehiveContext) SendToGroup(groupType string, message model.Message) {
	handler, err := getGroupMessageHandler(groupType)
	if err != nil {
		klog.Errorf("failed to send message to group %v, err is %v", groupType, err)
		return
	}

	handler.SendToGroup(groupType, message)

	return
}

// send2GroupSync broadcast the message to all of group members in sync mode
func (ctx *BeehiveContext) sendToGroupSync(groupType string, message model.Message, timeout time.Duration) error {
	handler, err := getGroupMessageHandler(groupType)
	if err != nil {
		klog.Errorf("failed to send sync message to group %v, err is %v", groupType, err)
		return err
	}

	return handler.SendToGroupSync(groupType, message, timeout)
}

func storeModuleInfo(module *common.ModuleInfo) {
	globalContext.moduleLock.Lock()
	defer globalContext.moduleLock.Unlock()
	globalContext.moduleInfo[module.ModuleName] = module

}

func storeGroupInfo(module string, group string) {
	globalContext.moduleLock.RLock()
	info, _ := globalContext.moduleInfo[module]
	globalContext.moduleLock.RUnlock()

	globalContext.groupLock.Lock()
	globalContext.groupInfo[group] = info
	globalContext.groupLock.Unlock()
}

func getGroupMessageHandler(group string) (MessageContext, error) {
	globalContext.moduleLock.RLock()
	defer globalContext.moduleLock.RUnlock()
	info, ok := globalContext.groupInfo[group]
	if !ok {
		return nil, fmt.Errorf("group %s not exist", group)
	}
	handler, ok := globalContext.messageHandler[info.ModuleType]
	if !ok {
		return nil, fmt.Errorf("message handler %v not exist", info.ModuleType)
	}

	return handler, nil
}

func getModuleHandler(moduleName string) (ModuleContext, error) {
	globalContext.moduleLock.RLock()
	defer globalContext.moduleLock.RUnlock()
	info, ok := globalContext.moduleInfo[moduleName]
	if !ok {
		return nil, fmt.Errorf("module %s not exist", moduleName)
	}
	handler, ok := globalContext.moduleHandler[info.ModuleType]
	if !ok {
		return nil, fmt.Errorf("module handler %v not exist", info.ModuleType)
	}

	return handler, nil
}

func getMessageHandler(moduleName string) (MessageContext, error) {
	globalContext.moduleLock.RLock()
	defer globalContext.moduleLock.RUnlock()
	info, ok := globalContext.moduleInfo[moduleName]
	if !ok {
		return nil, fmt.Errorf("module %s not exist", moduleName)
	}
	handler, ok := globalContext.messageHandler[info.ModuleType]
	if !ok {
		return nil, fmt.Errorf("message handler %v not exist", info.ModuleType)
	}

	return handler, nil
}

func removeHandler(module string) {
	globalContext.moduleLock.Lock()
	delete(globalContext.moduleInfo, module)
	delete(globalContext.messageHandler, module)
	globalContext.moduleLock.Unlock()
}
