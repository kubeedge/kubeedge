package imitator

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "imitator-test")
	if err != nil {
		panic(err)
	}
	dao.Init(filepath.Join(dir, "edgecore.db"), &edgecorev1alpha2.MetaManager{Enable: true})
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// newTestImitator returns an imitator on an empty meta_v2 table.
func newTestImitator(t *testing.T) *imitator {
	if err := dao.GetDB().Exec("DELETE FROM meta_v2").Error; err != nil {
		t.Fatal(err)
	}
	return newV2Client().(*imitator)
}

func podMessage(op string, pod *unstructured.Unstructured) model.Message {
	return *model.NewMessage("").
		BuildRouter("edgecontroller", "resource", pod.GetNamespace()+"/pod/"+pod.GetName(), op).
		FillBody(pod.Object)
}

func fullPod(uid, rv string) *unstructured.Unstructured {
	pod := podWithoutRV(uid)
	pod.SetResourceVersion(rv)
	pod.SetLabels(map[string]string{"app": "web"})
	_ = unstructured.SetNestedField(pod.Object, "edge-node", "spec", "nodeName")
	return pod
}

// podWithoutRV is the body of the DELETE synccontroller.gcOrphanedObjectSync sends: no
// resourceVersion, labels or spec.
func podWithoutRV(uid string) *unstructured.Unstructured {
	pod := &unstructured.Unstructured{}
	pod.SetAPIVersion("v1")
	pod.SetKind("Pod")
	pod.SetNamespace("default")
	pod.SetName("web-0")
	pod.SetUID(k8stypes.UID("uid-" + uid))
	return pod
}

func storedPods(t *testing.T, s *imitator) []models.MetaV2 {
	resp, err := s.List(context.TODO(), "/core/v1/pods/default")
	if err != nil {
		t.Fatal(err)
	}
	return *resp.Kvs
}

func nextEvent(ch <-chan watch.Event) (watch.Event, bool) {
	select {
	case e := <-ch:
		return e, true
	case <-time.After(time.Second):
		return watch.Event{}, false
	}
}

