/*
Copyright 2020 The KubeEdge Authors.
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

package debug

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	edgecoreCfg "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func TestNewCmdDebugGet(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCmdDebugGet()

	assert.NotNil(cmd)
	assert.Equal("get", cmd.Use)
	assert.Equal("Display one or many resources", cmd.Short)
	assert.Equal(debugGetLong, cmd.Long)
	assert.Equal(debugGetExample, cmd.Example)

	assert.NotNil(cmd.Run)

	getOption := NewGetOptions()

	flag := cmd.Flag("namespace")
	assert.NotNil(flag)
	assert.Equal(getOption.Namespace, flag.DefValue)
	assert.Equal("namespace", flag.Name)
	assert.Equal("n", flag.Shorthand)

	flag = cmd.Flag("output")
	assert.NotNil(flag)
	assert.Equal(*getOption.PrintFlags.OutputFormat, flag.DefValue)
	assert.Equal("output", flag.Name)
	assert.Equal("o", flag.Shorthand)

	flag = cmd.Flag("selector")
	assert.NotNil(flag)
	assert.Equal(getOption.LabelSelector, flag.DefValue)
	assert.Equal("selector", flag.Name)
	assert.Equal("l", flag.Shorthand)

	flag = cmd.Flag("edgedb-path")
	assert.NotNil(flag)
	assert.Equal(getOption.DataPath, flag.DefValue)
	assert.Equal("edgedb-path", flag.Name)
	assert.Equal("p", flag.Shorthand)

	flag = cmd.Flag("all-namespaces")
	assert.NotNil(flag)
	assert.Equal(getOption.AllNamespace, flag.DefValue == "true")
	assert.Equal("all-namespaces", flag.Name)
	assert.Equal("A", flag.Shorthand)
}

func TestCheckErr(t *testing.T) {
	assert := assert.New(t)

	mockHandler := func(msg string, exitCode int) {
		t.Errorf("handleErr should not be called for nil error")
	}
	CheckErr(nil, mockHandler)

	expectedMsg := "Test error"
	expectedExitCode := DefaultErrorExitCode
	expectedErr := errors.New(expectedMsg)

	var handledMsg string
	var handledExitCode int

	mockHandler = func(msg string, exitCode int) {
		handledMsg = msg
		handledExitCode = exitCode
	}

	CheckErr(expectedErr, mockHandler)

	assert.Equal(expectedMsg, handledMsg)
	assert.Equal(expectedExitCode, handledExitCode)
}

func TestAddGetOtherFlags(t *testing.T) {
	getOption := NewGetOptions()
	cmd := &cobra.Command{}

	addGetOtherFlags(cmd, getOption)

	assert := assert.New(t)

	flag := cmd.Flag("namespace")
	assert.NotNil(flag)
	assert.Equal(getOption.Namespace, flag.DefValue)
	assert.Equal("namespace", flag.Name)
	assert.Equal("n", flag.Shorthand)

	flag = cmd.Flag("output")
	assert.NotNil(flag)
	assert.Equal(*getOption.PrintFlags.OutputFormat, flag.DefValue)
	assert.Equal("output", flag.Name)
	assert.Equal("o", flag.Shorthand)

	flag = cmd.Flag("selector")
	assert.NotNil(flag)
	assert.Equal(getOption.LabelSelector, flag.DefValue)
	assert.Equal("selector", flag.Name)
	assert.Equal("l", flag.Shorthand)

	flag = cmd.Flag("edgedb-path")
	assert.NotNil(flag)
	assert.Equal(getOption.DataPath, flag.DefValue)
	assert.Equal("edgedb-path", flag.Name)
	assert.Equal("p", flag.Shorthand)

	flag = cmd.Flag("all-namespaces")
	assert.NotNil(flag)
	assert.Equal(getOption.AllNamespace, flag.DefValue == "true")
	assert.Equal("all-namespaces", flag.Name)
	assert.Equal("A", flag.Shorthand)
}

func TestNewGetOptions(t *testing.T) {
	assert := assert.New(t)
	opts := NewGetOptions()

	assert.NotNil(opts)
	assert.Equal(opts.Namespace, "default")
	assert.Equal(opts.DataPath, edgecoreCfg.DataBaseDataSource)
	assert.Equal(opts.PrintFlags, NewGetPrintFlags())
}

func TestIsAllowedFormat(t *testing.T) {
	assert := assert.New(t)
	getOptions := NewGetOptions()

	tests := []struct {
		format   string
		expected bool
	}{
		{
			"yaml",
			true,
		},
		{
			"json",
			true,
		},
		{
			"wide",
			true,
		},
		{
			"xml",
			false,
		},
		{
			"",
			false,
		},
		{
			"plain",
			false,
		},
	}

	for _, test := range tests {
		stdResult := getOptions.IsAllowedFormat(test.format)
		assert.Equal(test.expected, stdResult)
	}
}

func TestSplitSelectorParameters(t *testing.T) {
	assert := assert.New(t)
	type args struct {
		args string
	}
	tests := []struct {
		name    string
		args    args
		want    []Selector
		wantErr bool
	}{
		{
			name: "testWithAllLabels",
			args: args{args: "key1==value1,key2!=value2,key3=value3"},
			want: []Selector{
				{Key: "key1", Value: "value1", Exist: true},
				{Key: "key2", Value: "value2", Exist: false},
				{Key: "key3", Value: "value3", Exist: true},
			},
			wantErr: false,
		},
		{
			name:    "testWithoutLabel",
			args:    args{args: "key1"},
			want:    []Selector{},
			wantErr: false,
		},
		{
			name:    "testWithEmptyValue",
			args:    args{args: "key1!="},
			want:    []Selector{{Key: "key1", Value: "", Exist: false}},
			wantErr: false,
		},
		{
			name:    "testWithMoreThanOneLabel",
			args:    args{args: "key1=value1=,key2=value2"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "testEmptyString",
			args:    args{args: ""},
			want:    []Selector{},
			wantErr: false,
		},
		{
			name:    "testOnlyCommas",
			args:    args{args: ",,,"},
			want:    []Selector{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitSelectorParameters(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitSelectorParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestIsExistName(t *testing.T) {
	tests := []struct {
		name     string
		resNames []string
		key      string
		expected bool
	}{
		{
			name:     "Key exists in resNames",
			resNames: []string{"pod1", "pod2", "pod3"},
			key:      "pod1",
			expected: true,
		},
		{
			name:     "Key does not exist in resNames",
			resNames: []string{"pod1", "pod2", "pod3"},
			key:      "pod4",
			expected: false,
		},
		{
			name:     "Empty resNames",
			resNames: []string{},
			key:      "pod1",
			expected: false,
		},
		{
			name:     "Empty key",
			resNames: []string{"pod1", "pod2", "pod3"},
			key:      "",
			expected: false,
		},
		{
			name:     "Key is substring of one element in resNames",
			resNames: []string{"pod123", "pod2", "pod3"},
			key:      "pod1",
			expected: false,
		},
		{
			name:     "ResNames contains special characters",
			resNames: []string{"pod@123", "pod#2", "pod$3"},
			key:      "pod@123",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isExistName(test.resNames, test.key)
			assert.Equal(t, test.expected, result)
		})
	}
}
