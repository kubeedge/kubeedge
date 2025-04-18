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

package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"k8s.io/klog/v2"
)

// FileCopy copy file from src to dst
func FileCopy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file %s stat, err: %v", src, err)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("source file %s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s, err: %v", src, err)
	}
	defer source.Close()

	// copy file using src file mode
	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return fmt.Errorf("failed to open or create destination file %s, err: %v", dst, err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

// GetSubDirs returns the subdirectories of the given directory.
// If sorted is true, the subdirectories are sorted by modification time.
func GetSubDirs(dir string, sorted bool) ([]string, error) {
	var subdirs []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %v", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, entry.Name())
		}
	}
	if !sorted {
		return subdirs, nil
	}

	sort.SliceStable(subdirs, func(i, j int) bool {
		infoI, err := os.Stat(filepath.Join(dir, subdirs[i]))
		if err != nil {
			klog.Errorf("failed to get file info of %s, err: %v", subdirs[i], err)
			return false
		}
		infoJ, err := os.Stat(filepath.Join(dir, subdirs[j]))
		if err != nil {
			klog.Errorf("failed to get file info of %s, err: %v", subdirs[j], err)
			return true
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return subdirs, nil
}
