package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rulesv1 "github.com/kubeedge/kubeedge/pkg/apis/rules/v1"
)

func NewRule(sourceType, targetType rulesv1.RuleEndpointTypeDef) *rulesv1.Rule {
	switch {
	case sourceType == rulesv1.RuleEndpointTypeRest && targetType == rulesv1.RuleEndpointTypeEventBus:
		return NewRest2EventbusRule()
	case sourceType == rulesv1.RuleEndpointTypeEventBus && targetType == rulesv1.RuleEndpointTypeRest:
		return NewEventbus2RestRule()
	case sourceType == rulesv1.RuleEndpointTypeRest && targetType == rulesv1.RuleEndpointTypeServiceBus:
		return NewRest2ServicebusRule()
	case sourceType == rulesv1.RuleEndpointTypeServiceBus && targetType == rulesv1.RuleEndpointTypeRest:
		return NewServicebus2Rest()
	}
	return nil
}

func NewEventbus2RestRule() *rulesv1.Rule {
	rule := rulesv1.Rule{
		TypeMeta: v1.TypeMeta{
			Kind:       "Rule",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "rule-eventbus-rest-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleSpec{
			Source: "eventbus-test",
			SourceResource: map[string]string{
				"topic":     "test",
				"node_name": "edge-node",
			},
			Target: "rest-test",
			TargetResource: map[string]string{
				"resource": "http://127.0.0.1:9000/echo",
			},
		},
		Status: rulesv1.RuleStatus{
			Errors: []string{},
		},
	}
	return &rule
}

func NewRest2EventbusRule() *rulesv1.Rule {
	rule := rulesv1.Rule{
		TypeMeta: v1.TypeMeta{
			Kind:       "Rule",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "rule-rest-eventbus-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleSpec{
			Source: "rest-test",
			SourceResource: map[string]string{
				"path": "/ccc",
			},
			Target: "eventbus-test",
			TargetResource: map[string]string{
				"topic": "topic-test",
			},
		},
		Status: rulesv1.RuleStatus{
			Errors: []string{},
		},
	}
	return &rule
}

func NewRest2ServicebusRule() *rulesv1.Rule {
	rule := rulesv1.Rule{
		TypeMeta: v1.TypeMeta{
			Kind:       "Rule",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "rule-rest-servicebus-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleSpec{
			Source: "rest-test",
			SourceResource: map[string]string{
				"path": "/ddd",
			},
			Target: "servicebus-test",
			TargetResource: map[string]string{
				"path": "/url",
			},
		},
		Status: rulesv1.RuleStatus{
			Errors: []string{},
		},
	}
	return &rule
}

func NewServicebus2Rest() *rulesv1.Rule {
	rule := rulesv1.Rule{
		TypeMeta: v1.TypeMeta{
			Kind:       "Rule",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "rule-servicebus-rest-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleSpec{
			Source: "servicebus-test",
			SourceResource: map[string]string{
				"target_url": "http://127.0.0.1:9000/echo",
				"node_name":  "edge-node",
			},
			Target: "rest-test",
			TargetResource: map[string]string{
				"resource": "http://127.0.0.1:9000/echo",
			},
		},
		Status: rulesv1.RuleStatus{
			Errors: []string{},
		},
	}
	return &rule
}

func NewRuleEndpoint(endpointType rulesv1.RuleEndpointTypeDef) *rulesv1.RuleEndpoint {
	switch endpointType {
	case rulesv1.RuleEndpointTypeRest:
		return newRestRuleEndpoint()
	case rulesv1.RuleEndpointTypeEventBus:
		return newEventBusRuleEndpoint()
	case rulesv1.RuleEndpointTypeServiceBus:
		return newServiceBusRuleEndpoint()
	}
	return newRestRuleEndpoint()
}

func newRestRuleEndpoint() *rulesv1.RuleEndpoint {
	restRuleEndpoint := rulesv1.RuleEndpoint{
		TypeMeta: v1.TypeMeta{
			Kind:       "RuleEndpoint",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "rest-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeRest,
		},
	}
	return &restRuleEndpoint
}

func newEventBusRuleEndpoint() *rulesv1.RuleEndpoint {
	eventbusRuleEndpoint := rulesv1.RuleEndpoint{
		TypeMeta: v1.TypeMeta{
			Kind:       "RuleEndpoint",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "eventbus-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeEventBus,
		},
	}
	return &eventbusRuleEndpoint
}

func newServiceBusRuleEndpoint() *rulesv1.RuleEndpoint {
	servicebusRuleEndpoint := rulesv1.RuleEndpoint{
		TypeMeta: v1.TypeMeta{
			Kind:       "RuleEndpoint",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "servicebus-test",
			Namespace: Namespace,
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeServiceBus,
			Properties: map[string]string{
				"service_port": "9000"},
		},
	}
	return &servicebusRuleEndpoint
}

