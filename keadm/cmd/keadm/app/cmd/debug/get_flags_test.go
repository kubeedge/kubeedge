/*
Copyright 2024 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
@CHANGELOG
KubeEdge Authors: To create keadm debug get function like kubectl get,
This file is derived from K8S Kubectl code with reduced set of methods
Changes done are
1. Package edged got some functions from "k8s.io/kubectl/pkg/cmd/get/get_flags.go"
and made some variant
*/

package debug

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func AllowedFormats(t *testing.T) {
	assert := assert.New(t)
	printFlags := &PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		NamePrintFlags:     genericclioptions.NewNamePrintFlags(""),
		TemplateFlags:      genericclioptions.NewKubeTemplatePrintFlags(),
		HumanReadableFlags: NewHumanPrintFlags(),
	}

	formats := printFlags.AllowedFormats()
	expectedFormats := append(printFlags.JSONYamlPrintFlags.AllowedFormats(), printFlags.HumanReadableFlags.AllowedFormats()...)

	assert.Equal(expectedFormats, formats)
}

func TestToPrinter(t *testing.T) {
	assert := assert.New(t)

	printFlags := &PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		NamePrintFlags:     genericclioptions.NewNamePrintFlags(""),
		TemplateFlags:      genericclioptions.NewKubeTemplatePrintFlags(),
		HumanReadableFlags: NewHumanPrintFlags(),
		NoHeaders:          new(bool),
		OutputFormat:       new(string),
	}

	*printFlags.OutputFormat = "json"
	printer, err := printFlags.ToPrinter()
	assert.NoError(err)
	assert.NotNil(printer)

	*printFlags.OutputFormat = "yaml"
	printer, err = printFlags.ToPrinter()
	assert.NoError(err)
	assert.NotNil(printer)

	*printFlags.OutputFormat = FormatTypeWIDE
	printer, err = printFlags.ToPrinter()
	assert.NoError(err)
	assert.NotNil(printer)

	*printFlags.OutputFormat = "unsupported"
	printer, err = printFlags.ToPrinter()
	assert.Error(err)
	assert.Nil(printer)

	*printFlags.OutputFormat = FormatTypeWIDE
	*printFlags.NoHeaders = true
	printer, err = printFlags.ToPrinter()
	assert.NoError(err)
	assert.NotNil(printer)

	*printFlags.NoHeaders = false

	*printFlags.OutputFormat = ""
	printer, err = printFlags.ToPrinter()
	assert.NoError(err)
	assert.NotNil(printer)
}

func TestNewGetPrintFlags(t *testing.T) {
	assert := assert.New(t)

	printFlags := NewGetPrintFlags()
	assert.NotNil(printFlags)

	assert.IsType(&genericclioptions.JSONYamlPrintFlags{}, printFlags.JSONYamlPrintFlags)
	assert.IsType(&genericclioptions.NamePrintFlags{}, printFlags.NamePrintFlags)
	assert.IsType(&genericclioptions.KubeTemplatePrintFlags{}, printFlags.TemplateFlags)
	assert.IsType(&HumanPrintFlags{}, printFlags.HumanReadableFlags)

	assert.NotNil(printFlags.OutputFormat)
	assert.Equal("", *printFlags.OutputFormat)

	assert.NotNil(printFlags.NoHeaders)
	assert.Equal(false, *printFlags.NoHeaders)

	assert.Equal(printFlags.HumanReadableFlags.NoHeaders, false)
	assert.Equal(printFlags.HumanReadableFlags.WithNamespace, false)
	assert.Equal(*printFlags.HumanReadableFlags.ShowLabels, false)
	assert.Equal(*printFlags.HumanReadableFlags.SortBy, "")
	assert.Equal(*printFlags.HumanReadableFlags.ShowKind, false)
	assert.Empty(*printFlags.HumanReadableFlags.ColumnLabels)
}

func TestHumanPrintFlags_AllowedFormats(t *testing.T) {
	assert := assert.New(t)

	humanPrintFlags := &HumanPrintFlags{}
	formats := humanPrintFlags.AllowedFormats()

	expectedFormats := []string{FormatTypeWIDE}
	assert.Equal(expectedFormats, formats)
}

func TestNewHumanPrintFlags(t *testing.T) {
	assert := assert.New(t)

	humanPrintFlags := NewHumanPrintFlags()
	assert.NotNil(humanPrintFlags)

	assert.Equal(humanPrintFlags.NoHeaders, false)
	assert.Equal(humanPrintFlags.WithNamespace, false)
	assert.Equal(humanPrintFlags.ColumnLabels, &[]string{})
	assert.Equal(humanPrintFlags.Kind, schema.GroupKind{})
	assert.Equal(*humanPrintFlags.ShowLabels, false)
	assert.Equal(*humanPrintFlags.SortBy, "")
	assert.Equal(*humanPrintFlags.ShowKind, false)
}

func TestHumanPrintFlags_ToPrinter(t *testing.T) {
	assert := assert.New(t)

	humanPrintFlags := &HumanPrintFlags{
		ShowKind:      new(bool),
		ShowLabels:    new(bool),
		SortBy:        new(string),
		ColumnLabels:  new([]string),
		NoHeaders:     false,
		Kind:          schema.GroupKind{},
		WithNamespace: false,
	}

	outputFormat := FormatTypeWIDE
	printer, err := humanPrintFlags.ToPrinter(outputFormat)
	assert.NoError(err)
	assert.NotNil(printer)

	outputFormat = ""
	printer, err = humanPrintFlags.ToPrinter(outputFormat)
	assert.NoError(err)
	assert.NotNil(printer)

	outputFormat = "unsupported"
	printer, err = humanPrintFlags.ToPrinter(outputFormat)
	assert.Error(err)
	assert.Nil(printer)
}

func TestHumanPrintFlags_EnsureWithNamespace(t *testing.T) {
	assert := assert.New(t)
	humanPrintFlags := &HumanPrintFlags{}

	err := humanPrintFlags.EnsureWithNamespace()

	assert.NoError(err)
	assert.Equal(humanPrintFlags.WithNamespace, true)
}

func Test_EnsureWithNamespace(t *testing.T) {
	assert := assert.New(t)

	printFlags := NewGetPrintFlags()
	err := printFlags.EnsureWithNamespace()

	assert.NoError(err)
}
