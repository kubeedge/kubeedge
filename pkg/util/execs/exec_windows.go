//go:build windows

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

	"golang.org/x/text/encoding/simplifiedchinese"
)

func NewCommand(command string) *Command {
	return &Command{
		Cmd: exec.Command("powershell", "-c", command),
	}
}

func (cmd Command) GetStdOut() string {
	if len(cmd.StdOut) != 0 {
		return strings.TrimSuffix(ConvertByte2String(cmd.StdOut, "GB18030"), "\n")
	}
	return ""
}

func (cmd Command) GetStdErr() string {
	if len(cmd.StdErr) != 0 {
		return strings.TrimSuffix(ConvertByte2String(cmd.StdErr, "GB18030"), "\n")
	}
	return ""
}

func ConvertByte2String(byte []byte, charset string) string {
	var str string
	switch charset {
	case "GB18030":
		var decodeBytes, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case "UTF8":
		fallthrough
	default:
		str = string(byte)
	}
	return str
}
