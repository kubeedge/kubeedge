package endpointresource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	discoverylisters "k8s.io/client-go/listers/discovery/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

// Edge nodes keep the filtered copies of EndpointSlices that they already have
// until the slice gets a new resourceVersion: cloudhub and edgecore both
// deliver an object only when its resourceVersion is newer. So when an input
// of the filter changes, the slice must change in the cluster for the edge
// copies to be filtered again.
//
// The refilter controller does that. It writes a hash of the filter inputs into
// an annotation on every EndpointSlice of a scoped Service, and patches the
// annotation when the hash changes. The patch gives the slice a new
// resourceVersion, so it is sent to every edge node that watches it and
// filtered again on the way. Because the hash is stored on the slice, changes
// made while cloudcore was down are caught when it starts.
//
// core/v1 Endpoints are left out on purpose. The filter still filters an
// Endpoints object each time it is sent, but this controller does not re-send
// it when the filter inputs change, so an edge copy can stay stale until the
// object itself changes. The Endpoints API is deprecated as of Kubernetes
// 1.33: every patch of it logs a deprecation warning, and a wait for its cache
// would hold up the EndpointSlices too. kube-proxy and CoreDNS read only
// EndpointSlices.
//
// The inputs are:
//   - whether the Service is scoped (its annotation, the default topology, and
//     whether it has a selector);
//   - the NodeGroup of every node that is in one;
//   - the filter's behaviour, as filterVersion.
//
// A slice of a Service that is not scoped is sent unfiltered, so it carries no
// annotation. Then turning the feature on does not touch every slice in the
// cluster, and an upgrade from a cloudcore that never filtered patches only the
// slices of scoped Services.

const (
	// FilterInputsAnnotation holds the hash of the filter inputs that the
	// slice's edge copies were last filtered with.
	FilterInputsAnnotation = "apps.kubeedge.io/endpoint-filter-inputs"

	// filterVersion is part of the hash. Change it whenever a change to the
	// filter changes its result for the same inputs, so that the next cloudcore
	// re-sends every scoped slice.
	filterVersion = "1"

	// Each patch is sent to every edge node that watches the slice, and each
	// node's acknowledgement costs one ObjectSync status write, so patches are
	// rate limited.
	refilterPatchQPS   = 5
	refilterPatchBurst = 10
	refilterWorkers    = 2
)

type refilterKind string

const (
	kindEndpointSlice refilterKind = "EndpointSlice"
	// kindNodes is the queue item that recomputes the NodeGroup hash.
	kindNodes refilterKind = "Nodes"
)

type refilterKey struct {
	kind      refilterKind
	namespace string
	name      string
}

var nodesKey = refilterKey{kind: kindNodes}

// RefilterController patches the filter-inputs annotation on EndpointSlices
// when the inputs of the topology filter change.
type RefilterController struct {
	kubeClient      kubernetes.Interface
	defaultTopology string

	nodeLister    corelisters.NodeLister
	serviceLister corelisters.ServiceLister
	sliceLister   discoverylisters.EndpointSliceLister
	synced        []cache.InformerSynced

	queue   workqueue.TypedRateLimitingInterface[refilterKey]
	limiter *rate.Limiter

	// groupsHash is the hash of the NodeGroup of every node that is in one.
	groupsLock sync.RWMutex
	groupsHash string
}

