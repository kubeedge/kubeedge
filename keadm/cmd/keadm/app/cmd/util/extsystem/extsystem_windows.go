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

package extsystem

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows/svc"

	"golang.org/x/sys/windows/svc/mgr"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem"
)

type WindowsExtSystem struct {
	initsystem.WindowsInitSystem
}

func (WindowsExtSystem) ServiceCreate(service, cmdline string, _ map[string]string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	if s, err := m.OpenService(service); err == nil {
		_ = s.Close()
		return nil
	}
	cfg := mgr.Config{
		DisplayName: service,
		StartType:   mgr.StartAutomatic,
	}
	s, err := m.CreateService(service, cmdline, cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	return nil
}

func (WindowsExtSystem) ServiceRemove(service string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(service)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Delete()
}

func (WindowsExtSystem) ServiceEnable(service string) error {
	cmd := exec.Command("PowerShell", "-NoProfile", "-Command",
		fmt.Sprintf("Set-Service '%s' -StartupType Automatic", service))
	return cmd.Run()
}

func (w WindowsExtSystem) ServiceStart(service string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(service)
	if err != nil {
		return errors.Wrapf(err, "could not access service %s", service)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return errors.Wrapf(err, "could not query service %s", service)
	}

	if status.State != svc.Stopped && status.State != svc.StopPending {
		return nil
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return errors.Errorf("timeout waiting for %s service to stop", service)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return errors.Wrapf(err, "could not retrieve %s service status", service)
		}
	}

	// Start the service
	err = s.Start()
	if err != nil {
		return errors.Wrapf(err, "could not start service %s", service)
	}

	// Check that the start was successful
	status, err = s.Query()
	if err != nil {
		return errors.Wrapf(err, "could not query service %s", service)
	}
	timeout = time.Now().Add(10 * time.Second)
	for status.State != svc.Running {
		if timeout.Before(time.Now()) {
			return errors.Errorf("timeout waiting for %s service to start", service)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return errors.Wrapf(err, "could not retrieve %s service status", service)
		}
	}
	return nil
}

func (w WindowsExtSystem) ServiceStop(service string) error {
	return w.WindowsInitSystem.ServiceStop(service)
}

func (w WindowsExtSystem) ServiceRestart(service string) error {
	return w.WindowsInitSystem.ServiceRestart(service)
}

func (w WindowsExtSystem) ServiceExists(service string) bool {
	return w.WindowsInitSystem.ServiceExists(service)
}

func (w WindowsExtSystem) ServiceIsEnabled(service string) bool {
	return w.WindowsInitSystem.ServiceIsEnabled(service)
}

func (w WindowsExtSystem) ServiceIsActive(service string) bool {
	return w.WindowsInitSystem.ServiceIsActive(service)
}
func (WindowsExtSystem) ServiceDisable(service string) error {
	cmd := exec.Command("PowerShell", "-NoProfile", "-Command",
		fmt.Sprintf("Set-Service '%s' -StartupType Disabled", service))
	return cmd.Run()
}
func GetExtSystem() (ExtSystem, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, errors.New("connect to Windows Service Manager failed")
	}
	_ = m.Disconnect()
	return &WindowsExtSystem{}, nil
}
