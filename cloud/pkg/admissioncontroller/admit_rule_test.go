/*
Copyright 2019 The KubeEdge Authors.

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
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	rulesv1 "github.com/kubeedge/api/apis/rules/v1"
)

func TestInterfaceMethods(t *testing.T) {
	skipInternalCall = true
	defer func() { skipInternalCall = false }()

	ac := &AdmissionController{}

	// These calls should use the skipInternalCall path
	_, err1 := ac.GetRuleEndpoint("test", "test")
	if err1 != errTest {
		t.Errorf("Expected test error, got: %v", err1)
	}

	_, err2 := ac.ListRule("test")
	if err2 != errTest {
		t.Errorf("Expected test error, got: %v", err2)
	}
}

// SimpleRuleEndpointGetter implements RuleEndpointGetter for testing
type SimpleRuleEndpointGetter struct {
	GetRuleEndpointFunc func(namespace, name string) (*rulesv1.RuleEndpoint, error)
	ListRuleFunc        func(namespace string) ([]rulesv1.Rule, error)
}

func (m *SimpleRuleEndpointGetter) GetRuleEndpoint(namespace, name string) (*rulesv1.RuleEndpoint, error) {
	return m.GetRuleEndpointFunc(namespace, name)
}

func (m *SimpleRuleEndpointGetter) ListRule(namespace string) ([]rulesv1.Rule, error) {
	return m.ListRuleFunc(namespace)
}

func TestAdmitRuleNonCreate(t *testing.T) {
	testCases := []struct {
		name          string
		operation     admissionv1.Operation
		expectAllowed bool
	}{
		{
			name:          "delete operation",
			operation:     admissionv1.Delete,
			expectAllowed: true,
		},
		{
			name:          "connect operation",
			operation:     admissionv1.Connect,
			expectAllowed: true,
		},
		{
			name:          "update operation",
			operation:     admissionv1.Update,
			expectAllowed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			review := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Operation: tc.operation,
				},
			}

			response := admitRule(review)
			assert.Equal(t, tc.expectAllowed, response.Allowed)
			if !tc.expectAllowed {
				assert.Contains(t, response.Result.Message, "unsupported webhook operation")
			}
		})
	}
}

func TestAdmitRuleDecodeError(t *testing.T) {
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: []byte("invalid json"),
			},
		},
	}

	response := admitRule(review)
	assert.False(t, response.Allowed)
	assert.NotNil(t, response.Result)
	assert.Contains(t, response.Result.Message, "couldn't get version/kind")
}

func TestAdmitRuleCreateWithSourceEndpointNotFound(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{
		GetRuleEndpointFunc: func(namespace, name string) (*rulesv1.RuleEndpoint, error) {
			return nil, nil
		},
	}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	rule := rulesv1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rule",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleSpec{
			Source:         "source-endpoint",
			Target:         "target-endpoint",
			SourceResource: map[string]string{},
			TargetResource: map[string]string{},
		},
	}

	rawRule, err := json.Marshal(rule)
	assert.NoError(t, err)

	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: rawRule,
			},
		},
	}

	response := admitRule(review)

	assert.False(t, response.Allowed)
	assert.NotNil(t, response.Result)
	assert.Contains(t, response.Result.Message, "source ruleEndpoint test-ns/source-endpoint has not been created")
}

func TestAdmitRuleCreateWithSourceEndpointError(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{
		GetRuleEndpointFunc: func(namespace, name string) (*rulesv1.RuleEndpoint, error) {
			return nil, errors.New("connection error")
		},
	}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	rule := rulesv1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rule",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleSpec{
			Source:         "source-endpoint",
			Target:         "target-endpoint",
			SourceResource: map[string]string{},
			TargetResource: map[string]string{},
		},
	}

	rawRule, err := json.Marshal(rule)
	assert.NoError(t, err)

	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: rawRule,
			},
		},
	}

	response := admitRule(review)

	assert.False(t, response.Allowed)
	assert.NotNil(t, response.Result)
	assert.Contains(t, response.Result.Message, "cant get source ruleEndpoint test-ns/source-endpoint")
	assert.Contains(t, response.Result.Message, "connection error")
}

func TestValidateSourceRuleEndpointRESTMissingPath(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	endpoint := &rulesv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeRest,
		},
	}

	sourceResource := map[string]string{}
	err := validateSourceRuleEndpoint(endpoint, sourceResource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "\"path\" property missed in sourceResource")
}

func TestValidateSourceRuleEndpointRESTDuplicate(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{
		ListRuleFunc: func(namespace string) ([]rulesv1.Rule, error) {
			return []rulesv1.Rule{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-rule",
						Namespace: "test-ns",
					},
					Spec: rulesv1.RuleSpec{
						SourceResource: map[string]string{
							"path": "/test-path",
						},
					},
				},
			}, nil
		},
	}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	endpoint := &rulesv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeRest,
		},
	}

	sourceResource := map[string]string{"path": "/test-path"}
	err := validateSourceRuleEndpoint(endpoint, sourceResource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source properties exist in Rule test-ns/existing-rule")
	assert.Contains(t, err.Error(), "Path: /test-path")
}

func TestValidateSourceRuleEndpointEventBusMissingTopic(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	endpoint := &rulesv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeEventBus,
		},
	}

	sourceResource := map[string]string{}
	err := validateSourceRuleEndpoint(endpoint, sourceResource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "\"topic\" property missed in sourceResource")
}

func TestValidateSourceRuleEndpointEventBusMissingNodeName(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	endpoint := &rulesv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeEventBus,
		},
	}

	sourceResource := map[string]string{"topic": "test-topic"}
	err := validateSourceRuleEndpoint(endpoint, sourceResource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "\"node_name\" property missed in sourceResource")
}

func TestValidateSourceRuleEndpointEventBusDuplicate(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{
		ListRuleFunc: func(namespace string) ([]rulesv1.Rule, error) {
			return []rulesv1.Rule{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-rule",
						Namespace: "test-ns",
					},
					Spec: rulesv1.RuleSpec{
						SourceResource: map[string]string{
							"topic":     "test-topic",
							"node_name": "test-node",
						},
					},
				},
			}, nil
		},
	}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	endpoint := &rulesv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint",
			Namespace: "test-ns",
		},
		Spec: rulesv1.RuleEndpointSpec{
			RuleEndpointType: rulesv1.RuleEndpointTypeEventBus,
		},
	}

	sourceResource := map[string]string{
		"topic":     "test-topic",
		"node_name": "test-node",
	}
	err := validateSourceRuleEndpoint(endpoint, sourceResource)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source properties exist in Rule test-ns/existing-rule")
	assert.Contains(t, err.Error(), "Node_name: test-node")
	assert.Contains(t, err.Error(), "topic: test-topic")
}

func TestValidateSourceRuleEndpointListRuleError(t *testing.T) {
	mockCtrl := &SimpleRuleEndpointGetter{
		ListRuleFunc: func(namespace string) ([]rulesv1.Rule, error) {
			return []rulesv1.Rule{}, errors.New("list error")
		},
	}

	restore := SetControllerForTesting(mockCtrl)
	defer restore()

	t.Run("REST endpoint ListRule error", func(t *testing.T) {
		endpoint := &rulesv1.RuleEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-endpoint",
				Namespace: "test-ns",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeRest,
			},
		}

		sourceResource := map[string]string{"path": "/test-path"}
		err := validateSourceRuleEndpoint(endpoint, sourceResource)
		assert.Error(t, err)
		assert.Equal(t, "list error", err.Error())
	})
}

func TestValidateTargetRuleEndpoint(t *testing.T) {
	testCases := []struct {
		name          string
		endpointType  rulesv1.RuleEndpointTypeDef
		targetRes     map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name:          "rest endpoint missing resource",
			endpointType:  rulesv1.RuleEndpointTypeRest,
			targetRes:     map[string]string{},
			expectError:   true,
			errorContains: "resource",
		},
		{
			name:         "rest endpoint with resource",
			endpointType: rulesv1.RuleEndpointTypeRest,
			targetRes:    map[string]string{"resource": "/api/v1/nodes"},
			expectError:  false,
		},
		{
			name:          "eventbus endpoint missing topic",
			endpointType:  rulesv1.RuleEndpointTypeEventBus,
			targetRes:     map[string]string{},
			expectError:   true,
			errorContains: "topic",
		},
		{
			name:         "eventbus endpoint with topic",
			endpointType: rulesv1.RuleEndpointTypeEventBus,
			targetRes:    map[string]string{"topic": "test-topic"},
			expectError:  false,
		},
		{
			name:          "servicebus endpoint missing path",
			endpointType:  rulesv1.RuleEndpointTypeServiceBus,
			targetRes:     map[string]string{},
			expectError:   true,
			errorContains: "path",
		},
		{
			name:         "servicebus endpoint with path",
			endpointType: rulesv1.RuleEndpointTypeServiceBus,
			targetRes:    map[string]string{"path": "/target"},
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := &rulesv1.RuleEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "test-ns",
				},
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: tc.endpointType,
				},
			}

			err := validateTargetRuleEndpoint(endpoint, tc.targetRes)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRuleEndpointCompatibility(t *testing.T) {
	testCases := []struct {
		name           string
		sourceType     rulesv1.RuleEndpointTypeDef
		targetType     rulesv1.RuleEndpointTypeDef
		expectedResult bool
	}{
		{
			name:           "REST to EventBus - valid",
			sourceType:     rulesv1.RuleEndpointTypeRest,
			targetType:     rulesv1.RuleEndpointTypeEventBus,
			expectedResult: true,
		},
		{
			name:           "REST to ServiceBus - valid",
			sourceType:     rulesv1.RuleEndpointTypeRest,
			targetType:     rulesv1.RuleEndpointTypeServiceBus,
			expectedResult: true,
		},
		{
			name:           "EventBus to REST - valid",
			sourceType:     rulesv1.RuleEndpointTypeEventBus,
			targetType:     rulesv1.RuleEndpointTypeRest,
			expectedResult: true,
		},
		{
			name:           "EventBus to EventBus - invalid",
			sourceType:     rulesv1.RuleEndpointTypeEventBus,
			targetType:     rulesv1.RuleEndpointTypeEventBus,
			expectedResult: false,
		},
		{
			name:           "ServiceBus to EventBus - invalid",
			sourceType:     rulesv1.RuleEndpointTypeServiceBus,
			targetType:     rulesv1.RuleEndpointTypeEventBus,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sourceEndpoint := &rulesv1.RuleEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "source-endpoint",
					Namespace: "test-ns",
				},
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: tc.sourceType,
				},
			}

			targetEndpoint := &rulesv1.RuleEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "target-endpoint",
					Namespace: "test-ns",
				},
				Spec: rulesv1.RuleEndpointSpec{
					RuleEndpointType: tc.targetType,
				},
			}

			mockCtrl := &SimpleRuleEndpointGetter{
				GetRuleEndpointFunc: func(namespace, name string) (*rulesv1.RuleEndpoint, error) {
					if name == "source-endpoint" {
						return sourceEndpoint, nil
					}
					return targetEndpoint, nil
				},
				ListRuleFunc: func(namespace string) ([]rulesv1.Rule, error) {
					return []rulesv1.Rule{}, nil
				},
			}

			restore := SetControllerForTesting(mockCtrl)
			defer restore()

			sourceResource := map[string]string{}
			targetResource := map[string]string{}

			switch tc.sourceType {
			case rulesv1.RuleEndpointTypeRest:
				sourceResource["path"] = "/test-path"
			case rulesv1.RuleEndpointTypeEventBus:
				sourceResource["topic"] = "test-topic"
				sourceResource["node_name"] = "test-node"
			case rulesv1.RuleEndpointTypeServiceBus:
				sourceResource["path"] = "/test-path"
			}

			switch tc.targetType {
			case rulesv1.RuleEndpointTypeRest:
				targetResource["resource"] = "/api/v1/resource"
			case rulesv1.RuleEndpointTypeEventBus:
				targetResource["topic"] = "target-topic"
			case rulesv1.RuleEndpointTypeServiceBus:
				targetResource["path"] = "/target-path"
			}

			rule := &rulesv1.Rule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rule",
					Namespace: "test-ns",
				},
				Spec: rulesv1.RuleSpec{
					Source:         "source-endpoint",
					Target:         "target-endpoint",
					SourceResource: sourceResource,
					TargetResource: targetResource,
				},
			}

			err := validateRule(rule)

			if tc.expectedResult {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "is not validate")
			}
		})
	}
}

func TestServeRule(t *testing.T) {
	t.Run("valid request with delete operation", func(t *testing.T) {
		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UID:       "test-uid",
				Operation: admissionv1.Delete,
			},
		}

		reviewBytes, err := json.Marshal(review)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/rules", bytes.NewBuffer(reviewBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		serveRule(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respReview admissionv1.AdmissionReview
		err = json.NewDecoder(resp.Body).Decode(&respReview)
		assert.NoError(t, err)
		assert.Equal(t, review.Request.UID, respReview.Response.UID)
		assert.True(t, respReview.Response.Allowed)
	})

	t.Run("valid request with create operation", func(t *testing.T) {
		sourceEndpoint := &rulesv1.RuleEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "source-endpoint",
				Namespace: "test-ns",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeRest,
			},
		}

		targetEndpoint := &rulesv1.RuleEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "target-endpoint",
				Namespace: "test-ns",
			},
			Spec: rulesv1.RuleEndpointSpec{
				RuleEndpointType: rulesv1.RuleEndpointTypeEventBus,
			},
		}

		mockCtrl := &SimpleRuleEndpointGetter{
			GetRuleEndpointFunc: func(namespace, name string) (*rulesv1.RuleEndpoint, error) {
				if name == "source-endpoint" {
					return sourceEndpoint, nil
				}
				return targetEndpoint, nil
			},
			ListRuleFunc: func(namespace string) ([]rulesv1.Rule, error) {
				return []rulesv1.Rule{}, nil
			},
		}

		restore := SetControllerForTesting(mockCtrl)
		defer restore()

		rule := rulesv1.Rule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rule",
				Namespace: "test-ns",
			},
			Spec: rulesv1.RuleSpec{
				Source:         "source-endpoint",
				Target:         "target-endpoint",
				SourceResource: map[string]string{"path": "/test-path"},
				TargetResource: map[string]string{"topic": "test-topic"},
			},
		}

		rawRule, err := json.Marshal(rule)
		assert.NoError(t, err)

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UID:       "test-uid",
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: rawRule,
				},
			},
		}

		reviewBytes, err := json.Marshal(review)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/rules", bytes.NewBuffer(reviewBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		serveRule(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respReview admissionv1.AdmissionReview
		err = json.NewDecoder(resp.Body).Decode(&respReview)
		assert.NoError(t, err)
		assert.Equal(t, review.Request.UID, respReview.Response.UID)
		assert.True(t, respReview.Response.Allowed)
	})
}
