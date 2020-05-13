package cache

import lru "github.com/hashicorp/golang-lru"

const DefaultCapacity = 20

// meshCache is the lru cache to store edgemesh meta
var meshCache *lru.Cache

func init() {
	var err error
	meshCache, err = lru.New(DefaultCapacity)
	if err != nil {
		panic(err)
	}
}

func GetMeshCache() *lru.Cache {
	return meshCache
}
