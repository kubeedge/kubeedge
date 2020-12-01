/*
Copyright 2018 The Kubernetes Authors.
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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
)

// PrintFlags composes common printer flag structs
type PrintFlags struct {
	JSONYamlPrintFlags *genericclioptions.JSONYamlPrintFlags
	NamePrintFlags     *genericclioptions.NamePrintFlags
	HumanReadableFlags *HumanPrintFlags
	TemplateFlags      *genericclioptions.KubeTemplatePrintFlags

	NoHeaders    *bool
	OutputFormat *string
}

// AllowedFormats is the list of formats in which data can be displayed
func (f *PrintFlags) AllowedFormats() []string {
	formats := f.JSONYamlPrintFlags.AllowedFormats()
	formats = append(formats, f.HumanReadableFlags.AllowedFormats()...)
	return formats
}

// ToPrinter attempts to find a composed set of PrintFlags suitable for
// returning a printer based on current flag values.
func (f *PrintFlags) ToPrinter() (printers.ResourcePrinter, error) {
	outputFormat := ""
	if f.OutputFormat != nil {
		outputFormat = *f.OutputFormat
	}
	noHeaders := false
	if f.NoHeaders != nil {
		noHeaders = *f.NoHeaders
	}
	f.HumanReadableFlags.NoHeaders = noHeaders

	if p, err := f.JSONYamlPrintFlags.ToPrinter(outputFormat); !genericclioptions.IsNoCompatiblePrinterError(err) {
		return p, err
	}
	if p, err := f.HumanReadableFlags.ToPrinter(outputFormat); !genericclioptions.IsNoCompatiblePrinterError(err) {
		return p, err
	}

	return nil, genericclioptions.NoCompatiblePrinterError{OutputFormat: &outputFormat, AllowedFormats: f.AllowedFormats()}
}

// NewGetPrintFlags returns flags associated with JSONYamlPrintFlags
func NewGetPrintFlags() *PrintFlags {
	outputFormat := ""
	noHeaders := false

	return &PrintFlags{
		OutputFormat: &outputFormat,
		NoHeaders:    &noHeaders,

		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
		NamePrintFlags:     genericclioptions.NewNamePrintFlags(""),
		TemplateFlags:      genericclioptions.NewKubeTemplatePrintFlags(),

		HumanReadableFlags: NewHumanPrintFlags(),
	}
}

// HumanPrintFlags provides default flags necessary for printing.
// Given the following flag values, a printer can be requested that knows
// how to handle printing based on these values.
type HumanPrintFlags struct {
	ShowKind     *bool
	ShowLabels   *bool
	SortBy       *string
	ColumnLabels *[]string

	// get.go-specific values
	NoHeaders bool

	Kind          schema.GroupKind
	WithNamespace bool
}

// AllowedFormats returns more customized formating options
func (f *HumanPrintFlags) AllowedFormats() []string {
	return []string{"wide"}
}

// ToPrinter receives an outputFormat and returns a printer capable of
// handling human-readable output.
func (f *HumanPrintFlags) ToPrinter(outputFormat string) (printers.ResourcePrinter, error) {
	if len(outputFormat) > 0 && outputFormat != "wide" {
		return nil, genericclioptions.NoCompatiblePrinterError{Options: f, AllowedFormats: f.AllowedFormats()}
	}

	showKind := false
	if f.ShowKind != nil {
		showKind = *f.ShowKind
	}

	showLabels := false
	if f.ShowLabels != nil {
		showLabels = *f.ShowLabels
	}

	columnLabels := []string{}
	if f.ColumnLabels != nil {
		columnLabels = *f.ColumnLabels
	}

	p := printers.NewTablePrinter(printers.PrintOptions{
		Kind:          f.Kind,
		WithKind:      showKind,
		NoHeaders:     f.NoHeaders,
		Wide:          outputFormat == "wide",
		WithNamespace: f.WithNamespace,
		ColumnLabels:  columnLabels,
		ShowLabels:    showLabels,
	})

	return p, nil
}

// NewHumanPrintFlags returns flags associated with
// human-readable printing, with default values set.
func NewHumanPrintFlags() *HumanPrintFlags {
	showLabels := false
	sortBy := ""
	showKind := false
	columnLabels := []string{}

	return &HumanPrintFlags{
		NoHeaders:     false,
		WithNamespace: false,
		ColumnLabels:  &columnLabels,

		Kind:       schema.GroupKind{},
		ShowLabels: &showLabels,
		SortBy:     &sortBy,
		ShowKind:   &showKind,
	}
}

// EnsureWithNamespace ensures that humanreadable flags return
// a printer capable of printing with a "namespace" column.
func (f *PrintFlags) EnsureWithNamespace() error {
	return f.HumanReadableFlags.EnsureWithNamespace()
}

// EnsureWithNamespace sets the "WithNamespace" humanreadable option to true.
func (f *HumanPrintFlags) EnsureWithNamespace() error {
	f.WithNamespace = true
	return nil
}
