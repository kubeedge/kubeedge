package imitator

import (
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apiserver/pkg/storage"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func newTestImitator() *imitator {
	return &imitator{
		lock:      sync.RWMutex{},
		versioner: storage.APIObjectVersioner{},
		codec:     unstructured.UnstructuredJSONScheme,
	}
}

func TestEventPatchesMissingKindFromResource(t *testing.T) {
	s := newTestImitator()
	msg := &model.Message{}
	msg.BuildRouter("devicecontroller", "resource", "node/edge-node/default/device/test-device", model.UpdateOperation)
	msg.Content = []byte(`{"metadata":{"name":"test-device","namespace":"default"},"spec":{"nodeName":"edge-node"}}`)

	events := s.Event(msg)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Object.GetObjectKind().GroupVersionKind().Kind != "Device" {
		t.Fatalf("unexpected kind %q", events[0].Object.GetObjectKind().GroupVersionKind().Kind)
	}
}

func TestEventWithJSONArrayPayload(t *testing.T) {
	s := newTestImitator()
	msg := &model.Message{}
	msg.BuildRouter("devicecontroller", "resource", "default/device/test-device", model.UpdateOperation)
	msg.Content = []byte(`[{"metadata":{"name":"test-device","namespace":"default"}}]`)

	events := s.Event(msg)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for array payload, got %d", len(events))
	}
}

func TestEventDoesNotPatchMissingKindForNonResourceGroup(t *testing.T) {
	s := newTestImitator()
	msg := &model.Message{}
	msg.BuildRouter("devicecontroller", "twin", "node/edge-node/membership/detail", model.UpdateOperation)
	msg.Content = []byte(`{"event_id":"1","timestamp":1}`)

	events := s.Event(msg)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for non-resource group payload, got %d", len(events))
	}
}

func TestEventPatchesNodePayloadTypeMetaFromResource(t *testing.T) {
	s := newTestImitator()
	msg := &model.Message{}
	msg.BuildRouter("edgecontroller", "resource", "node/edge-node/default/node/edge-node", model.UpdateOperation)
	msg.Content = []byte(`{"metadata":{"name":"edge-node"},"spec":{},"status":{"conditions":[{"type":"Ready","status":"True"}]}}`)

	events := s.Event(msg)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	gvk := events[0].Object.GetObjectKind().GroupVersionKind()
	if gvk.Kind != "Node" {
		t.Fatalf("unexpected kind %q", gvk.Kind)
	}
	if gvk.Version != "v1" {
		t.Fatalf("unexpected version %q", gvk.Version)
	}
}

func TestEventSkipsDeleteOptionsPayload(t *testing.T) {
	s := newTestImitator()
	msg := &model.Message{}
	msg.BuildRouter("edgecontroller", "resource", "node/edge-node/default/pod/test-pod", model.DeleteOperation)
	msg.Content = []byte(`{"gracePeriodSeconds":0,"preconditions":{"uid":"7ce1efe9-9796-4125-a175-c181dfa7a78a"}}`)

	events := s.Event(msg)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for delete options payload, got %d", len(events))
	}
}

func TestParseResourceForNodeResourcePath(t *testing.T) {
	_, resType, resID := parseResource("node/edge-node/default/device")
	if resType != resourceTypeDevice {
		t.Fatalf("unexpected resource type %q", resType)
	}
	if resID != "" {
		t.Fatalf("unexpected resource id %q", resID)
	}

	_, resType, resID = parseResource("node/edge-node/default/device/test-device")
	if resType != resourceTypeDevice {
		t.Fatalf("unexpected resource type %q", resType)
	}
	if resID != "test-device" {
		t.Fatalf("unexpected resource id %q", resID)
	}
}
