//go:build !windows

/*
Copyright 2024 The KubeEdge Authors.

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

package extsystem

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"
)

type OpenRCExtSystem struct {
	initsystem.OpenRCInitSystem
}

func (openrc OpenRCExtSystem) ServiceEnable(service string) error {
	return exec.Command(openrc.EnableCommand(service)).Run()
}

func (OpenRCExtSystem) ServiceDisable(service string) error {
	args := []string{"delete", service, "default"}
	return exec.Command("rc-update", args...).Run()
}

func (OpenRCExtSystem) ServiceCreate(_, _ string, _ map[string]string) error {
	// TODO: Implement this method when we need.
	return nil
}

func (OpenRCExtSystem) ServiceRemove(_ string) error {
	// TODO: Implement this method when we need.
	return nil
}

type SystemdExtSystem struct {
	initsystem.SystemdInitSystem
}

const (
	systemdDir = "/etc/systemd/system"
)

func (sysd SystemdExtSystem) ServiceEnable(service string) error {
	return exec.Command(sysd.EnableCommand(service)).Run()
}

func (SystemdExtSystem) ServiceDisable(service string) error {
	args := []string{"disable", service}
	return exec.Command("systemctl", args...).Run()
}

//go:embed simple.service
var simpleSystemdServiceTemplate string

func (SystemdExtSystem) ServiceCreate(service, cmd string, envs map[string]string) error {
	var envsStr string
	for k, v := range envs {
		if envsStr != "" {
			envsStr += " "
		}
		envsStr += fmt.Sprintf("\"%s=%s\"", k, v)
	}
	content := fmt.Sprintf(simpleSystemdServiceTemplate, service, cmd, envsStr)
	fp := fmt.Sprintf("%s/%s.service", systemdDir, service)
	return os.WriteFile(fp, []byte(content), os.ModePerm)
}

func (SystemdExtSystem) ServiceRemove(service string) error {
	return os.Remove(fmt.Sprintf("%s/%s.service", systemdDir, service))
}

// GetExtSystem returns an ExtSystem for the current system, or nil
// if we cannot detect a supported init system.
// This indicates we will skip init system checks, not an error.
func GetExtSystem() (ExtSystem, error) {
	// Assume existence of systemctl in path implies this is a systemd system:
	_, err := exec.LookPath("systemctl")
	if err == nil {
		return &SystemdExtSystem{}, nil
	}
	_, err = exec.LookPath("openrc")
	if err == nil {
		return &OpenRCExtSystem{}, nil
	}
	return nil, errors.New("no supported init system detected, skipping checking for services")
}
