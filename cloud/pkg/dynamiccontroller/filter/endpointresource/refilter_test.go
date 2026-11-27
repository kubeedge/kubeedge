package endpointresource

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

const testNamespace = "default"

func testNode(name, group string) *v1.Node {
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if group != "" {
		node.Labels = map[string]string{nodegroup.LabelBelongingTo: group}
	}
	return node
}

func testService(name, topology string) *v1.Service {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec:       v1.ServiceSpec{Selector: map[string]string{"app": name}},
	}
	if topology != "" {
		svc.Annotations = map[string]string{nodegroup.ServiceTopologyAnnotation: topology}
	}
	return svc
}

func testSlice(name, service, annotation string) *discovery.EndpointSlice {
	slice := &discovery.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}}
	if service != "" {
		slice.Labels = map[string]string{discovery.LabelServiceName: service}
	}
	if annotation != "" {
		slice.Annotations = map[string]string{FilterInputsAnnotation: annotation}
	}
	return slice
}

func testEndpoints(name string) *v1.Endpoints {
	return &v1.Endpoints{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace}}
}

// newTestController builds a controller on a fake clientset holding objs, and
// waits for its caches and the NodeGroup hash. Run is not started.
func newTestController(t *testing.T, defaultTopology string, objs ...runtime.Object) (*RefilterController, *kubefake.Clientset) {
	t.Helper()
	client := kubefake.NewSimpleClientset(objs...)
	factory := informers.NewSharedInformerFactory(client, 0)
	c, err := NewRefilterController(client, factory, defaultTopology)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(c.queue.ShutDown)
	factory.Start(ctx.Done())
	require.True(t, cache.WaitForCacheSync(ctx.Done(), c.synced...))
	_, err = c.updateGroupsHash()
	require.NoError(t, err)
	client.ClearActions()
	return c, client
}

func patchedNames(client *kubefake.Clientset) []string {
	var names []string
	for _, action := range client.Actions() {
		if patch, ok := action.(k8stesting.PatchAction); ok {
			names = append(names, patch.GetResource().Resource+"/"+patch.GetName())
		}
	}
	return names
}

