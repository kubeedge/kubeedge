//go:build windows

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

package execs

import (
	"testing"
)

func TestConvertByte2String(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		charset  string
		expected string
	}{
		{
			name:     "UTF8",
			input:    []byte("hello"),
			charset:  "UTF8",
			expected: "hello",
		},
		{
			name:     "default fallthrough",
			input:    []byte("hello"),
			charset:  "UNKNOWN",
			expected: "hello",
		},
		{
			name:     "GB18030 ASCII",
			input:    []byte("hello"),
			charset:  "GB18030",
			expected: "hello",
		},
		{
			name:     "GB18030 Chinese",
			input:    []byte{0xC4, 0xE3, 0xBA, 0xC3}, // "你好" in GB18030
			charset:  "GB18030",
			expected: "你好",
		},
		{
			name:     "empty input",
			input:    []byte{},
			charset:  "UTF8",
			expected: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ConvertByte2String(c.input, c.charset)
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand("echo hello")
	if cmd == nil {
		t.Fatal("expected non-nil Command")
	}
	if cmd.Cmd == nil {
		t.Fatal("expected non-nil exec.Cmd")
	}
	args := cmd.Cmd.Args
	if len(args) < 3 || args[0] != "powershell" || args[1] != "-c" || args[2] != "echo hello" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestGetStdOut(t *testing.T) {
	cases := []struct {
		name     string
		stdout   []byte
		expected string
	}{
		{
			name:     "empty stdout",
			stdout:   []byte{},
			expected: "",
		},
		{
			name:     "ascii stdout",
			stdout:   []byte("hello\n"),
			expected: "hello",
		},
		{
			name:     "GB18030 chinese stdout",
			stdout:   []byte{0xC4, 0xE3, 0xBA, 0xC3, '\n'},
			expected: "你好",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := Command{StdOut: c.stdout}
			got := cmd.GetStdOut()
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}

func TestGetStdErr(t *testing.T) {
	cases := []struct {
		name     string
		stderr   []byte
		expected string
	}{
		{
			name:     "empty stderr",
			stderr:   []byte{},
			expected: "",
		},
		{
			name:     "ascii stderr",
			stderr:   []byte("error\n"),
			expected: "error",
		},
		{
			name:     "GB18030 chinese stderr",
			stderr:   []byte{0xC4, 0xE3, 0xBA, 0xC3, '\n'},
			expected: "你好",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := Command{StdErr: c.stderr}
			got := cmd.GetStdErr()
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}