// GetRuleList to get the rule list and verify whether the contents of the rule matches with what is expected
func GetRuleList(list *rulesv1.RuleList, getRuleAPI string, expectedRule *rulesv1.Rule) ([]rulesv1.Rule, error) {
	resp, err := SendHTTPRequest(http.MethodGet, getRuleAPI)
	defer resp.Body.Close()
	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		Fatalf("HTTP Response reading has failed: %v", err)
		return nil, err
	}
	err = json.Unmarshal(contents, &list)
	if err != nil {
		Fatalf("Unmarshal HTTP Response has failed: %v", err)
		return nil, err
	}
	if expectedRule != nil {
		modelExists := false
		for _, rule := range list.Items {
			if expectedRule.ObjectMeta.Name == rule.ObjectMeta.Name {
				modelExists = true
				if !reflect.DeepEqual(expectedRule.TypeMeta, rule.TypeMeta) ||
					expectedRule.ObjectMeta.Namespace != rule.ObjectMeta.Namespace ||
					!reflect.DeepEqual(expectedRule.Spec, rule.Spec) {
					return nil, errors.New("The rule is not matching with what was expected")
				}
			}
		}
		if !modelExists {
			return nil, errors.New("The requested rule is not found")
		}
	}
	return list.Items, nil
}

// HandleRule to handle rule.
func HandleRule(operation, apiserver, UID string, sourceType, targetType rulesv1.RuleEndpointTypeDef) (bool, int) {
	var req *http.Request
	var err error
	var body io.Reader

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}

	switch operation {
	case http.MethodPost:
		body := NewRule(sourceType, targetType)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Fatalf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
		req.Header.Set("Content-Type", "application/json")
	case http.MethodDelete:
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		// handle error
		Fatalf("Frame HTTP request failed: %v", err)
		return false, 0
	}
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Fatalf("HTTP request is failed :%v", err)
		return false, 0
	}
	defer resp.Body.Close()
	contents, err := io.ReadAll(resp.Body)
	Infof("%s %s %v  %v in %v", req.Method, req.URL, resp.Status, string(contents), time.Since(t))
	return true, resp.StatusCode
}

// HandleRuleEndpoint to handle ruleendpoint.
func HandleRuleEndpoint(operation string, apiserver string, UID string, endpointType rulesv1.RuleEndpointTypeDef) (bool, int) {
	var req *http.Request
	var err error
	var body io.Reader

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}

	switch operation {
	case http.MethodPost:
		body := NewRuleEndpoint(endpointType)
		respBytes, err := json.Marshal(body)
		if err != nil {
			Fatalf("Marshalling body failed: %v", err)
		}
		req, err = http.NewRequest(http.MethodPost, apiserver, bytes.NewBuffer(respBytes))
		req.Header.Set("Content-Type", "application/json")
	case http.MethodDelete:
		req, err = http.NewRequest(http.MethodDelete, apiserver+UID, body)
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		// handle error
		Fatalf("Frame HTTP request failed: %v", err)
		return false, 0
	}
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		Fatalf("HTTP request is failed :%v", err)
		return false, 0
	}
	defer resp.Body.Close()
	Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	return true, resp.StatusCode
}

// GetRuleEndpointList to get the rule endpoint list and verify whether the contents of the ruleendpoint matches with what is expected
func GetRuleEndpointList(list *rulesv1.RuleEndpointList, getRuleEndpointAPI string, expectedRule *rulesv1.RuleEndpoint) ([]rulesv1.RuleEndpoint, error) {
	resp, err := SendHTTPRequest(http.MethodGet, getRuleEndpointAPI)
	defer resp.Body.Close()
	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		Fatalf("HTTP Response reading has failed: %v", err)
		return nil, err
	}
	err = json.Unmarshal(contents, &list)
	if err != nil {
		Fatalf("Unmarshal HTTP Response has failed: %v", err)
		return nil, err
	}
	if expectedRule != nil {
		exists := false
		for _, ruleEndpoint := range list.Items {
			if expectedRule.ObjectMeta.Name == ruleEndpoint.ObjectMeta.Name {
				exists = true
				if !reflect.DeepEqual(expectedRule.TypeMeta, ruleEndpoint.TypeMeta) ||
					expectedRule.ObjectMeta.Namespace != ruleEndpoint.ObjectMeta.Namespace ||
					!reflect.DeepEqual(expectedRule.Spec, ruleEndpoint.Spec) {
					return nil, errors.New("The ruleendpoint is not matching with what was expected")
				}
			}
		}
		if !exists {
			return nil, errors.New("The requested ruleendpoint is not found")
		}
	}
	return list.Items, nil
}
