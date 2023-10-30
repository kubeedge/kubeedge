package manager

import (
	"sync"
)

// MapperManager is a manager for map from mapper to node
type MapperManager struct {
	// Mapper2NodeMap, key is mapper.Name, value is node.Name
	// TODO mapper.Name may be repeated on multiple nodes
	Mapper2NodeMap sync.Map

	// NodeMapperList stores the mapper list deployed on the corresponding node, key is node.Name, value is []*types.Mapper{}
	NodeMapperList sync.Map
}

// NewMapperManager is function to return new MapperManager
func NewMapperManager() *MapperManager {
	return &MapperManager{}
}
