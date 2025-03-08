package actions

import "fmt"

type SpecSerializer interface {
	GetSpec() any
}

func NewSpecSerializer(data []byte, serializeFn func(d []byte) (any, error),
) (SpecSerializer, error) {
	spec, err := serializeFn(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize the specm, err: %v", err)
	}
	return &cachedSpecSerializer{
		spec: spec,
	}, nil
}

type cachedSpecSerializer struct {
	spec any
}

func (s cachedSpecSerializer) GetSpec() any {
	return s.spec
}
