package util

import (
	"k8s.io/kubernetes/pkg/kubelet/config"
)

type sourcesReady struct{}

func (s *sourcesReady) AddSource(source string) {
	return
}

func (s *sourcesReady) AllReady() bool {
	return true
}

//NewSourcesReady returns a new sourceready object
func NewSourcesReady() config.SourcesReady {
	return &sourcesReady{}
}
