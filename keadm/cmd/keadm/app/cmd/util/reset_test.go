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

package util

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"
)

func TestNewResetOptions(t *testing.T) {
	opts := NewResetOptions()
	if opts == nil {
		t.Fatal("Expected non-nil ResetOptions")
	}
}

type fakeExecer struct {
	utilsexec.Interface
}

func (f *fakeExecer) Command(cmd string, args ...string) utilsexec.Cmd {
	return nil
}

func (f *fakeExecer) LookPath(file string) (string, error) {
	return "", nil
}

type mockRuntime struct {
	utilruntime.ContainerRuntime
}

func (m *mockRuntime) ListKubeContainers() ([]string, error) {
	return []string{"container1", "container2"}, nil
}

func (m *mockRuntime) RemoveContainers(containers []string) error {
	return nil
}

func TestRemoveContainers(t *testing.T) {
	p := gomonkey.NewPatches()
	defer p.Reset()

	t.Run("WithProvidedSocket", func(t *testing.T) {
		p1 := gomonkey.ApplyFunc(utilruntime.NewContainerRuntime,
			func(criSocketPath string) utilruntime.ContainerRuntime {
				r := &mockRuntime{}
				return r
			})
		defer p1.Reset()

		err := RemoveContainers("/test/socket")
		if err != nil {
			t.Errorf("Expected no error with provided socket, got: %v", err)
		}
	})

	t.Run("WithDetectedSocket", func(t *testing.T) {
		p1 := gomonkey.ApplyFunc(utilruntime.DetectCRISocket,
			func() (string, error) {
				return "/detected/socket", nil
			})
		defer p1.Reset()

		p2 := gomonkey.ApplyFunc(utilruntime.NewContainerRuntime,
			func(criSocketPath string) utilruntime.ContainerRuntime {
				r := &mockRuntime{}
				return r
			})
		defer p2.Reset()

		err := RemoveContainers("")
		if err != nil {
			t.Errorf("Expected no error with detected socket, got: %v", err)
		}
	})

	t.Run("ListContainersFails", func(t *testing.T) {
		mr := &mockRuntime{}

		p1 := gomonkey.ApplyFunc(utilruntime.NewContainerRuntime,
			func(criSocketPath string) utilruntime.ContainerRuntime {
				return mr
			})
		defer p1.Reset()

		p2 := gomonkey.ApplyMethod(reflect.TypeOf(mr), "ListKubeContainers",
			func(_ *mockRuntime) ([]string, error) {
				return nil, errors.New("mock list error")
			})
		defer p2.Reset()

		err := RemoveContainers("/test/socket")
		if err == nil {
			t.Error("Expected error when listing containers fails, got nil")
		}
	})

	t.Run("RemoveContainersFails", func(t *testing.T) {
		mr := &mockRuntime{}

		p1 := gomonkey.ApplyFunc(utilruntime.NewContainerRuntime,
			func(criSocketPath string) utilruntime.ContainerRuntime {
				return mr
			})
		defer p1.Reset()

		p2 := gomonkey.ApplyMethod(reflect.TypeOf(mr), "RemoveContainers",
			func(_ *mockRuntime, containers []string) error {
				return errors.New("mock remove error")
			})
		defer p2.Reset()

		err := RemoveContainers("/test/socket")
		if err == nil {
			t.Error("Expected error when removing containers fails, got nil")
		}
	})
}

func TestCleanDirectories(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	t.Run("DirectoriesDontExist", func(t *testing.T) {
		patches.ApplyFunc(os.Stat,
			func(string) (os.FileInfo, error) {
				return nil, errors.New("file not found")
			})

		patches.ApplyFunc(os.IsNotExist,
			func(error) bool {
				return true
			})

		if err := CleanDirectories(false); err != nil {
			t.Errorf("CleanDirectories(false) failed: %v", err)
		}

		if err := CleanDirectories(true); err != nil {
			t.Errorf("CleanDirectories(true) failed: %v", err)
		}
	})

	t.Run("DirectoriesExistAndRemoved", func(t *testing.T) {
		patches.Reset()

		patches.ApplyFunc(os.Stat,
			func(string) (os.FileInfo, error) {
				return nil, nil
			})

		patches.ApplyFunc(os.IsNotExist,
			func(error) bool {
				return false
			})

		patches.ApplyFunc(os.RemoveAll,
			func(string) error {
				return nil
			})

		if err := CleanDirectories(false); err != nil {
			t.Errorf("CleanDirectories(false) with existing dirs failed: %v", err)
		}

		if err := CleanDirectories(true); err != nil {
			t.Errorf("CleanDirectories(true) with existing dirs failed: %v", err)
		}
	})

	t.Run("DirectoriesExistButRemovalFails", func(t *testing.T) {
		patches.Reset()

		patches.ApplyFunc(os.Stat,
			func(string) (os.FileInfo, error) {
				return nil, nil
			})

		patches.ApplyFunc(os.IsNotExist,
			func(error) bool {
				return false
			})

		patches.ApplyFunc(os.RemoveAll,
			func(string) error {
				return errors.New("removal failed")
			})

		if err := CleanDirectories(false); err != nil {
			t.Errorf("CleanDirectories(false) with removal failure should still return nil, got: %v", err)
		}
	})
}
