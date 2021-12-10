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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	basePath string

	mux   *sync.RWMutex
	state *State
}

type State struct {
	// store which volume id belongs to which edgenode
	// this is needed for correct routing
	Volumes map[string]string `json:"volumes"`
}

const (
	stateFileName = "volumes.json"
)

var ErrNotExist = errors.New("volume does not exist")

func New(basePath string) (*Store, error) {
	s := &Store{
		basePath: basePath,
		mux:      &sync.RWMutex{},
		state: &State{
			Volumes: map[string]string{},
		},
	}

	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create base state path: %v", err)
	}

	return s, s.recover()
}

func (s *Store) recover() error {
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, stateFileName))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("error reading state: %v", err)
	}
	if err := json.Unmarshal(data, &s.state); err != nil {
		return fmt.Errorf("error decoding state file: %v", err)
	}
	return nil
}

func (s *Store) save() error {
	data, err := json.Marshal(&s.state)
	if err != nil {
		return fmt.Errorf("error encoding state: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(s.basePath, stateFileName), data, 0600); err != nil {
		return fmt.Errorf("error writing state file: %v", err)
	}
	return nil
}

func (s *Store) Get(volumeID string) (string, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	nodeID, ok := s.state.Volumes[volumeID]
	if !ok {
		return "", ErrNotExist
	}
	return nodeID, nil
}

func (s *Store) Update(volumeID, nodeID string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.state.Volumes[volumeID] = nodeID
	return s.save()
}

func (s *Store) Delete(volumeID string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	delete(s.state.Volumes, volumeID)
	return s.save()
}
