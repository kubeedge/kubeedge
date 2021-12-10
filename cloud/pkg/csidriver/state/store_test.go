/*
Copyright 2019 The KubeEdge Authors.

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
package state

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestStore(t *testing.T) {
	RegisterTestingT(t)

	td, err := os.MkdirTemp("", "")
	Expect(err).ToNot(HaveOccurred())
	s, err := New(td)
	Expect(err).ToNot(HaveOccurred())

	err = s.Update("pvc-76fabc8a7", "node-1")
	Expect(err).ToNot(HaveOccurred())

	nodeID, err := s.Get("pvc-76fabc8a7")
	Expect(err).ToNot(HaveOccurred())
	Expect(nodeID).To(Equal("node-1"))

	err = s.Delete("pvc-76fabc8a7")
	Expect(err).ToNot(HaveOccurred())
	_, err = s.Get("pvc-76fabc8a7")
	Expect(err).To(MatchError(ErrNotExist))
}

func TestRecover(t *testing.T) {
	RegisterTestingT(t)
	stateToRestore := `{
		"volumes": {
			"pvc-76fabc8a7": "node-1"
		}
	  }`
	td, err := os.MkdirTemp("", "")
	Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(td, "volumes.json"), []byte(stateToRestore), os.ModePerm)
	Expect(err).ToNot(HaveOccurred())

	s, err := New(td)
	Expect(err).ToNot(HaveOccurred())
	nodeID, err := s.Get("pvc-76fabc8a7")
	Expect(err).ToNot(HaveOccurred())
	Expect(nodeID).To(Equal("node-1"))
}
