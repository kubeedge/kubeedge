package rule

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	routerv1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
)

var (
	rules         sync.Map
	ruleEndpoints sync.Map
)

func init() {
	registerListener()
}

func registerListener() {
	endpointKey := fmt.Sprintf("%s/%s", modules.EdgeControllerModuleName, model.ResourceTypeRuleEndpoint)
	listener.MessageHandlerInstance.AddListener(endpointKey, handleRuleEndpoint)

	ruleKey := fmt.Sprintf("%s/%s", modules.EdgeControllerModuleName, model.ResourceTypeRule)
	listener.MessageHandlerInstance.AddListener(ruleKey, handleRule)
}

// implement listener.Handle
func handleRuleEndpoint(data interface{}) (interface{}, error) {
	message, ok := data.(*model.Message)
	if !ok {
		klog.Warningf("object type: %T unsupported", data)
		return nil, fmt.Errorf("data type is %T", data)
	}

	ruleEndpoint, ok := message.Content.(*routerv1.RuleEndpoint)
	if !ok {
		klog.Warningf("object type: %T unsupported", message.Content)
		return nil, fmt.Errorf("message content type should be ruleEndpoint type. operation: %s, resource: %s",
			message.GetOperation(), message.GetResource())
	}

	switch message.GetOperation() {
	case model.InsertOperation:
		addRuleEndpoint(ruleEndpoint)
	case model.DeleteOperation:
		deleteRuleEndpoint(ruleEndpoint.Namespace, ruleEndpoint.Name)
	default:
		klog.Warningf("invalid message operation.")
	}
	return nil, nil
}

// implement listener.Handle
func handleRule(data interface{}) (interface{}, error) {
	message, ok := data.(*model.Message)
	if !ok {
		klog.Warningf("object type: %T unsupported", data)
		return nil, fmt.Errorf("data type is %T", data)
	}

	rule, ok := message.Content.(*routerv1.Rule)
	if !ok {
		klog.Warningf("object type: %T unsupported", message.Content)
		return nil, fmt.Errorf("message content type should be rule type. operation: %s, resource: %s",
			message.GetOperation(), message.GetResource())
	}

	switch message.GetOperation() {
	case model.InsertOperation:
		addRuleWithRetry(rule)
	case model.DeleteOperation:
		delRule(rule.Namespace, rule.Name)
	default:
		klog.Warningf("invalid message operation.")
	}
	return nil, nil
}

func addRuleEndpoint(ruleEndpoint *routerv1.RuleEndpoint) {
	key := getKey(ruleEndpoint.Namespace, ruleEndpoint.Name)
	ruleEndpoints.Store(key, ruleEndpoint)
	klog.Infof("add ruleendpoint %s success.", key)
}

func deleteRuleEndpoint(namespace, name string) {
	key := getKey(namespace, name)
	ruleEndpoints.Delete(key)
	klog.Infof("delete ruleendpoint %s success.", key)
}

// AddRule add rule
func addRule(rule *routerv1.Rule) error {
	source, err := getSourceOfRule(rule)
	if err != nil {
		klog.Error(err)
		return err
	}
	target, err := getTargetOfRule(rule)
	if err != nil {
		klog.Error(err)
		return err
	}

	ruleKey := getKey(rule.Namespace, rule.Name)
	if err := source.RegisterListener(func(data interface{}) (interface{}, error) {
		//TODO Use goroutine pool later
		var execResult ExecResult
		resp, err := source.Forward(target, data)
		if err != nil {
			// rule.Status.Fail++
			// record error info for rule
			errMsg := ErrorMsg{Detail: err.Error(), Timestamp: time.Now()}
			execResult = ExecResult{RuleID: rule.Name, ProjectID: rule.Namespace, Status: "FAIL", Error: errMsg}
		} else {
			execResult = ExecResult{RuleID: rule.Name, ProjectID: rule.Namespace, Status: "SUCCESS"}
		}
		ResultChannel <- execResult
		return resp, nil
	}); err != nil {
		klog.Errorf("add rule %s failed, err: %v", ruleKey, err)
		errMsg := ErrorMsg{Detail: err.Error(), Timestamp: time.Now()}
		execResult := ExecResult{RuleID: rule.Name, ProjectID: rule.Namespace, Status: "FAIL", Error: errMsg}
		ResultChannel <- execResult
		return nil
	}

	rules.Store(ruleKey, rule)
	klog.Infof("add rule success: %+v", rule)
	return nil
}

// DelRule delete rule by rule id
func delRule(namespace, name string) {
	ruleKey := getKey(namespace, name)
	v, exist := rules.Load(ruleKey)
	if !exist {
		klog.Warningf("rule %s does not exist", ruleKey)
		return
	}
	rule := v.(*routerv1.Rule)

	source, err := getSourceOfRule(rule)
	// if source not exist, skip UnregisterListener
	if err == nil {
		klog.V(4).Infof("delRule: source of rule:%s exist, do UnregisterListener", rule.Spec.Source)
		source.UnregisterListener()
	} else {
		klog.Warningf("delRule: source of rule:%s not exist, unnecessary do UnregisterListener:%v", rule.Spec.Source, err)
	}

	rules.Delete(ruleKey)
	klog.V(4).Infof("delete rule success: %s", ruleKey)
}

func getSourceOfRule(rule *routerv1.Rule) (provider.Source, error) {
	sourceKey := getKey(rule.Namespace, rule.Spec.Source)
	v, exist := ruleEndpoints.Load(sourceKey)
	if !exist {
		return nil, fmt.Errorf("source rule endpoint %s does not existing", sourceKey)
	}

	sourceEp := v.(*routerv1.RuleEndpoint)
	sf, exist := provider.GetSourceFactory(sourceEp.Spec.RuleEndpointType)
	if !exist {
		return nil, fmt.Errorf("source definition %s does not existing", sourceEp.Spec.RuleEndpointType)
	}

	source := sf.GetSource(sourceEp, rule.Spec.SourceResource)
	if source == nil {
		return nil, fmt.Errorf("can't get source: %s", rule.Spec.Source)
	}
	return source, nil
}

func getTargetOfRule(rule *routerv1.Rule) (provider.Target, error) {
	targetKey := getKey(rule.Namespace, rule.Spec.Target)
	v, exist := ruleEndpoints.Load(targetKey)
	if !exist {
		return nil, fmt.Errorf("target rule endpoint %s does not existing", targetKey)
	}

	targetEp := v.(*routerv1.RuleEndpoint)
	tf, exist := provider.GetTargetFactory(targetEp.Spec.RuleEndpointType)
	if !exist {
		return nil, fmt.Errorf("target definition %s does not existing", targetEp.Spec.RuleEndpointType)
	}

	target := tf.GetTarget(targetEp, rule.Spec.TargetResource)
	if target == nil {
		return nil, fmt.Errorf("can't get target: %s", rule.Spec.Target)
	}
	return target, nil
}

func getKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func addRuleWithRetry(rule *routerv1.Rule) {
	retry, waitTime := 3, 5
	for i := 0; i <= retry; i++ {
		if err := addRule(rule); err == nil {
			break
		}
		klog.Errorf("add rule fail, wait to retry. retry time: %d", i+1)
		time.Sleep(time.Duration(waitTime*(i+1)) * time.Second)
	}
}
