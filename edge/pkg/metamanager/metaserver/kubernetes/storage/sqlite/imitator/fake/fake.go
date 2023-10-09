package fake

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
)

// Client fake
type Client struct {
	InjectF                       func(msg model.Message)
	InsertOrUpdateObjF            func(ctx context.Context, obj runtime.Object) error
	DeleteObjF                    func(ctx context.Context, obj runtime.Object) error
	InsertOrUpdatePassThroughObjF func(ctx context.Context, obj []byte, key string) error
	GetPassThroughObjF            func(ctx context.Context, key string) ([]byte, error)
	GetRevisionF                  func() uint64
	SetRevisionF                  func(version interface{})
	ListF                         func(ctx context.Context, key string) (imitator.Resp, error)
	GetF                          func(ctx context.Context, key string) (imitator.Resp, error)
	WatchF                        func(ctx context.Context, key string, ResourceVersion uint64) <-chan watch.Event
}

// Inject fake
func (c Client) Inject(msg model.Message) {
	c.InjectF(msg)
}

// InsertOrUpdateObj fake
func (c Client) InsertOrUpdateObj(ctx context.Context, obj runtime.Object) error {
	return c.InsertOrUpdateObjF(ctx, obj)
}

// DeleteObj fake
func (c Client) DeleteObj(ctx context.Context, obj runtime.Object) error {
	return c.DeleteObjF(ctx, obj)
}

// InsertOrUpdatePassThroughObj fake
func (c Client) InsertOrUpdatePassThroughObj(ctx context.Context, obj []byte, key string) error {
	return c.InsertOrUpdatePassThroughObjF(ctx, obj, key)
}

// GetPassThroughObj fake
func (c Client) GetPassThroughObj(ctx context.Context, key string) ([]byte, error) {
	return c.GetPassThroughObjF(ctx, key)
}

// GetRevision fake
func (c Client) GetRevision() uint64 {
	return c.GetRevisionF()
}

// SetRevision fake
func (c Client) SetRevision(version interface{}) {
	c.SetRevisionF(version)
}

// List fake
func (c Client) List(ctx context.Context, key string) (imitator.Resp, error) {
	return c.ListF(ctx, key)
}

// Get fake
func (c Client) Get(ctx context.Context, key string) (imitator.Resp, error) {
	return c.GetF(ctx, key)
}

// Watch fake
func (c Client) Watch(ctx context.Context, key string, ResourceVersion uint64) <-chan watch.Event {
	return c.WatchF(ctx, key, ResourceVersion)
}
