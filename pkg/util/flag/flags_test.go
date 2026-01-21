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

package flag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigValue_IsBoolFlag(t *testing.T) {
	var cv ConfigValue
	assert.True(t, cv.IsBoolFlag())
}

func TestConfigValue_Get(t *testing.T) {
	tests := []struct {
		name     string
		value    ConfigValue
		expected interface{}
	}{
		{
			name:     "ConfigFalse",
			value:    ConfigFalse,
			expected: ConfigFalse,
		},
		{
			name:     "ConfigTrue",
			value:    ConfigTrue,
			expected: ConfigTrue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.Get())
		})
	}
}

func TestConfigValue_Set(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue ConfigValue
		wantErr   bool
	}{
		{
			name:      "Set true",
			input:     "true",
			wantValue: ConfigTrue,
			wantErr:   false,
		},
		{
			name:      "Set false",
			input:     "false",
			wantValue: ConfigFalse,
			wantErr:   false,
		},
		{
			name:      "Set 1 (true)",
			input:     "1",
			wantValue: ConfigTrue,
			wantErr:   false,
		},
		{
			name:      "Set 0 (false)",
			input:     "0",
			wantValue: ConfigFalse,
			wantErr:   false,
		},
		{
			name:      "Set invalid",
			input:     "invalid",
			wantValue: ConfigFalse, // Default value if parsing fails, but error should be returned
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cv ConfigValue
			err := cv.Set(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantValue, cv)
		})
	}
}

func TestConfigValue_String(t *testing.T) {
	tests := []struct {
		name     string
		value    ConfigValue
		expected string
	}{
		{
			name:     "ConfigFalse",
			value:    ConfigFalse,
			expected: "false",
		},
		{
			name:     "ConfigTrue",
			value:    ConfigTrue,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.value.String())
		})
	}
}

func TestConfigValue_Type(t *testing.T) {
	var cv ConfigValue
	assert.Equal(t, "config", cv.Type())
}

func TestConfig(t *testing.T) {
	name := "test-config"
	usage := "test usage"
	val := ConfigTrue

	// Since Config calls pflag.Var which registers flags to the global CommandLine,
	// we need to be careful. However, for unit testing the return value structure,
	// we can check if it returns a pointer to the correct value.

	// Note: We cannot easily test pflag registration side effects without resetting pflag.CommandLine,
	// but we can verify the returned object.

	p := Config(name, val, usage)
	assert.NotNil(t, p)
	assert.Equal(t, val, *p)
}

func TestPrintMinConfigAndExitIfRequested(t *testing.T) {
	// This function calls os.Exit(1) or fmt.Println, which is hard to test directly.
	// However, we can test the case where minConfigFlag is False (default),
	// ensuring it does NOT panic or exit.

	// Save original flag value
	original := *minConfigFlag
	defer func() { *minConfigFlag = original }()

	// Test case 1: minConfigFlag is False (should just return)
	*minConfigFlag = ConfigFalse
	config := map[string]string{"key": "value"}

	assert.NotPanics(t, func() {
		PrintMinConfigAndExitIfRequested(config)
	})
}

func TestPrintDefaultConfigAndExitIfRequested(t *testing.T) {
	// This function calls os.Exit(1) or fmt.Println, which is hard to test directly.
	// However, we can test the case where defaultConfigFlag is False (default),
	// ensuring it does NOT panic or exit.

	// Save original flag value
	original := *defaultConfigFlag
	defer func() { *defaultConfigFlag = original }()

	// Test case 1: defaultConfigFlag is False (should just return)
	*defaultConfigFlag = ConfigFalse
	config := map[string]string{"key": "value"}

	assert.NotPanics(t, func() {
		PrintDefaultConfigAndExitIfRequested(config)
	})
}
