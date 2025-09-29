package streamrule

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubeedge/api/apis/streamrules/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/provider"
	"k8s.io/klog/v2"
)

var (
	streamrules         sync.Map
	streamruleEndpoints sync.Map
)

func init() {
	registerListener()
}

func registerListener() {
	endpointKey := fmt.Sprintf("%s/%s", modules.EdgeControllerModuleName, "streamruleendpoint")
	listener.MessageHandlerInstance.AddListener(endpointKey, handleStreamRuleEndpoint)

	ruleKey := fmt.Sprintf("%s/%s", modules.EdgeControllerModuleName, "streamrule")
	listener.MessageHandlerInstance.AddListener(ruleKey, handleStreamRule)
}

// implement listener.Handle
func handleStreamRuleEndpoint(data interface{}) (interface{}, error) {
	message, ok := data.(*model.Message)
	if !ok {
		return nil, fmt.Errorf("data type is %T", data)
	}

	streamRuleEndpoint, ok := message.Content.(*v1alpha1.StreamRuleEndpoint)
	if !ok {
		return nil, fmt.Errorf("message content type should be streamRuleEndpoint type. operation: %s, resource: %s",
			message.GetOperation(), message.GetResource())
	}

	switch message.GetOperation() {
	case model.InsertOperation:
		addStreamRuleEndpoint(streamRuleEndpoint)
	case model.DeleteOperation:
		deleteStreamRuleEndpoint(streamRuleEndpoint.Namespace, streamRuleEndpoint.Name)
	default:
		klog.Warningf("invalid message operation.")
	}
	return nil, nil
}

// implement listener.Handle
func handleStreamRule(data interface{}) (interface{}, error) {
	message, ok := data.(*model.Message)
	if !ok {
		return nil, fmt.Errorf("data type is %T", data)
	}

	streamRule, ok := message.Content.(*v1alpha1.StreamRule)
	if !ok {
		return nil, fmt.Errorf("message content type should be streamRule type. operation: %s, resource: %s",
			message.GetOperation(), message.GetResource())
	}

	switch message.GetOperation() {
	case model.InsertOperation:
		addStreamRuleWithRetry(streamRule)
	case model.DeleteOperation:
		deleteStreamRule(streamRule.Namespace, streamRule.Name)
	default:
		klog.Warningf("invalid message operation.")
	}
	return nil, nil
}

func addStreamRuleEndpoint(endpoint *v1alpha1.StreamRuleEndpoint) {
	key := getKey(endpoint.Namespace, endpoint.Name)
	streamruleEndpoints.Store(key, endpoint)
	klog.Infof("add streamruleendpoint %s success.", key)
}

func deleteStreamRuleEndpoint(namespace, name string) {
	key := getKey(namespace, name)
	streamruleEndpoints.Delete(key)
	klog.Infof("delete streamruleendpoint %s success.", key)
}

func addStreamRule(sr *v1alpha1.StreamRule) error {
	targets, err := getTargetsOfStreamRule(sr)
	if err != nil {
		klog.Error(err)
		return err
	}
	streamruleKey := getKey(sr.Namespace, sr.Name)

	for _, target := range targets {
		t := target // 新建一个局部变量，避免闭包捕获问题

		if err := target.RegisterListener(func(data interface{}) (interface{}, error) {
			var execResult ExecResult
			resp, err := t.SendMsg(data)
			if err != nil {
				errMsg := ErrorMsg{Detail: err.Error(), Timestamp: time.Now()}
				execResult = ExecResult{StreamruleID: sr.Name, Namespace: sr.Namespace, Status: "FAIL", Error: errMsg}
			} else {
				execResult = ExecResult{StreamruleID: sr.Name, Namespace: sr.Namespace, Status: "SUCCESS"}
			}
			ResultChannel <- execResult
			return resp, nil
		}); err != nil {
			klog.Error(err)
			errMsg := ErrorMsg{Detail: err.Error(), Timestamp: time.Now()}
			execResult := ExecResult{StreamruleID: sr.Name, Namespace: sr.Namespace, Status: "FAIL", Error: errMsg}
			ResultChannel <- execResult
			return err
		}

	}
	streamrules.Store(streamruleKey, sr)
	klog.Infof("add streamrule %s success.", streamruleKey)
	return nil
}

func addStreamRuleWithRetry(sr *v1alpha1.StreamRule) {
	retry, waitTime := 3, 5
	for i := 0; i < retry; i++ {
		if err := addStreamRule(sr); err == nil {
			break
		}
		klog.Errorf("add streamrule failed, retry times: %d", i+1)
		time.Sleep(time.Duration(waitTime) * time.Second)
	}
}

func deleteStreamRule(namespace, name string) {
	streamruleKey := getKey(namespace, name)
	value, exist := streamrules.Load(streamruleKey)
	if !exist {
		klog.Errorf("streamrule %s not found", streamruleKey)
		return
	}
	streamrule := value.(*v1alpha1.StreamRule)
	targets, err := getTargetsOfStreamRule(streamrule)
	if err != nil {
		klog.Error(err)
	}
	for _, target := range targets {
		target.UnregisterListener()
		klog.Infof("deleteStreamRule: unregister target %s listener success.", target.Name())
	}
	streamrules.Delete(streamruleKey)
	klog.Infof("delete streamrule %s success.", streamruleKey)
}

func getKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func getTargetsOfStreamRule(sr *v1alpha1.StreamRule) (provider.Targets, error) {
	var result provider.Targets
	targets := sr.Spec.Targets

	for _, target := range targets {
		targetKey := getKey(sr.Namespace, target.EndpointRef)
		endpoint, exist := streamruleEndpoints.Load(targetKey)
		if !exist {
			return nil, fmt.Errorf("target streamruleendpoint %s does not existing", targetKey)
		}

		targetEp := endpoint.(*v1alpha1.StreamRuleEndpoint)
		tf, exist := provider.GetTargetFactory(targetEp.Spec.Protocol)
		if !exist {
			return nil, fmt.Errorf("target definition %s does not existing", targetEp.Spec.Protocol)
		}

		t := tf.GetTarget(targetEp, target.TargetResource)
		if t == nil {
			return nil, fmt.Errorf("can't get target: %s", target.EndpointRef)
		}
		result = append(result, t)
	}
	return result, nil
}
