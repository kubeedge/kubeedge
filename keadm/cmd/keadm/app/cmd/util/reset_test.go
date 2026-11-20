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
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
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

func (m *mockRuntime) Connect() error {
	return nil
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

	mountedDirs, failingDirs := map[string]bool{}, map[string]bool{}
	unmountPatch := gomonkey.ApplyFunc(UnmountBelow, func(dir string) (bool, error) {
		if failingDirs[dir] {
			return false, errors.New("device or resource busy")
		}
		return mountedDirs[dir], nil
	})
	defer unmountPatch.Reset()

	var cleaned []string
	cleanPatch := gomonkey.ApplyFunc(phases.CleanDir, func(dir string) error {
		cleaned = append(cleaned, dir)
		return nil
	})
	defer cleanPatch.Reset()

	var edgecoreRunning bool
	var edgecoreCheckErr error
	runningPatch := gomonkey.ApplyFunc(IsKubeEdgeProcessRunning, func(string) (bool, error) {
		return edgecoreRunning, edgecoreCheckErr
	})
	defer runningPatch.Reset()

	t.Run("DirectoriesDontExist", func(t *testing.T) {
		patches.ApplyFunc(os.Stat,
			func(string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
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

	t.Run("MissingDirectoryDoesNotStopCleanup", func(t *testing.T) {
		patches.Reset()

		statCalls := 0

		patches.ApplyFunc(os.Stat,
			func(string) (os.FileInfo, error) {
				statCalls++
				if statCalls == 2 {
					return nil, os.ErrNotExist
				}
				return nil, nil
			})

		patches.ApplyFunc(os.IsNotExist,
			func(err error) bool {
				return errors.Is(err, os.ErrNotExist)
			})

		patches.ApplyFunc(os.RemoveAll,
			func(string) error {
				return nil
			})

		if err := CleanDirectories(false); err != nil {
			t.Errorf("CleanDirectories(false) failed: %v", err)
		}

		if statCalls != 4 {
			t.Fatalf("expected cleanup to continue checking remaining directories, got %d stat calls", statCalls)
		}
	})

	t.Run("UnmountsBeforeDeleting", func(t *testing.T) {
		patches.Reset()

		patches.ApplyFunc(os.Stat,
			func(string) (os.FileInfo, error) {
				return nil, nil
			})

		var removed []string
		patches.ApplyFunc(os.RemoveAll,
			func(path string) error {
				removed = append(removed, path)
				return nil
			})

		mountedDirs[constants.DefaultRootDir] = true
		failingDirs[common.EdgeRootDir] = true

		if err := CleanDirectories(true); err != nil {
			t.Fatalf("CleanDirectories(true) failed: %v", err)
		}
		wantRemoved := []string{constants.KubeEdgePath, constants.KubeEdgeLogPath, common.KubeEdgeSocketPath}
		if !reflect.DeepEqual(removed, wantRemoved) || !reflect.DeepEqual(cleaned, []string{constants.DefaultRootDir}) {
			t.Fatalf("expected %v removed and %s cleaned, got %v and %v", wantRemoved, constants.DefaultRootDir, removed, cleaned)
		}

		removed = nil
		if err := CleanDirectories(false); err != nil {
			t.Fatalf("CleanDirectories(false) failed: %v", err)
		}
		if want := append(wantRemoved, common.EdgeRootDir); !reflect.DeepEqual(removed, want) {
			t.Fatalf("expected cloud cleanup to remove %v without unmounting, got %v", want, removed)
		}

		timeout := edgeCoreStopTimeout
		defer func() { edgecoreRunning, edgecoreCheckErr, edgeCoreStopTimeout = false, nil, timeout }()
		removed, cleaned, edgecoreRunning, edgeCoreStopTimeout = nil, nil, true, time.Millisecond
		if err := CleanDirectories(true); err == nil || len(removed) != 0 || len(cleaned) != 0 {
			t.Fatalf("expected no cleanup while edgecore runs, got %v, removed %v, cleaned %v", err, removed, cleaned)
		}

		edgecoreRunning, edgecoreCheckErr = false, errors.New("pidof: command not found")
		if err := CleanDirectories(true); err == nil || len(removed) != 0 || len(cleaned) != 0 {
			t.Fatalf("expected no cleanup when edgecore cannot be checked, got %v, removed %v, cleaned %v", err, removed, cleaned)
		}
	})
}

func TestUnmountBelow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mount points are linux paths")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := root + "/kubelet"
	a, ab, c, outside := dir+"/a", dir+"/a/b", dir+"/c", root+"/outside/m"
	for _, d := range []string{ab, c, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	file := root + "/file"
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := root + "/link"
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root+"/outside", dir+"/s"); err != nil {
		t.Fatal(err)
	}
	escape := strings.NewReplacer(`\`, `\134`, " ", `\040`, "\t", `\011`, "\n", `\012`).Replace

	tests := []struct {
		name          string
		dir           string
		mounts        []string
		busy          map[string]bool
		peers         map[string]string
		unreadable    bool
		wantMounted   bool
		wantUnmounted []string
		wantErr       bool
		wantErrPaths  []string
	}{
		{name: "unmounts mount points below dir, deepest first", dir: dir, mounts: []string{"/", dir, a, ab, c, dir + "-other"},
			wantMounted: true, wantUnmounted: []string{ab, a, c}},
		{name: "retries a mount uncovered by unmounting the one on top", dir: dir, mounts: []string{a, ab, a},
			wantUnmounted: []string{ab, a, a, ab, a}},
		{name: "skips a mount already removed by propagation", dir: dir, mounts: []string{a, c}, peers: map[string]string{a: c},
			wantUnmounted: []string{a, c}},
		{name: "returns every error when no unmount succeeds", dir: dir, mounts: []string{a, c}, busy: map[string]bool{a: true, c: true},
			wantUnmounted: []string{a, c}, wantErr: true, wantErrPaths: []string{a, c}},
		{name: "skips a mount point reached through a symlink", dir: dir, mounts: []string{dir + "/s/m"},
			wantErr: true, wantErrPaths: []string{root + "/outside"}},
		{name: "unmounts a mount point whose own root cannot be read", dir: dir, mounts: []string{dir + "/stale"},
			wantUnmounted: []string{dir + "/stale"}},
		{name: "resolves an intermediate symlink", dir: link + "/a", mounts: []string{ab}, wantUnmounted: []string{ab}},
		{name: "refuses the filesystem root", dir: "/", mounts: []string{a}, wantErr: true},
		{name: "does not follow a final symlink", dir: link + "/", mounts: []string{a}, unreadable: true},
		{name: "missing dir", dir: root + "/missing", unreadable: true},
		{name: "path that is not a dir", dir: file + "/x", wantErr: true},
		{name: "unreadable mounts", dir: dir, unreadable: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := append([]string(nil), tt.mounts...)
			last := func(path string) int {
				for i := len(table) - 1; i >= 0; i-- {
					if table[i] == path {
						return i
					}
				}
				return -1
			}
			readMounts := func() ([]byte, error) {
				if tt.unreadable {
					return nil, errors.New("cannot read mounts")
				}
				var b strings.Builder
				for _, mp := range table {
					b.WriteString("none " + escape(mp) + " tmpfs rw 0 0\n")
				}
				return []byte(b.String()), nil
			}
			var unmounted []string
			unmount := func(path string) error {
				unmounted = append(unmounted, path)
				i := last(path)
				if i < 0 {
					return errors.New("invalid argument")
				}
				for _, mp := range table[i+1:] {
					if strings.HasPrefix(path, mp+"/") {
						return errors.New("no such file or directory")
					}
					if strings.HasPrefix(mp, path+"/") {
						return errors.New("device or resource busy")
					}
				}
				if tt.busy[path] {
					return errors.New("device or resource busy")
				}
				table = append(table[:i], table[i+1:]...)
				if j := last(tt.peers[path]); j >= 0 {
					table = append(table[:j], table[j+1:]...)
				}
				return nil
			}

			mounted, err := unmountBelow(tt.dir, readMounts, unmount)
			if mounted != tt.wantMounted || (err != nil) != tt.wantErr || !reflect.DeepEqual(unmounted, tt.wantUnmounted) {
				t.Fatalf("unmountBelow() = %v, %v, unmounted %v; want %v, error %v, unmounted %v",
					mounted, err, unmounted, tt.wantMounted, tt.wantErr, tt.wantUnmounted)
			}
			for _, path := range tt.wantErrPaths {
				if !strings.Contains(err.Error(), path) {
					t.Fatalf("expected the error to name %s, got %v", path, err)
				}
			}
		})
	}
}

func TestParseMountPoints(t *testing.T) {
	procMounts := "/dev/sda1 / ext4 rw 0 0\n" +
		"tmpfs /var/lib/kubelet/pods/uid/volumes/kubernetes.io~secret/token tmpfs rw 0 0\n" +
		"srv:/a\u00a0b /srv/node\\040data/kubelet nfs4 rw 0 0\n" +
		"tmpfs /tab\\011and\\134slash\\012 tmpfs rw 0 0\n" +
		"tmpfs /not\\999octal tmpfs rw 0 0\n"
	want := []string{"/", "/var/lib/kubelet/pods/uid/volumes/kubernetes.io~secret/token",
		"/srv/node data/kubelet", "/tab\tand\\slash\n", "/not\\999octal"}
	if got := parseMountPoints(procMounts); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseMountPoints() = %q, want %q", got, want)
	}
}
