package nodegroup

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

const testNodeGroup = "site-a"

// newFinalizerTestController records the NodeGroup patches, decoded, and fails on a full
// Update. The fake client stores Go types, so it cannot show a dropped omitempty value
// itself; the test checks the write that would drop one: a locked patch with no spec.
func newFinalizerTestController(t *testing.T, ng *appsv1alpha1.NodeGroup) (*Controller, *[]map[string]map[string]interface{}) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := appsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	var patches []map[string]map[string]interface{}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ng).
		WithStatusSubresource(&appsv1alpha1.NodeGroup{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*appsv1alpha1.NodeGroup); ok {
					t.Errorf("full Update of NodeGroup %s", obj.GetName())
				}
				return c.Update(ctx, obj, opts...)
			},
			Patch: func(ctx context.Context, c client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				if _, ok := obj.(*appsv1alpha1.NodeGroup); ok {
					data, err := patch.Data(obj)
					if err != nil {
						t.Fatal(err)
					}
					var p map[string]map[string]interface{}
					if err := json.Unmarshal(data, &p); err != nil {
						t.Fatalf("patch %s: %v", data, err)
					}
					patches = append(patches, p)
				}
				return c.Patch(ctx, obj, patch, opts...)
			},
		}).Build()
	return NewController(cli), &patches
}

// checkFinalizerPatch fails unless the only patch touched metadata.finalizers alone, under a
// resourceVersion lock, and the stored NodeGroup, which it returns, has the finalizers given.
func checkFinalizerPatch(t *testing.T, c *Controller, patches []map[string]map[string]interface{}, want ...string) *appsv1alpha1.NodeGroup {
	t.Helper()
	if len(patches) != 1 {
		t.Fatalf("got %d NodeGroup patches, want 1: %v", len(patches), patches)
	}
	for key := range patches[0] {
		if key != "metadata" {
			t.Errorf("patch sets %q: %v", key, patches[0])
		}
	}
	for key := range patches[0]["metadata"] {
		if key != "finalizers" && key != "resourceVersion" {
			t.Errorf("patch sets metadata.%s: %v", key, patches[0])
		}
	}
	if _, ok := patches[0]["metadata"]["resourceVersion"]; !ok {
		t.Errorf("patch has no resourceVersion lock: %v", patches[0])
	}
	got := &appsv1alpha1.NodeGroup{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Name: testNodeGroup}, got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Finalizers, want) {
		t.Errorf("stored finalizers %v, want %v", got.Finalizers, want)
	}
	return got
}

func TestFinalizerIsPatched(t *testing.T) {
	ng := &appsv1alpha1.NodeGroup{
		ObjectMeta: v1.ObjectMeta{Name: testNodeGroup},
		Spec:       appsv1alpha1.NodeGroupSpec{MatchLabels: map[string]string{}, TopologyEnabled: false},
	}
	c, patches := newFinalizerTestController(t, ng)
	ctx := context.Background()
	req := controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testNodeGroup}}

	if _, err := c.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	got := checkFinalizerPatch(t, c, *patches, NodeGroupControllerFinalizer)

	*patches = nil
	if err := c.removeFinalizer(ctx, got); err != nil {
		t.Fatal(err)
	}
	checkFinalizerPatch(t, c, *patches)
}

func TestFinalizerPatchConflictsOnStaleRead(t *testing.T) {
	ng := &appsv1alpha1.NodeGroup{
		ObjectMeta: v1.ObjectMeta{Name: testNodeGroup, Finalizers: []string{NodeGroupControllerFinalizer}},
	}
	c, _ := newFinalizerTestController(t, ng)
	ctx := context.Background()
	stale := &appsv1alpha1.NodeGroup{}
	if err := c.Client.Get(ctx, types.NamespacedName{Name: testNodeGroup}, stale); err != nil {
		t.Fatal(err)
	}
	// another writer changes the NodeGroup after the read
	fresh := stale.DeepCopy()
	fresh.Labels = map[string]string{"changed": "yes"}
	if err := c.Client.Patch(ctx, fresh, client.MergeFrom(stale)); err != nil {
		t.Fatal(err)
	}
	if err := c.removeFinalizer(ctx, stale); !apierrors.IsConflict(err) {
		t.Fatalf("removeFinalizer on a stale read: got %v, want a conflict", err)
	}
}
