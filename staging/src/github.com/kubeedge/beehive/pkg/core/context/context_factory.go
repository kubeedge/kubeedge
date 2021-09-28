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

// GlobalContext is global context: only use for local cache to dispatch message
type GlobalContext struct {
	// context type(socket/channel) -> context
	moduleContext  map[string]ModuleContext
	messageContext map[string]MessageContext

	// module name to context type
	moduleContextType map[string]string
	// group name to context type
	groupContextType map[string]string

	ctx     gocontext.Context
	cancel  gocontext.CancelFunc
	ctxLock sync.RWMutex
}

func init() {
	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	globalContext = &GlobalContext{
		moduleContext:  make(map[string]ModuleContext),
		messageContext: make(map[string]MessageContext),

		moduleContextType: make(map[string]string),
		groupContextType:  make(map[string]string),

		ctx:    ctx,
		cancel: cancel,
	}
}

var (
	// singleton
	globalContext *GlobalContext
	once          sync.Once
)

// InitContext gets global context instance
func InitContext(contextTypes []string) {
	for _, contextType := range contextTypes {
		switch contextType {
		case common.MsgCtxTypeChannel:
			channelContext := channel.NewChannelContext()
			globalContext.moduleContext[contextType] = channelContext
			globalContext.messageContext[contextType] = channelContext
		case common.MsgCtxTypeUS:
			socketContext := socket.InitSocketContext()
			globalContext.moduleContext[contextType] = socketContext
			globalContext.messageContext[contextType] = socketContext
		default:
			klog.Fatalf("unsupported context type: %s", contextType)
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
func AddModule(module *common.ModuleInfo) {
	setModuleContextType(module.ModuleName, module.ModuleType)

	moduleContext, err := getModuleContext(module.ModuleName)
	if err != nil {
		klog.Errorf("failed to get module context, module name: %s, err: %v", module.ModuleName, err)
		return
	}

	moduleContext.AddModule(module)
}

// AddModuleGroup adds module into module context group
func AddModuleGroup(module, group string) {
	setGroupContextType(module, group)

	moduleContext, err := getModuleContext(module)
	if err != nil {
		klog.Errorf("failed to get module context, module name: %s, err: %v", module, err)
		return
	}

	moduleContext.AddModuleGroup(module, group)
}

// Cancel function
func Cancel() {
	globalContext.cancel()
}

// Cleanup cleans up module
func Cleanup(module string) {
	moduleContext, err := getModuleContext(module)
	if err != nil {
		klog.Errorf("failed to get module context, module name: %s, err: %v", module, err)
		return
	}

	moduleContext.Cleanup(module)
}

// Send the message
func Send(module string, message model.Message) {
	messageContext, err := getMessageContext(module)
	if err != nil {
		return
	}

	messageContext.Send(module, message)
}

// Receive the message
// module : local module name
func Receive(module string) (model.Message, error) {
	messageContext, err := getMessageContext(module)
	if err != nil {
		return model.Message{}, err
	}

	return messageContext.Receive(module)
}

// SendSync sends message in sync mode
// module: the destination of the message
// timeout: if <= 0 using default value(30s)
func SendSync(module string,
	message model.Message, timeout time.Duration) (model.Message, error) {
	messageContext, err := getMessageContext(module)
	if err != nil {
		return model.Message{}, err
	}

	return messageContext.SendSync(module, message, timeout)
}

// SendResp sends response
// please get resp message using model.NewRespByMessage
func SendResp(resp model.Message) {
	messageContext, err := getMessageContextByMessageType(resp.GetType())
	if err != nil {
		klog.Errorf("message context for module doesn't exist, module name: %s", resp.GetSource())
		return
	}

	messageContext.SendResp(resp)
}

// SendToGroup broadcasts the message to all of group members
func SendToGroup(group string, message model.Message) {
	messageContext, err := getMessageContextByGroup(group)
	if err != nil {
		klog.Errorf("message context for group doesn't exist, group name: %s", group)
		return
	}

	messageContext.SendToGroup(group, message)
}

// SendToGroupSync broadcasts the message to all of group members in sync mode
func SendToGroupSync(group string, message model.Message, timeout time.Duration) error {
	messageContext, err := getMessageContextByGroup(group)
	if err != nil {
		return fmt.Errorf("message context for group doesn't exist, group name: %s", group)
	}

	return messageContext.SendToGroupSync(group, message, timeout)
}

func getModuleContext(moduleName string) (ModuleContext, error) {
	globalContext.ctxLock.RLock()
	defer globalContext.ctxLock.RUnlock()

	moduleContextType := getModuleContextType(moduleName)
	moduleContext, ok := globalContext.moduleContext[moduleContextType]
	if !ok {
		return nil, fmt.Errorf("module context %v doesn't exist", moduleContextType)
	}

	return moduleContext, nil
}

func getMessageContext(moduleName string) (MessageContext, error) {
	globalContext.ctxLock.RLock()
	defer globalContext.ctxLock.RUnlock()

	moduleContextType := getModuleContextType(moduleName)
	messageContext, ok := globalContext.messageContext[moduleContextType]
	if !ok {
		return nil, fmt.Errorf("message context %v doesn't exist", moduleContextType)
	}

	return messageContext, nil
}

func getMessageContextByMessageType(messageType string) (MessageContext, error) {
	if messageType == "" {
		messageType = common.MsgCtxTypeChannel
	}

	globalContext.ctxLock.RLock()
	defer globalContext.ctxLock.RUnlock()

	messageContext, ok := globalContext.messageContext[messageType]
	if !ok {
		return nil, fmt.Errorf("message context for message type doesn't exist, message type: %s", messageType)
	}

	return messageContext, nil
}

func getMessageContextByGroup(group string) (MessageContext, error) {
	globalContext.ctxLock.RLock()
	defer globalContext.ctxLock.RUnlock()

	contextType := globalContext.groupContextType[group]
	messageContext, ok := globalContext.messageContext[contextType]
	if !ok {
		return nil, fmt.Errorf("message context doesn't exist, group: %s, contextType: %s", group, contextType)
	}

	return messageContext, nil
}

// caller must lock the globalContext.ctxLock
func getModuleContextType(moduleName string) string {
	return globalContext.moduleContextType[moduleName]
}

func setModuleContextType(moduleName string, contextType string) {
	globalContext.ctxLock.Lock()
	defer globalContext.ctxLock.Unlock()
	globalContext.moduleContextType[moduleName] = contextType
}

func setGroupContextType(module string, group string) {
	globalContext.ctxLock.Lock()
	defer globalContext.ctxLock.Unlock()

	globalContext.groupContextType[group] = globalContext.moduleContextType[module]
}
