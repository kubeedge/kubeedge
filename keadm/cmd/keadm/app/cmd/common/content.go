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
	"fmt"
	"os"
)

// TODO (@zc2638) Need to migrate util's constants to common

var serviceFileTemplate = `[Unit]
Description=%s.service

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`

func GenerateServiceFile(process string, execStartCmd string) error {
	filename := fmt.Sprintf("%s.service", process)
	content := fmt.Sprintf(serviceFileTemplate, process, execStartCmd)
	serviceFilePath := fmt.Sprintf("/etc/systemd/system/%s", filename)
	return os.WriteFile(serviceFilePath, []byte(content), os.ModePerm)
}

func RemoveServiceFile(process string) error {
	filename := fmt.Sprintf("%s.service", process)
	return os.Remove(fmt.Sprintf("/etc/systemd/system/%s", filename))
}
