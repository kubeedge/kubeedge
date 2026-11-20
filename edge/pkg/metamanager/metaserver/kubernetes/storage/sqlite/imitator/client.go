package imitator

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// DefaultV2Client is the only one client. Because of v2Client
// maintainers the revision and message cache(todo), so we do not see
// there are multi-clients.
var DefaultV2Client = newV2Client()
var Versioner = storage.APIObjectVersioner{}

var registerMigrationOnce sync.Once

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
	Kvs *[]models.MetaV2
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
	meta, err := dbclient.NewMetaV2Service().GetLatestMetaV2()

	utilruntime.Must(err)

	DefaultV2Client.SetRevision(meta.ResourceVersion)
	klog.Infof("StorageInit set revision to: %d", meta.ResourceVersion)

	initRESTMapper(util.DefaultRESTMapper())
}

// initRESTMapper keeps the objects saved by the imitator under the resource
// known to mapper, and teaches mapper the mappings of the discovery documents
// cached by previous runs. Objects saved under a guessed resource by a previous
// version are moved as soon as the authoritative resource is learned.
func initRESTMapper(mapper *util.RESTMapper) {
	// StorageInit runs again when the module restarts, set up the mapper once.
	registerMigrationOnce.Do(func() {
		s, ok := DefaultV2Client.(*imitator)
		if !ok {
			return
		}
		// Mapping changes and the migration of the objects they move are
		// applied under s.lock, the lock objects are read and written under.
		mapper.SetChangeLocker(&s.lock)
		mapper.AddMappingChangedListener(s.migrateResourceLocked)
		// Load the saved list kinds before saving new ones.
		klog.Infof("StorageInit loaded %d saved list kinds", loadListKinds(mapper))
		mapper.AddListKindLearnedListener(s.saveListKind(mapper))
	})

	docs, err := dbclient.NewMetaV2Service().ListDiscoveryMetaV2()
	if err != nil {
		klog.Errorf("failed to load cached discovery documents: %v", err)
		return
	}
	learned := 0
	for _, doc := range *docs {
		learned += mapper.LearnFromDiscovery([]byte(doc.Value))
	}
	klog.Infof("StorageInit learned %d resource mappings from cached discovery documents", learned)
}
