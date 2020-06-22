package util

import (
	"k8s.io/kubernetes/pkg/kubelet/config"
)

type SourcesReadyFn func() bool

type sourcesReady struct {
	// sourcesReady is a function that evaluates if the sources are ready.
	sourcesReadyFn SourcesReadyFn
}

func (s *sourcesReady) AddSource(source string) {}

func (s *sourcesReady) AllReady() bool {
	return s.sourcesReadyFn()
}

//NewSourcesReady returns a new sourceready object
func NewSourcesReady(sourcesReadyFn SourcesReadyFn) config.SourcesReady {
	return &sourcesReady{
		sourcesReadyFn: sourcesReadyFn,
	}
}