// The watch opened after the pod's last update, so its revision is above the pod's
// resourceVersion: the case where a DELETE without resourceVersion reached no watch client.
func TestInjectDeleteWithoutRV(t *testing.T) {
	s := newTestImitator(t)
	s.Inject(podMessage(model.InsertOperation, fullPod("a", "10")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 20)

	go s.Inject(podMessage(model.DeleteOperation, podWithoutRV("a")))

	e, ok := nextEvent(ch)
	if !ok {
		t.Fatal("watch got no event for the delete without resourceVersion")
	}
	if e.Type != watch.Deleted {
		t.Fatalf("event type = %s, want DELETED", e.Type)
	}
	obj := e.Object.(*unstructured.Unstructured)
	if obj.GetLabels()["app"] != "web" || obj.GetResourceVersion() != "10" {
		t.Errorf("DELETED object is not the stored pod: labels %v, resourceVersion %q", obj.GetLabels(), obj.GetResourceVersion())
	}
	if n := len(storedPods(t, s)); n != 0 {
		t.Errorf("meta_v2 holds %d pods after the delete, want 0", n)
	}
}

// A DELETE for an older object under the same key must not remove the current one.
func TestInjectDeleteOtherUID(t *testing.T) {
	s := newTestImitator(t)
	s.Inject(podMessage(model.InsertOperation, fullPod("b", "30")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 0)

	for _, msg := range []model.Message{
		podMessage(model.DeleteOperation, podWithoutRV("a")),
		podMessage(model.DeleteOperation, fullPod("a", "25")),
	} {
		go s.Inject(msg)
		if e, ok := nextEvent(ch); ok {
			t.Errorf("watch got %s for a delete of another uid", e.Type)
		}
	}
	if pods := storedPods(t, s); len(pods) != 1 {
		t.Errorf("meta_v2 holds %d pods, want the current one", len(pods))
	}
}

// A DELETE that waited in cloudhub behind an un-ACKed update carries
// a resourceVersion below a watch opened meanwhile. The watch listed the pod, so it must get
// the DELETED.
func TestInjectDeleteBelowWatchRevision(t *testing.T) {
	s := newTestImitator(t)
	s.Inject(podMessage(model.InsertOperation, fullPod("a", "10")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 20)

	go s.Inject(podMessage(model.DeleteOperation, fullPod("a", "15")))

	e, ok := nextEvent(ch)
	if !ok {
		t.Fatal("watch got no event for the delete below its revision")
	}
	if e.Type != watch.Deleted {
		t.Fatalf("event type = %s, want DELETED", e.Type)
	}
	if rv := e.Object.(*unstructured.Unstructured).GetResourceVersion(); rv != "15" {
		t.Errorf("DELETED object has resourceVersion %q, want the message's 15", rv)
	}
	if n := len(storedPods(t, s)); n != 0 {
		t.Errorf("meta_v2 holds %d pods after the delete, want 0", n)
	}
}

// A DELETE that finds no row is filtered by resourceVersion, as before the UID check: a client
// may hold the object from a list served by the cloud, whose insert never reached the edge.
// A DELETE without resourceVersion that finds no row reaches no watch.
func TestInjectDeleteWithoutRow(t *testing.T) {
	tests := []struct {
		name  string
		first *unstructured.Unstructured
		then  *unstructured.Unstructured
		event bool
	}{
		{"delete without resourceVersion after one with", fullPod("a", "30"), podWithoutRV("a"), false},
		{"delete with resourceVersion after one without", podWithoutRV("a"), fullPod("a", "30"), true},
		{"delete with resourceVersion repeated", fullPod("a", "30"), fullPod("a", "30"), true},
		{"never stored", nil, fullPod("a", "30"), true},
		{"never stored, below the watch's revision", nil, fullPod("a", "15"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newTestImitator(t)
			if test.first != nil {
				s.Inject(podMessage(model.InsertOperation, fullPod("a", "10")))
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ch := s.Watch(ctx, "/core/v1/pods/default", 20)

			if test.first != nil {
				go s.Inject(podMessage(model.DeleteOperation, test.first))
				if e, ok := nextEvent(ch); !ok || e.Type != watch.Deleted {
					t.Fatalf("watch got %v (%v) for the first delete, want DELETED", e.Type, ok)
				}
			}
			go s.Inject(podMessage(model.DeleteOperation, test.then))
			e, ok := nextEvent(ch)
			if ok != test.event || ok && e.Type != watch.Deleted {
				t.Errorf("watch got %v (%v) for a delete that found no row, want DELETED: %v", e.Type, ok, test.event)
			}
		})
	}
}

// A DELETE that fails in the database still goes out, as before the UID check, rather than
// leaving the object in every watch client.
func TestInjectDeleteDBError(t *testing.T) {
	s := newTestImitator(t)
	s.Inject(podMessage(model.InsertOperation, fullPod("a", "10")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 20)

	if err := dao.GetDB().Exec("ALTER TABLE meta_v2 RENAME TO meta_v2_off").Error; err != nil {
		t.Fatal(err)
	}
	go s.Inject(podMessage(model.DeleteOperation, fullPod("a", "30")))
	e, ok := nextEvent(ch)
	if err := dao.GetDB().Exec("ALTER TABLE meta_v2_off RENAME TO meta_v2").Error; err != nil {
		t.Fatal(err)
	}
	if !ok || e.Type != watch.Deleted {
		t.Errorf("watch got %v (%v) for a delete that failed, want DELETED", e.Type, ok)
	}
}

// edgecontroller deletes a pod by name only, with no UID or resourceVersion, when the edge
// reports the status of a pod that the cloud no longer has under that UID. The row may hold
// the recreated pod, so the watch must not get it as DELETED. The row goes, as before.
func TestInjectDeleteWithoutUID(t *testing.T) {
	s := newTestImitator(t)
	s.Inject(podMessage(model.InsertOperation, fullPod("b", "10")))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 20)

	byName := podWithoutRV("")
	byName.SetUID("")
	go s.Inject(podMessage(model.DeleteOperation, byName))
	if e, ok := nextEvent(ch); ok {
		t.Errorf("watch got %s for a delete without UID", e.Type)
	}
	if n := len(storedPods(t, s)); n != 0 {
		t.Errorf("meta_v2 holds %d pods after the delete, want 0", n)
	}
}

// A pod recreated under the same key while a DELETE of the old UID is injected must stay:
// the UID check and the delete are one step, whichever of the two runs first.
func TestInjectDeleteOtherUIDConcurrent(t *testing.T) {
	s := newTestImitator(t)
	for i := 0; i < 300; i++ {
		if err := s.InsertOrUpdateObj(context.TODO(), fullPod("a", "10")); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			s.Inject(podMessage(model.DeleteOperation, podWithoutRV("a")))
		}()
		go func() {
			defer wg.Done()
			<-start
			_ = s.InsertOrUpdateObj(context.TODO(), fullPod("b", "20"))
		}()
		close(start)
		wg.Wait()
		if len(storedPods(t, s)) != 1 {
			t.Fatalf("round %d: the delete of uid-a removed the recreated pod", i)
		}
		_ = s.Delete(context.TODO(), "/core/v1/pods/default/web-0")
	}
}

// The stored pod swapped in for a DELETE without resourceVersion carries its podpatch, as
// Get serves it.
func TestInjectDeleteWithoutRVMergesPodPatch(t *testing.T) {
	s := newTestImitator(t)
	pod := fullPod("a", "10")
	_ = unstructured.SetNestedField(pod.Object, "Pending", "status", "phase")
	s.Inject(podMessage(model.InsertOperation, pod))
	patch := &models.Meta{Key: "default/podpatch/web-0", Type: "podpatch", Value: `{"status":{"phase":"Running"}}`}
	if err := dbclient.NewMetaService().InsertOrUpdate(patch); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dbclient.NewMetaService().DeleteMetaByKey(patch.Key) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 20)

	go s.Inject(podMessage(model.DeleteOperation, podWithoutRV("a")))
	e, ok := nextEvent(ch)
	if !ok {
		t.Fatal("watch got no event for the delete without resourceVersion")
	}
	phase, _, _ := unstructured.NestedString(e.Object.(*unstructured.Unstructured).Object, "status", "phase")
	if phase != "Running" {
		t.Errorf("DELETED pod has status.phase %q, want the podpatch's Running", phase)
	}
}

// The UID check reads the stored row without its podpatch: kubelet puts the pod's UID in
// every status patch, so an older pod's patch names the old UID, and a patch that fails to
// merge must not skip the check.
func TestInjectDeleteOtherUIDWithPodPatch(t *testing.T) {
	for name, value := range map[string]string{
		"older pod's patch":         `{"metadata":{"uid":"uid-a"},"status":{"phase":"Running"}}`,
		"patch that fails to merge": `{"$patch":"bogus"}`,
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestImitator(t)
			s.Inject(podMessage(model.InsertOperation, fullPod("b", "30")))
			patch := &models.Meta{Key: "default/podpatch/web-0", Type: "podpatch", Value: value}
			if err := dbclient.NewMetaService().InsertOrUpdate(patch); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = dbclient.NewMetaService().DeleteMetaByKey(patch.Key) }()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ch := s.Watch(ctx, "/core/v1/pods/default", 0)

			for _, msg := range []model.Message{
				podMessage(model.DeleteOperation, podWithoutRV("a")),
				podMessage(model.DeleteOperation, fullPod("a", "25")),
			} {
				go s.Inject(msg)
				if e, ok := nextEvent(ch); ok {
					t.Errorf("watch got %s for a delete of another uid", e.Type)
				}
			}
			if pods := storedPods(t, s); len(pods) != 1 {
				t.Errorf("meta_v2 holds %d pods, want the current one", len(pods))
			}
		})
	}
}

