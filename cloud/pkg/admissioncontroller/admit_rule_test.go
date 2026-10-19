/*
Copyright 2026 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admissioncontroller

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	rulesv1 "github.com/kubeedge/api/apis/rules/v1"
	"github.com/kubeedge/api/client/clientset/versioned/fake"
)

const ruleTestNamespace = "test"

// setupCrdClient points the package level controller at a fake clientset preloaded with objects,
// and restores the previous client when the test ends.
func setupCrdClient(t *testing.T, objects ...runtime.Object) {
	t.Helper()
	original := controller.CrdClient
	controller.CrdClient = fake.NewSimpleClientset(objects...)
	t.Cleanup(func() {
		controller.CrdClient = original
	})
}

func newRuleEndpoint(name string, endpointType rulesv1.RuleEndpointTypeDef) *rulesv1.RuleEndpoint {
	return &rulesv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ruleTestNamespace,
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: endpointType,
		},
	}
}

// newRestSourceRule builds a rest -> eventbus rule, the shape used by most of the cases below.
func newRestSourceRule(name, path string) *rulesv1.Rule {
	return &rulesv1.Rule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Rule",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ruleTestNamespace,
		},
		Spec: rulesv1.RuleSpec{
			Source:         "rest-test",
			SourceResource: map[string]string{"path": path},
			Target:         "eventbus-test",
			TargetResource: map[string]string{"topic": "test-topic"},
		},
	}
}

// newEventBusSourceRule builds an eventbus -> rest rule.
func newEventBusSourceRule(name, topic, nodeName string) *rulesv1.Rule {
	return &rulesv1.Rule{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Rule",
			APIVersion: "rules.kubeedge.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ruleTestNamespace,
		},
		Spec: rulesv1.RuleSpec{
			Source:         "eventbus-test",
			SourceResource: map[string]string{"topic": topic, "node_name": nodeName},
			Target:         "rest-test",
			TargetResource: map[string]string{"resource": "http://127.0.0.1:8080"},
		},
	}
}

func TestAdmitRule(t *testing.T) {
	restEndpoint := newRuleEndpoint("rest-test", rulesv1.RuleEndpointTypeRest)
	eventBusEndpoint := newRuleEndpoint("eventbus-test", rulesv1.RuleEndpointTypeEventBus)

	testCases := []struct {
		name string
		// objects preloaded into the fake clientset, the cluster state the webhook validates against.
		objects   []runtime.Object
		operation admissionv1.Operation
		// rule is marshalled into the admission request. raw takes precedence when set.
		rule            *rulesv1.Rule
		raw             []byte
		expectedAllowed bool
		expectedError   string
	}{
		{
			name:            "create rule successful",
			objects:         []runtime.Object{restEndpoint, eventBusEndpoint},
			operation:       admissionv1.Create,
			rule:            newRestSourceRule("rule-test", "/a/b"),
			expectedAllowed: true,
		},
		{
			name: "create rule whose source path is taken failed",
			objects: []runtime.Object{
				restEndpoint, eventBusEndpoint,
				newRestSourceRule("rule-other", "/a/b"),
			},
			operation:       admissionv1.Create,
			rule:            newRestSourceRule("rule-test", "/a/b"),
			expectedAllowed: false,
			expectedError:   "source properties exist in Rule",
		},
		{
			// Regression test: the stored copy of the rule being updated is returned by the
			// list and must not be treated as a conflicting rule.
			name: "update rule successful",
			objects: []runtime.Object{
				restEndpoint, eventBusEndpoint,
				newRestSourceRule("rule-test", "/a/b"),
			},
			operation:       admissionv1.Update,
			rule:            newRestSourceRule("rule-test", "/a/b"),
			expectedAllowed: true,
		},
		{
			name: "update eventbus source rule successful",
			objects: []runtime.Object{
				restEndpoint, eventBusEndpoint,
				newEventBusSourceRule("rule-test", "test-topic", "edge-node"),
			},
			operation:       admissionv1.Update,
			rule:            newEventBusSourceRule("rule-test", "test-topic", "edge-node"),
			expectedAllowed: true,
		},
		{
			name: "update rule whose source path is taken by another rule failed",
			objects: []runtime.Object{
				restEndpoint, eventBusEndpoint,
				newRestSourceRule("rule-test", "/a/b"),
				newRestSourceRule("rule-other", "/c/d"),
			},
			operation:       admissionv1.Update,
			rule:            newRestSourceRule("rule-test", "/c/d"),
			expectedAllowed: false,
			expectedError:   "source properties exist in Rule",
		},
		{
			name:            "update rule with unknown source ruleEndpoint failed",
			objects:         []runtime.Object{eventBusEndpoint},
			operation:       admissionv1.Update,
			rule:            newRestSourceRule("rule-test", "/a/b"),
			expectedAllowed: false,
			expectedError:   "can't get source ruleEndpoint",
		},
		{
			name:            "rule data error, update rule failed",
			objects:         []runtime.Object{restEndpoint, eventBusEndpoint},
			operation:       admissionv1.Update,
			raw:             []byte{10, 20},
			expectedAllowed: false,
		},
		{
			name:            "delete rule successful",
			operation:       admissionv1.Delete,
			expectedAllowed: true,
		},
		{
			name:            "connect rule successful",
			operation:       admissionv1.Connect,
			expectedAllowed: true,
		},
		{
			name:            "unsupported operation failed",
			operation:       admissionv1.Operation("UNKNOWN"),
			expectedAllowed: false,
			expectedError:   "unsupported webhook operation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setupCrdClient(t, tc.objects...)

			raw := tc.raw
			if raw == nil && tc.rule != nil {
				var err error
				raw, err = json.Marshal(tc.rule)
				assert.NoError(t, err)
			}

			admissionResp := admitRule(admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: tc.operation,
					Object: runtime.RawExtension{
						Raw: raw,
					},
				},
			})

			assert.Equal(t, tc.expectedAllowed, admissionResp.Allowed)
			if tc.expectedAllowed {
				return
			}
			if assert.NotNil(t, admissionResp.Result) && tc.expectedError != "" {
				assert.Contains(t, admissionResp.Result.Message, tc.expectedError)
			}
		})
	}
}

// TestAdmitRuleUpdateIsValidated guards the operation dispatch itself: an update carrying an
// invalid rule must fail validation rather than be rejected as an unsupported operation.
func TestAdmitRuleUpdateIsValidated(t *testing.T) {
	setupCrdClient(t, newRuleEndpoint("rest-test", rulesv1.RuleEndpointTypeRest))

	rule := newRestSourceRule("rule-test", "/a/b")
	rule.Spec.SourceResource = map[string]string{}
	raw, err := json.Marshal(rule)
	assert.NoError(t, err)

	admissionResp := admitRule(admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Update,
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
	})

	assert.False(t, admissionResp.Allowed)
	assert.NotNil(t, admissionResp.Result)
	assert.Contains(t, admissionResp.Result.Message, "\"path\" property missed in sourceResource")
}
