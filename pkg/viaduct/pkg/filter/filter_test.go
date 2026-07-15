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

package filter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// newMsg returns a minimal *model.Message for use in tests.
func newMsg(id string) *model.Message {
	return &model.Message{
		Header: model.MessageHeader{ID: id},
	}
}

// ---------------------------------------------------------------------------
// AddFilterFunc
// ---------------------------------------------------------------------------

// TestAddFilterFunc_AppendsSingleFilter verifies that AddFilterFunc registers
// a filter function so that ProcessFilter invokes it.
func TestAddFilterFunc_AppendsSingleFilter(t *testing.T) {
	called := false
	mf := &MessageFilter{}

	mf.AddFilterFunc(func(msg *model.Message) error {
		called = true
		return nil
	})

	err := mf.ProcessFilter(newMsg("m1"))

	assert.NoError(t, err)
	assert.True(t, called, "expected the registered filter to be called")
}

// TestAddFilterFunc_AppendsMultipleFilters verifies that multiple calls to
// AddFilterFunc register all filters and that ProcessFilter invokes them in
// registration order.
func TestAddFilterFunc_AppendsMultipleFilters(t *testing.T) {
	var order []int
	mf := &MessageFilter{}

	mf.AddFilterFunc(func(msg *model.Message) error {
		order = append(order, 1)
		return nil
	})
	mf.AddFilterFunc(func(msg *model.Message) error {
		order = append(order, 2)
		return nil
	})
	mf.AddFilterFunc(func(msg *model.Message) error {
		order = append(order, 3)
		return nil
	})

	err := mf.ProcessFilter(newMsg("m2"))

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, order, "filters must be invoked in registration order")
}

// TestAddFilterFunc_PreservesExistingFilters verifies that AddFilterFunc
// appends new filters without replacing previously registered filters.
func TestAddFilterFunc_PreservesExistingFilters(t *testing.T) {
	var order []int
	mf := &MessageFilter{}

	mf.AddFilterFunc(func(msg *model.Message) error {
		order = append(order, 1)
		return nil
	})
	mf.AddFilterFunc(func(msg *model.Message) error {
		order = append(order, 2)
		return nil
	})

	assert.Len(t, mf.Filters, 2)
	assert.NoError(t, mf.Filters[0](newMsg("m3")))
	assert.NoError(t, mf.Filters[1](newMsg("m4")))
	assert.Equal(t, []int{1, 2}, order, "registered filters should remain addressable in append order")
}

// ---------------------------------------------------------------------------
// ProcessFilter
// ---------------------------------------------------------------------------

// TestProcessFilter_EmptyFilterList verifies that ProcessFilter returns nil
// without error when no filters have been registered.
func TestProcessFilter_EmptyFilterList(t *testing.T) {
	mf := &MessageFilter{}

	err := mf.ProcessFilter(newMsg("m3"))

	assert.NoError(t, err)
}

// TestProcessFilter_AllFiltersPass verifies that ProcessFilter returns nil
// when every registered filter returns nil.
func TestProcessFilter_AllFiltersPass(t *testing.T) {
	mf := &MessageFilter{}
	mf.AddFilterFunc(func(msg *model.Message) error { return nil })
	mf.AddFilterFunc(func(msg *model.Message) error { return nil })

	err := mf.ProcessFilter(newMsg("m4"))

	assert.NoError(t, err)
}

// TestProcessFilter_ShortCircuitsOnError verifies that ProcessFilter stops
// invoking subsequent filters as soon as one returns an error, and that the
// exact same error instance is propagated to the caller (not a wrapped copy).
func TestProcessFilter_ShortCircuitsOnError(t *testing.T) {
	filterErr := errors.New("message rejected by filter")
	thirdCalled := false

	mf := &MessageFilter{}
	mf.AddFilterFunc(func(msg *model.Message) error { return nil })
	mf.AddFilterFunc(func(msg *model.Message) error { return filterErr })
	mf.AddFilterFunc(func(msg *model.Message) error {
		thirdCalled = true
		return nil
	})

	err := mf.ProcessFilter(newMsg("m5"))

	assert.Same(t, filterErr, err, "ProcessFilter must return the exact error instance, not a wrapped copy")
	assert.False(t, thirdCalled, "filters after the failing one must not be called")
}

// TestProcessFilter_ReturnsFirstError verifies that when the first filter
// returns an error the remaining filters are skipped and the exact same error
// instance is returned (not a wrapped copy).
func TestProcessFilter_ReturnsFirstError(t *testing.T) {
	firstErr := errors.New("first filter error")
	secondCalled := false

	mf := &MessageFilter{}
	mf.AddFilterFunc(func(msg *model.Message) error { return firstErr })
	mf.AddFilterFunc(func(msg *model.Message) error {
		secondCalled = true
		return nil
	})

	err := mf.ProcessFilter(newMsg("m6"))

	assert.Same(t, firstErr, err, "ProcessFilter must return the exact error instance, not a wrapped copy")
	assert.False(t, secondCalled, "second filter must not be called after first returns error")
}

// TestProcessFilter_PassesSameMessageInstanceToAllFilters verifies that the
// same *model.Message pointer (not a copy) is passed to every filter in the
// chain.  It also confirms that a mutation applied by an earlier filter is
// visible to later filters, proving the shared-message chain behaviour.
func TestProcessFilter_PassesSameMessageInstanceToAllFilters(t *testing.T) {
	msg := newMsg("m7")
	var seen []*model.Message

	mf := &MessageFilter{}
	// First filter: record the pointer and mutate a field.
	mf.AddFilterFunc(func(m *model.Message) error {
		seen = append(seen, m)
		m.Header.ID = "mutated-by-filter-1"
		return nil
	})
	// Second filter: record the pointer and verify the mutation is visible.
	mf.AddFilterFunc(func(m *model.Message) error {
		seen = append(seen, m)
		assert.Equal(t, "mutated-by-filter-1", m.Header.ID,
			"second filter must observe the mutation made by the first filter")
		return nil
	})

	err := mf.ProcessFilter(msg)

	assert.NoError(t, err)
	assert.Len(t, seen, 2)
	assert.Same(t, msg, seen[0], "filter 1 must receive the original message pointer")
	assert.Same(t, msg, seen[1], "filter 2 must receive the original message pointer")
}