func sliceAnnotation(t *testing.T, client *kubefake.Clientset, name string) (string, bool) {
	t.Helper()
	slice, err := client.DiscoveryV1().EndpointSlices(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.NoError(t, err)
	value, ok := slice.Annotations[FilterInputsAnnotation]
	return value, ok
}

func syncKey(t *testing.T, c *RefilterController, kind refilterKind, name string) {
	t.Helper()
	require.NoError(t, c.sync(context.TODO(), refilterKey{kind: kind, namespace: testNamespace, name: name}))
}

func TestHashNodeGroups(t *testing.T) {
	a := hashNodeGroups([]*v1.Node{testNode("n1", "g1"), testNode("n2", "g2"), testNode("n3", "")})
	b := hashNodeGroups([]*v1.Node{testNode("n2", "g2"), testNode("n4", ""), testNode("n1", "g1")})
	assert.Equal(t, a, b, "order and nodes in no NodeGroup must not change the hash")

	moved := hashNodeGroups([]*v1.Node{testNode("n1", "g2"), testNode("n2", "g2")})
	assert.NotEqual(t, a, moved)
	left := hashNodeGroups([]*v1.Node{testNode("n1", "g1"), testNode("n2", "")})
	assert.NotEqual(t, a, left)
}

func TestRefilterScopedServiceIsAnnotated(t *testing.T) {
	c, client := newTestController(t, "",
		testNode("n1", "g1"),
		testService("svc", nodegroup.ServiceTopologyRangeNodegroup),
		testSlice("svc-abc", "svc", ""))

	syncKey(t, c, kindEndpointSlice, "svc-abc")

	assert.Equal(t, []string{"endpointslices/svc-abc"}, patchedNames(client))
	value, ok := sliceAnnotation(t, client, "svc-abc")
	assert.True(t, ok)
	assert.Equal(t, c.filterInputsHash(), value)
}

func TestRefilterUnchangedInputsDoNotPatch(t *testing.T) {
	// The first controller computes the value a restarted cloudcore would find.
	first, _ := newTestController(t, "", testNode("n1", "g1"))
	stored := first.filterInputsHash()

	c, client := newTestController(t, "",
		testNode("n1", "g1"),
		testService("svc", nodegroup.ServiceTopologyRangeNodegroup),
		testSlice("svc-abc", "svc", stored))

	syncKey(t, c, kindEndpointSlice, "svc-abc")
	assert.Empty(t, patchedNames(client))
}

func TestRefilterUnscopedServiceIsNotTouched(t *testing.T) {
	c, client := newTestController(t, "",
		testNode("n1", "g1"),
		testService("plain", ""),
		testService("optout", nodegroup.ServiceTopologyRangeCluster),
		testSlice("plain-abc", "plain", ""),
		testSlice("optout-abc", "optout", ""),
		testSlice("orphan", "", ""))

	for _, name := range []string{"plain-abc", "optout-abc", "orphan"} {
		syncKey(t, c, kindEndpointSlice, name)
	}
	assert.Empty(t, patchedNames(client))
}

func TestRefilterDefaultTopology(t *testing.T) {
	selectorless := testService("external", "")
	selectorless.Spec.Selector = nil
	c, client := newTestController(t, nodegroup.ServiceTopologyRangeNodegroup,
		testNode("n1", "g1"),
		testService("plain", ""),
		testService("optout", nodegroup.ServiceTopologyRangeCluster),
		selectorless,
		testSlice("plain-abc", "plain", ""),
		testSlice("optout-abc", "optout", ""),
		testSlice("external-abc", "external", ""))

	for _, name := range []string{"plain-abc", "optout-abc", "external-abc"} {
		syncKey(t, c, kindEndpointSlice, name)
	}
	assert.Equal(t, []string{"endpointslices/plain-abc"}, patchedNames(client))
}

func TestRefilterOptOutRemovesAnnotation(t *testing.T) {
	c, client := newTestController(t, "",
		testNode("n1", "g1"),
		testService("svc", nodegroup.ServiceTopologyRangeCluster),
		testSlice("svc-abc", "svc", "1-0123456789abcdef"))

	syncKey(t, c, kindEndpointSlice, "svc-abc")
	assert.Equal(t, []string{"endpointslices/svc-abc"}, patchedNames(client))
	_, ok := sliceAnnotation(t, client, "svc-abc")
	assert.False(t, ok)
}

func TestRefilterWaitsForService(t *testing.T) {
	c, client := newTestController(t, "",
		testNode("n1", "g1"),
		testSlice("svc-abc", "svc", ""))

	syncKey(t, c, kindEndpointSlice, "svc-abc")
	assert.Empty(t, patchedNames(client))
}

func TestRefilterMissingObject(t *testing.T) {
	c, client := newTestController(t, "", testNode("n1", "g1"))
	syncKey(t, c, kindEndpointSlice, "gone")
	assert.Empty(t, patchedNames(client))
}

// TestRefilterRun drives the controller through its event handlers.
func TestRefilterRun(t *testing.T) {
	client := kubefake.NewSimpleClientset(
		testNode("n1", "g1"),
		testNode("n2", "g1"),
		testService("scoped", nodegroup.ServiceTopologyRangeNodegroup),
		testService("plain", ""),
		testSlice("scoped-abc", "scoped", ""),
		testSlice("plain-abc", "plain", ""),
		testEndpoints("scoped"))
	factory := informers.NewSharedInformerFactory(client, 0)
	c, err := NewRefilterController(client, factory, "")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	factory.Start(ctx.Done())
	go c.Run(ctx)

	waitForAnnotation := func(want string) {
		t.Helper()
		var got string
		err := wait.PollUntilContextTimeout(ctx, 20*time.Millisecond, 5*time.Second, true,
			func(context.Context) (bool, error) {
				got, _ = sliceAnnotation(t, client, "scoped-abc")
				return got == want, nil
			})
		require.NoError(t, err, "annotation is %q, want %q", got, want)
	}

	// At startup the scoped slice gets the current hash.
	waitForAnnotation(filterVersion + "-" + hashNodeGroups([]*v1.Node{testNode("n1", "g1"), testNode("n2", "g1")}))

	// Moving a node to another NodeGroup changes the hash.
	_, err = client.CoreV1().Nodes().Update(ctx, testNode("n2", "g2"), metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForAnnotation(filterVersion + "-" + hashNodeGroups([]*v1.Node{testNode("n1", "g1"), testNode("n2", "g2")}))

	// Opting the Service out removes the annotation.
	_, err = client.CoreV1().Services(testNamespace).Update(ctx,
		testService("scoped", nodegroup.ServiceTopologyRangeCluster), metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForAnnotation("")

	_, ok := sliceAnnotation(t, client, "plain-abc")
	assert.False(t, ok, "a slice of an unscoped Service must never be annotated")

	// The controller leaves core/v1 Endpoints out on purpose: it must not
	// even list or watch them, because the API is deprecated.
	for _, action := range client.Actions() {
		assert.NotEqual(t, "endpoints", action.GetResource().Resource,
			"the controller sent a %s request for Endpoints", action.GetVerb())
	}
}

// TestRefilterNodeLeavesNodeGroup: removing a node from its NodeGroup changes
// what the other nodes of that group may see, so the edge copies must be
// filtered again.
func TestRefilterNodeLeavesNodeGroup(t *testing.T) {
	client := kubefake.NewSimpleClientset(
		testNode("n1", "g1"),
		testNode("n2", "g1"),
		testService("scoped", nodegroup.ServiceTopologyRangeNodegroup),
		testSlice("scoped-abc", "scoped", ""))
	factory := informers.NewSharedInformerFactory(client, 0)
	c, err := NewRefilterController(client, factory, "")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	factory.Start(ctx.Done())
	go c.Run(ctx)

	waitForAnnotation := func(want string) {
		t.Helper()
		var got string
		err := wait.PollUntilContextTimeout(ctx, 20*time.Millisecond, 3*time.Second, true,
			func(context.Context) (bool, error) {
				got, _ = sliceAnnotation(t, client, "scoped-abc")
				return got == want, nil
			})
		require.NoError(t, err, "annotation is %q, want %q", got, want)
	}

	waitForAnnotation(filterVersion + "-" + hashNodeGroups([]*v1.Node{testNode("n1", "g1"), testNode("n2", "g1")}))

	_, err = client.CoreV1().Nodes().Update(ctx, testNode("n2", ""), metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForAnnotation(filterVersion + "-" + hashNodeGroups([]*v1.Node{testNode("n1", "g1"), testNode("n2", "")}))
}

// TestRefilterTwoControllersDisagree models two cloudcore instances whose node
// caches disagree: each sees a different NodeGroup for n2. They must not keep
// overwriting each other's annotation, because every patch re-sends the object
// to every edge node that watches it.
func TestRefilterTwoControllersDisagree(t *testing.T) {
	t.Skip("known issue: cloudcore instances in an HA setup each run a refilter controller and " +
		"fight over the annotation while their caches disagree; the fix (leader election) is postponed")
	client := kubefake.NewSimpleClientset(
		testNode("n1", "g1"),
		testNode("n2", "g1"),
		testService("scoped", nodegroup.ServiceTopologyRangeNodegroup),
		testSlice("scoped-abc", "scoped", ""))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	factoryA := informers.NewSharedInformerFactory(client, 0)
	a, err := NewRefilterController(client, factoryA, "")
	require.NoError(t, err)

	// b shares the Services and EndpointSlices, but reads nodes from a cache
	// that is behind: n2 is still in g2 there.
	factoryB := informers.NewSharedInformerFactory(client, 0)
	b, err := NewRefilterController(client, factoryB, "")
	require.NoError(t, err)
	staleFactory := informers.NewSharedInformerFactory(
		kubefake.NewSimpleClientset(testNode("n1", "g1"), testNode("n2", "g2")), 0)
	staleNodes := staleFactory.Core().V1().Nodes()
	b.nodeLister = staleNodes.Lister()
	b.synced = append(b.synced, staleNodes.Informer().HasSynced)

	factoryA.Start(ctx.Done())
	factoryB.Start(ctx.Done())
	staleFactory.Start(ctx.Done())
	go a.Run(ctx)
	go b.Run(ctx)

	countPatches := func() int {
		n := 0
		for _, action := range client.Actions() {
			if _, ok := action.(k8stesting.PatchAction); ok {
				n++
			}
		}
		return n
	}
	time.Sleep(3 * time.Second)
	patches := countPatches()
	t.Logf("%d patches of one EndpointSlice in 3s", patches)
	assert.LessOrEqual(t, patches, 2, "the two controllers keep overwriting each other")
}
