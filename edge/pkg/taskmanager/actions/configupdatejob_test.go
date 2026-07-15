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

package actions

import (
	"strings"
	"testing"
)

func TestBuildConfigUpdateArgsDoesNotUseShell(t *testing.T) {
	updateFields := map[string]string{
		"modules.edgehub.websocket.url":           "ws://127.0.0.1:10000/e632aba927ea4ac2b575ec1603d56f10/events",
		"modules.edgehub.websocket.writeDeadline": "30; touch /tmp/pwned",
	}

	args := buildConfigUpdateArgs(updateFields)

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "config-update" {
		t.Fatalf("expected config-update subcommand, got %q", args[0])
	}
	if args[1] != "--set" {
		t.Fatalf("expected --set flag, got %q", args[1])
	}
	if !strings.Contains(args[2], "30; touch /tmp/pwned") {
		t.Fatalf("expected update value to remain a single argv value, got %q", args[2])
	}

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "bash -c") || strings.Contains(joined, "sh -c") {
		t.Fatalf("args must not invoke a shell: %v", args)
	}
}

func TestBuildConfigUpdateArgsSortsFields(t *testing.T) {
	updateFields := map[string]string{
		"z.key": "z",
		"a.key": "a",
	}

	args := buildConfigUpdateArgs(updateFields)

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "config-update" {
		t.Fatalf("expected config-update subcommand, got %q", args[0])
	}
	if args[1] != "--set" {
		t.Fatalf("expected --set flag, got %q", args[1])
	}
	if args[2] != "a.key=a,z.key=z" {
		t.Fatalf("unexpected --set value: %q", args[2])
	}
}

func TestBuildConfigUpdateArgsEmptyFields(t *testing.T) {
	args := buildConfigUpdateArgs(map[string]string{})

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "config-update" || args[1] != "--set" || args[2] != "" {
		t.Fatalf("unexpected args for empty update fields: %v", args)
	}
}
