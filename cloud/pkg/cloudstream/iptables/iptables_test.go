package iptables

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	clientgov1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
)

func newTunnelPortConfigMap(t *testing.T, record *TunnelPortRecord) *v1.ConfigMap {
	t.Helper()

	recordBytes, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("failed to marshal record: %v", err)
	}

	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modules.TunnelPort,
			Namespace: constants.SystemNamespace,
			Annotations: map[string]string{
				modules.TunnelPortRecordAnnotationKey: string(recordBytes),
			},
		},
	}
}

func newConfigMapLister(t *testing.T, cms ...*v1.ConfigMap) clientgov1.ConfigMapLister {
	t.Helper()

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, cm := range cms {
		if err := indexer.Add(cm); err != nil {
			t.Fatalf("failed to add configmap to indexer: %v", err)
		}
	}

	return clientgov1.NewConfigMapLister(indexer)
}

func newTestManager(lister clientgov1.ConfigMapLister, pre *TunnelPortRecord) *Manager {
	if pre == nil {
		pre = &TunnelPortRecord{
			IPTunnelPort: map[string]int{},
			Port:         map[int]bool{},
		}
	}

	return &Manager{
		cmLister:            lister,
		preTunnelPortRecord: pre,
		streamPort:          10003,
	}
}

func TestGetLatestTunnelPortRecords_Success(t *testing.T) {
	want := &TunnelPortRecord{
		IPTunnelPort: map[string]int{
			"10.0.0.2": 10002,
		},
		Port: map[int]bool{
			10002: true,
		},
	}

	lister := newConfigMapLister(t, newTunnelPortConfigMap(t, want))
	im := newTestManager(lister, nil)

	got, err := im.getLatestTunnelPortRecords()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected record:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestGetLatestTunnelPortRecords_MissingAnnotation(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        modules.TunnelPort,
			Namespace:   constants.SystemNamespace,
			Annotations: map[string]string{},
		},
	}

	lister := newConfigMapLister(t, cm)
	im := newTestManager(lister, nil)

	_, err := im.getLatestTunnelPortRecords()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetLatestTunnelPortRecords_InvalidJSON(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modules.TunnelPort,
			Namespace: constants.SystemNamespace,
			Annotations: map[string]string{
				modules.TunnelPortRecordAnnotationKey: "{invalid-json",
			},
		},
	}

	lister := newConfigMapLister(t, cm)
	im := newTestManager(lister, nil)

	_, err := im.getLatestTunnelPortRecords()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetAddedAndDeletedCloudCoreIPPort(t *testing.T) {
	pre := &TunnelPortRecord{
		IPTunnelPort: map[string]int{
			"10.0.0.1": 10001,
			"10.0.0.2": 10002,
		},
		Port: map[int]bool{
			10001: true,
			10002: true,
		},
	}

	latest := &TunnelPortRecord{
		IPTunnelPort: map[string]int{
			"10.0.0.2": 10002,
			"10.0.0.3": 10003,
		},
		Port: map[int]bool{
			10002: true,
			10003: true,
		},
	}

	lister := newConfigMapLister(t, newTunnelPortConfigMap(t, latest))
	im := newTestManager(lister, pre)

	added, deleted, err := im.getAddedAndDeletedCloudCoreIPPort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(added)
	sort.Strings(deleted)

	wantAdded := []string{"10.0.0.2:10002", "10.0.0.3:10003"}
	wantDeleted := []string{"10.0.0.1:10001"}

	if !reflect.DeepEqual(wantAdded, added) {
		t.Fatalf("unexpected added list:\nwant: %#v\ngot:  %#v", wantAdded, added)
	}

	if !reflect.DeepEqual(wantDeleted, deleted) {
		t.Fatalf("unexpected deleted list:\nwant: %#v\ngot:  %#v", wantDeleted, deleted)
	}

	if !reflect.DeepEqual(im.preTunnelPortRecord, latest) {
		t.Fatalf("preTunnelPortRecord was not updated:\nwant: %#v\ngot:  %#v", latest, im.preTunnelPortRecord)
	}
}

func TestNewConfigMapLister_IsValid(t *testing.T) {
	cm := newTunnelPortConfigMap(t, &TunnelPortRecord{
		IPTunnelPort: map[string]int{"10.0.0.1": 10001},
		Port:         map[int]bool{10001: true},
	})

	lister := newConfigMapLister(t, cm)
	got, err := lister.ConfigMaps(constants.SystemNamespace).Get(modules.TunnelPort)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("expected configmap, got nil")
	}
}

var _ runtime.Object = &v1.ConfigMap{}
var _ informers.SharedInformerFactory
