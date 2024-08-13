package admissioncontroller

import (
	"encoding/json"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	rulesv1 "github.com/kubeedge/api/apis/rules/v1"
)

func Test_admitRuleEndpoint(t *testing.T) {
	t.Run("create servicebus ruleEndpoint successful", func(t *testing.T) {
		ruleEndpoint := rulesv1.RuleEndpoint{
			TypeMeta: v1.TypeMeta{
				Kind:       "RuleEndpoint",
				APIVersion: "rules.kubeedge.io/v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "servicebus-test",
				Namespace: "test",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeServiceBus,
				Properties: map[string]string{
					"service_port": "9001",
				},
			},
		}
		jsonData, _ := json.Marshal(ruleEndpoint)
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw:    jsonData,
					Object: nil,
				},
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed != true {
			t.Fatalf("create servicebus ruleEndpoint error:%v", admissionResp.Result.Message)
		}
	})

	t.Run("ruleEndpoint data error, create ruleEndpoint failed", func(t *testing.T) {
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw:    []byte{10, 20},
					Object: nil,
				},
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed == true {
			t.Fatalf("create servicebus ruleEndpoint should not success")
		}
	})

	t.Run("service_port not in range 1-65535,create servicebus ruleEndpoint failed", func(t *testing.T) {
		ruleEndpoint := rulesv1.RuleEndpoint{
			TypeMeta: v1.TypeMeta{
				Kind:       "RuleEndpoint",
				APIVersion: "rules.kubeedge.io/v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "servicebus-test",
				Namespace: "test",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeServiceBus,
				Properties: map[string]string{
					"service_port": "69999",
				},
			},
		}
		jsonData, _ := json.Marshal(ruleEndpoint)
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw:    jsonData,
					Object: nil,
				},
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed == true {
			t.Fatalf("create servicebus ruleEndpoint should not success")
		}
	})

	t.Run("service_port not integer,create servicebus ruleEndpoint failed", func(t *testing.T) {
		ruleEndpoint := rulesv1.RuleEndpoint{
			TypeMeta: v1.TypeMeta{
				Kind:       "RuleEndpoint",
				APIVersion: "rules.kubeedge.io/v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "servicebus-test",
				Namespace: "test",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeServiceBus,
				Properties: map[string]string{
					"service_port": "abcd",
				},
			},
		}
		jsonData, _ := json.Marshal(ruleEndpoint)
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw:    jsonData,
					Object: nil,
				},
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed == true {
			t.Fatalf("create servicebus ruleEndpoint should not success")
		}
	})

	t.Run("service_port is nil,create servicebus ruleEndpoint failed", func(t *testing.T) {
		ruleEndpoint := rulesv1.RuleEndpoint{
			TypeMeta: v1.TypeMeta{
				Kind:       "RuleEndpoint",
				APIVersion: "rules.kubeedge.io/v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      "servicebus-test",
				Namespace: "test",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeServiceBus,
				Properties:       map[string]string{},
			},
		}
		jsonData, _ := json.Marshal(ruleEndpoint)
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw:    jsonData,
					Object: nil,
				},
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed == true {
			t.Fatalf("create servicebus ruleEndpoint should not success")
		}
	})

	t.Run("update ruleEndpoint failed", func(t *testing.T) {
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Name:      "servicebus-test",
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed == true {
			t.Fatalf("update ruleEndpoint should not success")
		}
	})

	t.Run("delete ruleEndpoint  successful", func(t *testing.T) {
		admissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Delete,
				Name:      "servicebus-test",
			},
			Response: nil,
		}
		admissionResp := admitRuleEndpoint(admissionReview)
		if admissionResp.Allowed != true {
			t.Fatalf("delete ruleEndpoint error:%v", admissionResp.Result.Message)
		}
	})
}
