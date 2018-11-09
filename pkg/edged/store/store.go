package store

import (
	"kubeedge/pkg/metamanager/dao"
)

type UpdateFunc func(input interface{}) (output interface{}, ttl uint64, err error)

type FilterFunc func(obj interface{}) bool

type BackendStore interface {
	Create(key string, kind string, obj interface{}, output bool, ttl uint64) (interface{}, error)
	Delete(key string, kind string, obj interface{}) error
	Get(key string, kind string, obj interface{}) (interface{}, error)
	List(key string, kind string, obj interface{}, filter FilterFunc) (interface{}, error)
	Update(key string, kind string, obj interface{}, tryUpdate UpdateFunc) (interface{}, error)
}
type store struct {
	pathPrefix string
}

// New returns an etcd3 implementation of storage.Interface.
func NewStore(prefix string) BackendStore {
	return &store{}
}

func (s *store) Create(key string, kind string, obj interface{}, output bool, ttl uint64) (interface{}, error) {
	return nil, nil
}

func (s *store) Delete(key string, kind string, obj interface{}) error {
	return dao.DeleteMetaByKey(key)
}

func (s *store) Get(key string, kind string, obj interface{}) (interface{}, error) {
	return dao.QueryMeta(key, kind)
}

func (s *store) List(key string, kind string, obj interface{}, filter FilterFunc) (interface{}, error) {
	return dao.QueryMeta(key, kind)
}

func (s *store) Update(key string, kind string, obj interface{}, tryUpdate UpdateFunc) (interface{}, error) {
	return nil, nil
}