// NewRefilterController builds the controller and registers its event handlers
// on the informers of factory. The caller starts factory.
func NewRefilterController(kubeClient kubernetes.Interface, factory informers.SharedInformerFactory,
	defaultTopology string) (*RefilterController, error) {
	nodes := factory.Core().V1().Nodes()
	services := factory.Core().V1().Services()
	slices := factory.Discovery().V1().EndpointSlices()

	c := &RefilterController{
		kubeClient:      kubeClient,
		defaultTopology: defaultTopology,
		nodeLister:      nodes.Lister(),
		serviceLister:   services.Lister(),
		sliceLister:     slices.Lister(),
		synced: []cache.InformerSynced{
			nodes.Informer().HasSynced,
			services.Informer().HasSynced,
			slices.Informer().HasSynced,
		},
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.NewTypedItemExponentialFailureRateLimiter[refilterKey](time.Second, 5*time.Minute),
			workqueue.TypedRateLimitingQueueConfig[refilterKey]{Name: "endpoint-refilter"}),
		limiter: rate.NewLimiter(refilterPatchQPS, refilterPatchBurst),
	}

	if _, err := nodes.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.onNode,
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Queue directly: a node that left its NodeGroup has none now,
			// and onNode would ignore it.
			if nodeGroupOf(oldObj) != nodeGroupOf(newObj) {
				c.queue.Add(nodesKey)
			}
		},
		DeleteFunc: c.onNode,
	}); err != nil {
		return nil, err
	}
	if _, err := services.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.onService,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldSvc, ok1 := oldObj.(*v1.Service)
			newSvc, ok2 := newObj.(*v1.Service)
			if ok1 && ok2 && isScoped(oldSvc, defaultTopology) != isScoped(newSvc, defaultTopology) {
				c.onService(newObj)
			}
		},
	}); err != nil {
		return nil, err
	}
	if _, err := slices.Informer().AddEventHandler(objectHandler(c.queue, kindEndpointSlice)); err != nil {
		return nil, err
	}
	return c, nil
}

func objectHandler(queue workqueue.TypedRateLimitingInterface[refilterKey], kind refilterKind) cache.ResourceEventHandler {
	enqueue := func(obj interface{}) {
		if m, err := meta.Accessor(obj); err == nil {
			queue.Add(refilterKey{kind: kind, namespace: m.GetNamespace(), name: m.GetName()})
		}
	}
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    enqueue,
		UpdateFunc: func(_, newObj interface{}) { enqueue(newObj) },
	}
}

func nodeGroupOf(obj interface{}) string {
	if node, ok := obj.(*v1.Node); ok {
		return node.Labels[nodegroup.LabelBelongingTo]
	}
	return ""
}

func (c *RefilterController) onNode(obj interface{}) {
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = tombstone.Obj
	}
	if nodeGroupOf(obj) != "" {
		c.queue.Add(nodesKey)
	}
}

func (c *RefilterController) onService(obj interface{}) {
	svc, ok := obj.(*v1.Service)
	if !ok {
		return
	}
	c.enqueueService(svc.Namespace, svc.Name)
}

func (c *RefilterController) enqueueService(namespace, name string) {
	selector := labels.SelectorFromSet(labels.Set{discovery.LabelServiceName: name})
	slices, err := c.sliceLister.EndpointSlices(namespace).List(selector)
	if err != nil {
		klog.Errorf("refilter: list EndpointSlices of Service %s/%s: %v", namespace, name, err)
	}
	for _, slice := range slices {
		c.queue.Add(refilterKey{kind: kindEndpointSlice, namespace: namespace, name: slice.Name})
	}
}

func (c *RefilterController) enqueueAll() {
	slices, err := c.sliceLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("refilter: list EndpointSlices: %v", err)
	}
	for _, slice := range slices {
		c.queue.Add(refilterKey{kind: kindEndpointSlice, namespace: slice.Namespace, name: slice.Name})
	}
}

// Run waits for the caches, then processes the queue until ctx is done.
func (c *RefilterController) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("refilter: waiting for the node, Service and EndpointSlice caches")
	if !cache.WaitForCacheSync(ctx.Done(), c.synced...) {
		klog.Error("refilter: caches did not sync, edge copies will not be re-filtered when filter inputs change")
		return
	}
	// Compute the NodeGroup hash before any object is checked. The node events
	// seen so far put nodesKey in the queue; it finds no change.
	if _, err := c.updateGroupsHash(); err != nil {
		klog.Errorf("refilter: %v", err)
		return
	}
	klog.Info("refilter: started")

	for i := 0; i < refilterWorkers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}
	<-ctx.Done()
}

