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

var rules sync.Map
var ruleEndpoints sync.Map

func init() {
	registerListener()
}

func registerListener() {
	eKey := fmt.Sprintf("%s/%s", modules.EdgeControllerModuleName, model.ResourceTypeRuleEndpoint)
	listener.MessageHandlerInstance.AddListener(eKey, func(data interface{}) (interface{}, error) {
		message := data.(*model.Message)
		ruleEndpoint, ok := message.GetContent().(*routerv1.RuleEndpoint)
		if !ok {
			klog.Warningf("object type: %T unsupported", ruleEndpoint)
			return nil, fmt.Errorf("message content type should be ruleEndpoint type. operation: %s, resource: %s",
				message.GetOperation(), message.GetResource())
		}
		if message.GetOperation() == model.InsertOperation {
			addRuleEndpoint(ruleEndpoint)
		} else if message.GetOperation() == model.DeleteOperation {
			deleteRuleEndpoint(ruleEndpoint.Namespace, ruleEndpoint.Name)
		} else {
			klog.Warningf("invalid message operation.")
		}
		return nil, nil
	})
	rKey := fmt.Sprintf("%s/%s", modules.EdgeControllerModuleName, model.ResourceTypeRule)
	listener.MessageHandlerInstance.AddListener(rKey, func(data interface{}) (interface{}, error) {
		message := data.(*model.Message)
		rule, ok := message.GetContent().(*routerv1.Rule)
		if !ok {
			klog.Warningf("object type: %T unsupported", rule)
			return nil, fmt.Errorf("message content type should be rule type. operation: %s, resource: %s",
				message.GetOperation(), message.GetResource())
		}
		if message.GetOperation() == model.InsertOperation {
			addRuleWithRetry(rule)
		} else if message.GetOperation() == model.DeleteOperation {
			delRule(rule.Namespace, rule.Name)
		} else {
			klog.Warningf("invalid message operation.")
		}
		return nil, nil
	})
}

func addRuleEndpoint(ruleEndpoint *routerv1.RuleEndpoint) {
	ruleEndpoints.Store(getKey(ruleEndpoint.Namespace, ruleEndpoint.Name), ruleEndpoint)
	klog.Infof("add ruleendpoint %s/%s success.", ruleEndpoint.Namespace, ruleEndpoint.Name)
}

func deleteRuleEndpoint(namespace, name string) {
	ruleEndpoints.Delete(getKey(namespace, name))
	klog.Infof("delete ruleendpoint %s/%s success.", namespace, name)
}

// AddRule add rule
func addRule(rule *routerv1.Rule) error {
	v, exist := ruleEndpoints.Load(getKey(rule.Namespace, rule.Spec.Source))
	if !exist {
		err := fmt.Errorf("source rule endpoint %s/%s does not existing", rule.Namespace, rule.Spec.Source)
		klog.Error(err.Error())
		return err
	}
	sourceEp := v.(*routerv1.RuleEndpoint)
	sf, exist := provider.GetSourceFactory(sourceEp.Spec.RuleEndpointType)
	if !exist {
		err := fmt.Errorf("source definition %s does not existing", sourceEp.Spec.RuleEndpointType)
		klog.Error(err.Error())
		return err
	}
	v, exist = ruleEndpoints.Load(getKey(rule.Namespace, rule.Spec.Target))
	if !exist {
		err := fmt.Errorf("target rule endpoint %s/%s does not existing", rule.Namespace, rule.Spec.Target)
		klog.Error(err.Error())
		return err
	}
	targetEp := v.(*routerv1.RuleEndpoint)
	tf, exist := provider.GetTargetFactory(targetEp.Spec.RuleEndpointType)
	if !exist {
		err := fmt.Errorf("target definition %s does not existing", targetEp.Spec.RuleEndpointType)
		klog.Error(err.Error())
		return err
	}
	source := sf.GetSource(sourceEp, rule.Spec.SourceResource)
	if source == nil {
		err := fmt.Errorf("can't get source: %+v", rule.Spec.Source)
		klog.Error(err.Error())
		return err
	}

	target := tf.GetTarget(targetEp, rule.Spec.TargetResource)
	if target == nil {
		err := fmt.Errorf("can't get target: %+v", rule.Spec.Target)
		klog.Error(err.Error())
		return err
	}

	err := source.RegisterListener(func(d interface{}) (interface{}, error) {
		//TODO Use goroutine pool later
		resp, err := source.Forward(target, d)
		var execResult ExecResult
		if err != nil {
			//rule.Status.Fail++
			// record error info for rule
			errMsg := ErrorMsg{Detail: err.Error(), Timestamp: time.Now()}
			execResult = ExecResult{RuleID: rule.Name, ProjectID: rule.Namespace, Status: "FAIL", Error: errMsg}
		} else {
			execResult = ExecResult{RuleID: rule.Name, ProjectID: rule.Namespace, Status: "SUCCESS"}
		}
		ResultChannel <- execResult
		return resp, nil
	})
	if err != nil {
		klog.Errorf("add rule %s:%s failed, err: %v", rule.Name, rule.Name, err)
		errMsg := ErrorMsg{Detail: err.Error(), Timestamp: time.Now()}
		execResult := ExecResult{RuleID: rule.Name, ProjectID: rule.Namespace, Status: "FAIL", Error: errMsg}
		ResultChannel <- execResult
		return nil
	}

	rules.Store(getKey(rule.Namespace, rule.Name), rule)
	klog.Infof("add rule success: %+v", rule)
	return nil
}

// DelRule delete rule by rule id
func delRule(namespace, name string) {
	key := getKey(namespace, name)
	v, exist := rules.Load(key)
	if !exist {
		klog.Warningf("rule %s does not exist", key)
		return
	}
	rule := v.(*routerv1.Rule)
	v, exist = ruleEndpoints.Load(getKey(namespace, rule.Spec.Source))
	if !exist {
		klog.Warningf("ruleEndpoint does not exist. namespace: %s, sourceType: %s", namespace, rule.Spec.Source)
		return
	}
	sourceEp := v.(*routerv1.RuleEndpoint)
	// source UnregisterListener
	sf, exist := provider.GetSourceFactory(sourceEp.Spec.RuleEndpointType)
	if !exist {
		klog.Warningf("source definition %s does not existing", sourceEp.Spec.RuleEndpointType)
		return
	}
	source := sf.GetSource(sourceEp, rule.Spec.SourceResource)
	if source == nil {
		klog.Errorf("can't get source: %s", rule.Spec.Source)
		return
	}
	source.UnregisterListener()
	rules.Delete(key)
	klog.Infof("delete rule success: %s", key)
}

func getKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func addRuleWithRetry(rule *routerv1.Rule) {
	retry := 3
	waitTime := 5
	for i := 0; i <= retry; i++ {
		err := addRule(rule)
		if err != nil {
			klog.Errorf("add rule fail,wait to retry. retry time: %d", i+1)
			time.Sleep(time.Duration(waitTime*(i+1)) * time.Second)
		} else {
			break
		}
	}
}
