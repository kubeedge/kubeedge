package admissioncontroller

import (
	"fmt"
	"net/http"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	rulesv1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
)

var (
	sourceToTarget = [][2]rulesv1.RuleEndpointTypeDef{{rulesv1.RuleEndpointTypeRest, rulesv1.RuleEndpointTypeEventBus},
		{rulesv1.RuleEndpointTypeRest, rulesv1.RuleEndpointTypeServiceBus},
		{rulesv1.RuleEndpointTypeEventBus, rulesv1.RuleEndpointTypeRest}}
)

func admitRule(review admissionv1beta1.AdmissionReview) *admissionv1beta1.AdmissionResponse {
	reviewResponse := admissionv1beta1.AdmissionResponse{}
	var msg string
	switch review.Request.Operation {
	case admissionv1beta1.Create:
		raw := review.Request.Object.Raw
		rule := rulesv1.Rule{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(raw, nil, &rule); err != nil {
			klog.Errorf("validation failed with error: %v", err)
			msg = err.Error()
			break
		}
		err := validateRule(&rule)
		if err != nil {
			msg = err.Error()
			break
		}
		reviewResponse.Allowed = true
	case admissionv1beta1.Delete, admissionv1beta1.Connect:
		//no rule defined for above operations, greenlight for all of above.
		reviewResponse.Allowed = true
		klog.Info("admission validation passed!")
	default:
		klog.Infof("Unsupported webhook operation %v", review.Request.Operation)
		msg = msg + "Unsupported webhook operation!"
	}
	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}

func validateRule(rule *rulesv1.Rule) error {
	sourceKey := fmt.Sprintf("%s/%s", rule.Namespace, rule.Spec.Source)
	sourceEndpoint, err := controller.getRuleEndpoint(rule.Namespace, rule.Spec.Source)
	if err != nil {
		return fmt.Errorf("cant get source ruleEndpoint %s. Reason: %w", sourceKey, err)
	} else if sourceEndpoint == nil {
		return fmt.Errorf("source ruleEndpoint %s has not been created", sourceKey)
	}
	if err = validateSourceRuleEndpoint(sourceEndpoint, rule.Spec.SourceResource); err != nil {
		return err
	}
	targetKey := fmt.Sprintf("%s/%s", rule.Namespace, rule.Spec.Target)
	targetEndpoint, err := controller.getRuleEndpoint(rule.Namespace, rule.Spec.Target)
	if err != nil {
		return fmt.Errorf("cant get target ruleEndpoint %s. Reason: %w", targetKey, err)
	} else if targetEndpoint == nil {
		return fmt.Errorf("target ruleEndpoint %s has not been created", targetKey)
	}
	if err = validateTargetRuleEndpoint(targetEndpoint, rule.Spec.TargetResource); err != nil {
		return err
	}
	var exist bool
	for _, s2t := range sourceToTarget {
		if s2t[0] == sourceEndpoint.Spec.RuleEndpointType && s2t[1] == targetEndpoint.Spec.RuleEndpointType {
			exist = true
			break
		}
	}
	if !exist {
		return fmt.Errorf("the rule which is from source ruleEndpoint type %s to target ruleEndpoint type %s is not validate ",
			sourceEndpoint.Spec.RuleEndpointType, targetEndpoint.Spec.RuleEndpointType)
	}
	return nil
}
func validateSourceRuleEndpoint(ruleEndpoint *rulesv1.RuleEndpoint, sourceResource map[string]string) error {
	switch ruleEndpoint.Spec.RuleEndpointType {
	case rulesv1.RuleEndpointTypeRest:
		_, exist := sourceResource[SourceResourcePathKey]
		if !exist {
			return fmt.Errorf(`%q property missed in sourceResource when ruleEndpoint is "rest"`, SourceResourcePathKey)
		}
		rules, err := controller.listRule(ruleEndpoint.Namespace)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if sourceResource[SourceResourcePathKey] == r.Spec.SourceResource[SourceResourcePathKey] {
				return fmt.Errorf("source properties exist in Rule %s/%s. Path: %s", r.Namespace, r.Name, sourceResource[SourceResourcePathKey])
			}
		}
	case rulesv1.RuleEndpointTypeEventBus:
		_, exist := sourceResource[SourceResourceTopicKey]
		if !exist {
			return fmt.Errorf(`%q property missed in sourceResource when ruleEndpoint is "eventbus"`, SourceResourceTopicKey)
		}
		_, exist = sourceResource[SourceResourceNodeNameKey]
		if !exist {
			return fmt.Errorf("eventbus")
		}
		rules, err := controller.listRule(ruleEndpoint.Namespace)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if sourceResource[SourceResourceTopicKey] == r.Spec.SourceResource[SourceResourceTopicKey] &&
				sourceResource[SourceResourceNodeNameKey] == r.Spec.SourceResource[SourceResourceNodeNameKey] {
				return fmt.Errorf("source properties exist in Rule %s/%s. Node_name: %s, topic: %s", r.Namespace, r.Name,
					sourceResource[SourceResourceNodeNameKey], sourceResource[SourceResourceTopicKey])
			}
		}
	}
	return nil
}

func validateTargetRuleEndpoint(ruleEndpoint *rulesv1.RuleEndpoint, targetResource map[string]string) error {
	switch ruleEndpoint.Spec.RuleEndpointType {
	case rulesv1.RuleEndpointTypeRest:
		_, exist := targetResource[SourceResourceResourceKey]
		if !exist {
			return fmt.Errorf(`%q property missed in targetResource when ruleEndpoint is "rest"`, SourceResourceResourceKey)
		}
	case rulesv1.RuleEndpointTypeEventBus:
		_, exist := targetResource[SourceResourceTopicKey]
		if !exist {
			return fmt.Errorf(`%q property missed in targetResource when ruleEndpoint is "eventbus"`, SourceResourceTopicKey)
		}
	case rulesv1.RuleEndpointTypeServiceBus:
		_, exist := targetResource[SourceResourcePathKey]
		if !exist {
			return fmt.Errorf(`%q property missed in targetResource when ruleEndpoint is "servicebus"`, SourceResourcePathKey)
		}
	}
	return nil
}

func serveRule(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitRule)
}
