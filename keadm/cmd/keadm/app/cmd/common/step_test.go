/*
Copyright 2025 The KubeEdge Authors.

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
	"flag"
	"testing"

	"k8s.io/klog/v2"
)

func TestNewStep(t *testing.T) {
	tests := []struct {
		name string
		want *Step
	}{
		{
			name: "Create new Step instance",
			want: &Step{n: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStep()
			if got.n != tt.want.n {
				t.Errorf("NewStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStep_Printf(t *testing.T) {
	// Initialize klog flags properly
	fs := flag.NewFlagSet("test", flag.PanicOnError)
	klog.InitFlags(fs)

	tests := []struct {
		name   string
		step   *Step
		format string
		args   []interface{}
		wantN  int
	}{
		{
			name:   "Print first step",
			step:   &Step{n: 0},
			format: "Test message %s",
			args:   []interface{}{"one"},
			wantN:  1,
		},
		{
			name:   "Print second step",
			step:   &Step{n: 1},
			format: "Test message %d %s",
			args:   []interface{}{2, "two"},
			wantN:  2,
		},
		{
			name:   "Print with no arguments",
			step:   &Step{n: 2},
			format: "Simple message",
			args:   []interface{}{},
			wantN:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.step.Printf(tt.format, tt.args...)
			if tt.step.n != tt.wantN {
				t.Errorf("Step.Printf() counter = %v, want %v", tt.step.n, tt.wantN)
			}
		})
	}
}
