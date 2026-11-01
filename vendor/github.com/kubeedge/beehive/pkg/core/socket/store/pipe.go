package store

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"
)

// PipeStore pipe store
type PipeStore struct {
	pipeMap          map[string]PipeInfo
	pipeMapLock      sync.RWMutex
	groupPipeMap     map[string]map[string]PipeInfo
	groupPipeMapLock sync.RWMutex
}

// NewPipeStore new pipe store
func NewPipeStore() *PipeStore {
	return &PipeStore{
		pipeMap:      make(map[string]PipeInfo),
		groupPipeMap: make(map[string]map[string]PipeInfo),
	}
}

// Add add
func (s *PipeStore) Add(module string, pipe interface{}) {
	s.pipeMapLock.Lock()
	defer s.pipeMapLock.Unlock()
	s.pipeMap[module] = PipeInfo{pipe: pipe}
}

// Delete delete
func (s *PipeStore) Delete(module string) {
	// delete module conn from conn map
	s.pipeMapLock.Lock()
	_, exist := s.pipeMap[module]
	if !exist {
		klog.Warningf("failed to get pipe, module: %s", module)
		return
	}
	delete(s.pipeMap, module)
	s.pipeMapLock.Unlock()

	// delete module conn from group conn map
	s.groupPipeMapLock.Lock()
	for _, moduleMap := range s.groupPipeMap {
		if _, exist := moduleMap[module]; exist {
			delete(moduleMap, module)
			break
		}
	}
	s.groupPipeMapLock.Unlock()
}

// Get get
func (s *PipeStore) Get(module string) (PipeInfo, error) {
	s.pipeMapLock.RLock()
	defer s.pipeMapLock.RUnlock()

	if info, exist := s.pipeMap[module]; exist {
		return info, nil
	}
	return PipeInfo{}, fmt.Errorf("failed to get module(%s)", module)
}

// AddGroup add group
func (s *PipeStore) AddGroup(module, group string, pipe interface{}) {
	s.groupPipeMapLock.Lock()
	defer s.groupPipeMapLock.Unlock()

	if _, exist := s.groupPipeMap[group]; !exist {
		s.groupPipeMap[group] = make(map[string]PipeInfo)
	}
	s.groupPipeMap[group][module] = PipeInfo{pipe: pipe}
}

// GetGroup get group
func (s *PipeStore) GetGroup(group string) map[string]PipeInfo {
	s.groupPipeMapLock.RLock()
	defer s.groupPipeMapLock.RUnlock()

	if _, exist := s.groupPipeMap[group]; exist {
		return s.groupPipeMap[group]
	}
	klog.Warningf("failed to get group, type: %s", group)
	return nil
}

// WalkGroup walk group
func (s *PipeStore) WalkGroup(group string, walkFunc func(string, PipeInfo) error) error {
	s.groupPipeMapLock.RLock()
	defer s.groupPipeMapLock.RUnlock()

	if _, exist := s.groupPipeMap[group]; !exist {
		klog.Warningf("failed to get group, type: %s", group)
		return fmt.Errorf("failed to get group, type(%s)", group)
	}

	for module, pipe := range s.groupPipeMap[group] {
		err := walkFunc(module, pipe)
		if err != nil {
			return err
		}
	}

	return nil
}
