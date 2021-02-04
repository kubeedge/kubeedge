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
	// TODO: check rule endpoint type whether is cloud or edge
	if targetEndpoint.Spec.RuleEndpointType == sourceEndpoint.Spec.RuleEndpointType {
		return fmt.Errorf("target ruleEndpoint type %s can not be the same with source ruleEndpoint type", targetEndpoint.Spec.RuleEndpointType)
	}
	if err = validateTargetRuleEndpoint(targetEndpoint, rule.Spec.TargetResource); err != nil {
		return err
	}
	return nil
}
func validateSourceRuleEndpoint(ruleEndpoint *rulesv1.RuleEndpoint, sourceResource map[string]string) error {
	switch ruleEndpoint.Spec.RuleEndpointType {
	case "rest":
		_, exist := sourceResource["path"]
		if !exist {
			return fmt.Errorf("\"path\" property missed in sourceResource when ruleEndpoint is \"rest\"")
		}
		rules, err := controller.listRule(ruleEndpoint.Namespace)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if sourceResource["path"] == r.Spec.SourceResource["path"] {
				return fmt.Errorf("source properties exist in Rule %s/%s. Path: %s", r.Namespace, r.Name, sourceResource["path"])
			}
		}
	case "eventbus":
		_, exist := sourceResource["topic"]
		if !exist {
			return fmt.Errorf("\"topic\" property missed in sourceResource when ruleEndpoint is \"eventbus\"")
		}
		_, exist = sourceResource["node_name"]
		if !exist {
			return fmt.Errorf("eventbus")
		}
		rules, err := controller.listRule(ruleEndpoint.Namespace)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if sourceResource["topic"] == r.Spec.SourceResource["topic"] && sourceResource["node_name"] == r.Spec.SourceResource["node_name"] {
				return fmt.Errorf("source properties exist in Rule %s/%s. Node_name: %s, topic: %s", r.Namespace, r.Name, sourceResource["node_name"], sourceResource["topic"])
			}
		}
	}
	return nil
}

func validateTargetRuleEndpoint(ruleEndpoint *rulesv1.RuleEndpoint, targetResource map[string]string) error {
	switch ruleEndpoint.Spec.RuleEndpointType {
	case "rest":
		_, exist := targetResource["resource"]
		if !exist {
			return fmt.Errorf("\"resource\" property missed in targetResource when ruleEndpoint is \"rest\"")
		}
	case "eventbus":
		_, exist := targetResource["topic"]
		if !exist {
			return fmt.Errorf("\"topic\" property missed in targetResource when ruleEndpoint is \"eventbus\"")
		}
	}
	return nil
}

func serveRule(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitRule)
}
