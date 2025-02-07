/*
Copyright 2022 The KubeEdge Authors.

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

package common

import (
	"testing"
)

func TestNewStep(t *testing.T) {
	step := NewStep()
	if step == nil {
		t.Error("NewStep() returned nil")
	}
	if step.n != 0 {
		t.Errorf("NewStep() initial counter should be 0, got %d", step.n)
	}
}

func TestStep_Printf(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		wantStep int
	}{
		{
			name:     "single print",
			format:   "test message",
			args:     []interface{}{},
			wantStep: 1,
		},
		{
			name:     "print with format",
			format:   "test message %s",
			args:     []interface{}{"arg1"},
			wantStep: 1,
		},
		{
			name:     "print with multiple args",
			format:   "test %s %d",
			args:     []interface{}{"message", 42},
			wantStep: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStep()
			s.Printf(tt.format, tt.args...)
			if s.n != tt.wantStep {
				t.Errorf("Step.Printf() counter = %d, want %d", s.n, tt.wantStep)
			}
		})
	}
}

func TestStep_PrintfMultiple(t *testing.T) {
	s := NewStep()

	// Test multiple Printf calls increment counter correctly
	expectedSteps := 3
	for i := 0; i < expectedSteps; i++ {
		s.Printf("test message %d", i)
	}

	if s.n != expectedSteps {
		t.Errorf("Step.Printf() multiple calls counter = %d, want %d", s.n, expectedSteps)
	}
}
