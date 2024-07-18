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
package certs

import (
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func ReadPEMFile(file string) (*pem.Block, error) {
	bff, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(bff)
	return p, nil
}

func WriteDERToPEMFile(file, t string, der []byte) (*pem.Block, error) {
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0722); err != nil {
		return nil, fmt.Errorf("failed to create dir %s, err: %v", dir, err)
	}
	block := &pem.Block{Type: t, Bytes: der}
	bff := pem.EncodeToMemory(block)
	if err := os.WriteFile(file, bff, 0622); err != nil {
		return nil, fmt.Errorf("failed to write file %s, err: %v", file, err)
	}
	return block, nil
}
