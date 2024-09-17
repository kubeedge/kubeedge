/*
Copyright 2024 The KubeEdge Authors.

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

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewSelector(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		labelSelector string
		fieldSelector string
		expectedLabel string
		expectedField string
	}{
		{
			name:          "Empty selectors",
			labelSelector: "",
			fieldSelector: "",
			expectedLabel: labels.Everything().String(),
			expectedField: fields.Everything().String(),
		},
		{
			name:          "Empty field selector",
			labelSelector: "app=myapp",
			fieldSelector: "",
			expectedLabel: "app=myapp",
			expectedField: fields.Everything().String(),
		},
		{
			name:          "Empty label selector",
			labelSelector: "",
			fieldSelector: "metadata.name=pod1",
			expectedLabel: labels.Everything().String(),
			expectedField: "metadata.name=pod1",
		},
		{
			name:          "Valid label and field selectors",
			labelSelector: "app=myapp,tier=frontend",
			fieldSelector: "metadata.namespace=default",
			expectedLabel: "app=myapp,tier=frontend",
			expectedField: "metadata.namespace=default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector := NewSelector(tc.labelSelector, tc.fieldSelector)

			assert.Equal(tc.expectedLabel, selector.Label.String())
			assert.Equal(tc.expectedField, selector.Field.String())
		})
	}
}

func TestLabelFieldSelector_Labels(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		labelSelector string
		expected      string
	}{
		{
			name:          "Empty label selector",
			labelSelector: "",
			expected:      labels.Everything().String(),
		},
		{
			name:          "Valid label selector",
			labelSelector: "app=myapp",
			expected:      "app=myapp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector := NewSelector(tc.labelSelector, "")
			result := selector.Labels()

			assert.Equal(tc.expected, result.String())
		})
	}
}

func TestLabelFieldSelector_Fields(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		fieldSelector string
		expected      string
	}{
		{
			name:          "Empty field selector",
			fieldSelector: "",
			expected:      fields.Everything().String(),
		},
		{
			name:          "Valid field selector",
			fieldSelector: "metadata.name=pod1",
			expected:      "metadata.name=pod1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector := NewSelector("", tc.fieldSelector)
			result := selector.Fields()

			assert.Equal(tc.expected, result.String())
		})
	}
}

func TestLabelFieldSelector_String(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		labelSelector  string
		fieldSelector  string
		expectedString string
	}{
		{
			name:           "Empty selectors",
			labelSelector:  "",
			fieldSelector:  "",
			expectedString: ";",
		},
		{
			name:           "Label selector only",
			labelSelector:  "app=myapp",
			fieldSelector:  "",
			expectedString: "app=myapp;",
		},
		{
			name:           "Field selector only",
			labelSelector:  "",
			fieldSelector:  "metadata.name=pod1",
			expectedString: ";metadata.name=pod1",
		},
		{
			name:           "Both selectors",
			labelSelector:  "app=myapp",
			fieldSelector:  "metadata.name=pod1",
			expectedString: "app=myapp;metadata.name=pod1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector := NewSelector(tc.labelSelector, tc.fieldSelector)
			result := selector.String()

			assert.Equal(tc.expectedString, result)
		})
	}
}

func TestLabelFieldSelector_Match(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		labelSelector string
		fieldSelector string
		labelSet      labels.Set
		fieldSet      fields.Set
		expected      bool
	}{
		{
			name:          "Match both label and field",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			labelSet:      labels.Set{"app": "myapp"},
			fieldSet:      fields.Set{"metadata.name": "pod1"},
			expected:      true,
		},
		{
			name:          "Match label but not field",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			labelSet:      labels.Set{"app": "myapp"},
			fieldSet:      fields.Set{"metadata.name": "pod2"},
			expected:      false,
		},
		{
			name:          "Match field but not label",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			labelSet:      labels.Set{"app": "otherapp"},
			fieldSet:      fields.Set{"metadata.name": "pod1"},
			expected:      false,
		},
		{
			name:          "Empty selector matches everything",
			labelSelector: "",
			fieldSelector: "",
			labelSet:      labels.Set{"app": "myapp"},
			fieldSet:      fields.Set{"metadata.name": "pod1"},
			expected:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			selector := NewSelector(tc.labelSelector, tc.fieldSelector)
			result := selector.Match(tc.labelSet, tc.fieldSet)

			assert.Equal(tc.expected, result)
		})
	}
}

func TestLabelFieldSelector_MatchObj(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		labelSelector string
		fieldSelector string
		obj           *unstructured.Unstructured
		expected      bool
	}{
		{
			name:          "Match both label and field",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "pod1",
						"labels": map[string]interface{}{
							"app": "myapp",
						},
					},
				},
			},
			expected: true,
		},
		{
			name:          "Match label but not field",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "pod2",
						"labels": map[string]interface{}{
							"app": "myapp",
						},
					},
				},
			},
			expected: false,
		},
		{
			name:          "Match field but not label",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "pod1",
						"labels": map[string]interface{}{
							"app": "otherapp",
						},
					},
				},
			},
			expected: false,
		},
		{
			name:          "Match neither label nor field",
			labelSelector: "app=myapp",
			fieldSelector: "metadata.name=pod1",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "pod2",
						"labels": map[string]interface{}{
							"app": "otherapp",
						},
					},
				},
			},
			expected: false,
		},
		{
			name:          "Empty selector matches everything",
			labelSelector: "",
			fieldSelector: "",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "pod1",
						"labels": map[string]interface{}{
							"app": "myapp",
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.obj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "Pod"})

			selector := NewSelector(tc.labelSelector, tc.fieldSelector)
			result := selector.MatchObj(tc.obj)

			assert.Equal(tc.expected, result)
		})
	}
}