func (c *RefilterController) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *RefilterController) processNextItem(ctx context.Context) bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	if err := c.sync(ctx, key); err != nil {
		klog.Warningf("refilter: %s %s/%s: %v", key.kind, key.namespace, key.name, err)
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *RefilterController) sync(ctx context.Context, key refilterKey) error {
	if key.kind == kindNodes {
		changed, err := c.updateGroupsHash()
		if err != nil {
			return err
		}
		if changed {
			c.enqueueAll()
		}
		return nil
	}

	want, current, skip, err := c.wantedAnnotation(key)
	if err != nil || skip || want == current {
		return err
	}
	// Wait for a token only when a patch is due, then check again, because the
	// inputs may have changed while waiting.
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}
	want, current, skip, err = c.wantedAnnotation(key)
	if err != nil || skip || want == current {
		return err
	}
	return c.patch(ctx, key, want)
}

// wantedAnnotation returns the value the filter-inputs annotation should have
// and the value it has. An empty value means no annotation. skip is true when
// the slice is gone, or its Service is not in the cache yet: the Service's
// add event will queue the slice again.
func (c *RefilterController) wantedAnnotation(key refilterKey) (want, current string, skip bool, err error) {
	if key.kind != kindEndpointSlice {
		return "", "", true, nil
	}
	slice, err := c.sliceLister.EndpointSlices(key.namespace).Get(key.name)
	if err != nil {
		return "", "", apierrors.IsNotFound(err), ignoreNotFound(err)
	}
	current = slice.Annotations[FilterInputsAnnotation]
	serviceName := slice.Labels[discovery.LabelServiceName]

	// The filter does not scope a slice that names no Service.
	if serviceName == "" {
		return "", current, false, nil
	}
	svc, err := c.serviceLister.Services(key.namespace).Get(serviceName)
	if err != nil {
		return "", current, apierrors.IsNotFound(err), ignoreNotFound(err)
	}
	if !isScoped(svc, c.defaultTopology) {
		return "", current, false, nil
	}
	return c.filterInputsHash(), current, false, nil
}

func (c *RefilterController) patch(ctx context.Context, key refilterKey, value string) error {
	var annotation interface{} = value
	if value == "" {
		annotation = nil // a merge patch removes a key set to null
	}
	data, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{FilterInputsAnnotation: annotation},
		},
	})
	if err != nil {
		return err
	}
	_, err = c.kubeClient.DiscoveryV1().EndpointSlices(key.namespace).
		Patch(ctx, key.name, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		return ignoreNotFound(err)
	}
	klog.V(2).Infof("refilter: patched EndpointSlice %s/%s so that edge nodes filter it again", key.namespace, key.name)
	return nil
}

func (c *RefilterController) filterInputsHash() string {
	c.groupsLock.RLock()
	defer c.groupsLock.RUnlock()
	return filterVersion + "-" + c.groupsHash
}

// updateGroupsHash recomputes the NodeGroup hash and reports whether it changed.
func (c *RefilterController) updateGroupsHash() (bool, error) {
	nodes, err := c.nodeLister.List(labels.Everything())
	if err != nil {
		return false, fmt.Errorf("list nodes: %v", err)
	}
	hash := hashNodeGroups(nodes)

	c.groupsLock.Lock()
	defer c.groupsLock.Unlock()
	if hash == c.groupsHash {
		return false, nil
	}
	c.groupsHash = hash
	return true, nil
}

// hashNodeGroups hashes the NodeGroup of every node that is in one. A node in
// no NodeGroup sees only its own endpoints, so it is left out: a node that
// joins the cluster without a NodeGroup changes no other node's view.
func hashNodeGroups(nodes []*v1.Node) string {
	pairs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if group := node.Labels[nodegroup.LabelBelongingTo]; group != "" {
			pairs = append(pairs, node.Name+"\x00"+group)
		}
	}
	sort.Strings(pairs)
	h := sha256.New()
	for _, p := range pairs {
		h.Write([]byte(p))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func ignoreNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
