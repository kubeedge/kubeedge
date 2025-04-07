//go:build !windows

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

package execs

import (
	"os/exec"
	"strings"
)

func NewCommand(command string) *Command {
	return &Command{
		Cmd: exec.Command("bash", "-c", command),
	}
}

func (cmd Command) GetStdOut() string {
	if len(cmd.StdOut) != 0 {
		return strings.TrimSuffix(string(cmd.StdOut), "\n")
	}
	return ""
}

func (cmd Command) GetStdErr() string {
	if len(cmd.StdErr) != 0 {
		return strings.TrimSuffix(string(cmd.StdErr), "\n")
	}
	return ""
}
