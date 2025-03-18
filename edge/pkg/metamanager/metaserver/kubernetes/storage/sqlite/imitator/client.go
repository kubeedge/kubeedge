package imitator

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	daov2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
)

// DefaultV2Client is the only one client. Because of v2Client
// maintainers the revision and message cache(todo), so we do not see
// there are multi-clients.
var DefaultV2Client = newV2Client()
var Versioner = storage.APIObjectVersioner{}

type Client interface {
	// This set of functions is for metamanager
	// Inject the msg to the backend storage
	Inject(msg model.Message)
	InsertOrUpdateObj(ctx context.Context, obj runtime.Object) error
	DeleteObj(ctx context.Context, obj runtime.Object) error
	InsertOrUpdatePassThroughObj(ctx context.Context, obj []byte, key string) error
	GetPassThroughObj(ctx context.Context, key string) ([]byte, error)

	GetRevision() uint64
	SetRevision(version interface{})

	// This set of functions for upper storage
	List(ctx context.Context, key string) (Resp, error)
	Get(ctx context.Context, key string) (Resp, error)
	Watch(ctx context.Context, key string, ResourceVersion uint64) <-chan watch.Event
}

type Resp struct {
	//TODO: change to []*MetaV2
	Kvs *[]daov2.MetaV2
	// synonymous with resource version
	Revision uint64
}

func newV2Client() Client {
	return &imitator{
		lock:      sync.RWMutex{},
		versioner: Versioner,
		codec:     unstructured.UnstructuredJSONScheme,
	}
}

// StorageInit must be called before using imitator storage (run metaserver or metamanager)
func StorageInit() {
	m := new(daov2.MetaV2)
	// get the most recent record as the init resource version
	_, err := dbm.DBAccess.QueryTable(daov2.NewMetaTableName).OrderBy("-" + daov2.RV).Limit(1).All(m)
	utilruntime.Must(err)
	DefaultV2Client.SetRevision(m.ResourceVersion)
}