// A DELETE without resourceVersion for the stored pod removes it even when its podpatch
// fails to merge, and the DELETED then carries the DELETE's own body.
func TestInjectDeleteWithoutRVPodPatchFails(t *testing.T) {
	s := newTestImitator(t)
	s.Inject(podMessage(model.InsertOperation, fullPod("a", "10")))
	patch := &models.Meta{Key: "default/podpatch/web-0", Type: "podpatch", Value: `{"$patch":"bogus"}`}
	if err := dbclient.NewMetaService().InsertOrUpdate(patch); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dbclient.NewMetaService().DeleteMetaByKey(patch.Key) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := s.Watch(ctx, "/core/v1/pods/default", 20)

	go s.Inject(podMessage(model.DeleteOperation, podWithoutRV("a")))
	e, ok := nextEvent(ch)
	if !ok || e.Type != watch.Deleted {
		t.Fatalf("watch got %v (%v) for the delete without resourceVersion, want DELETED", e.Type, ok)
	}
	if uid := e.Object.(*unstructured.Unstructured).GetUID(); uid != "uid-a" {
		t.Errorf("DELETED object has uid %q, want the delete's uid-a", uid)
	}
	if n := len(storedPods(t, s)); n != 0 {
		t.Errorf("meta_v2 holds %d pods after the delete, want 0", n)
	}
}
