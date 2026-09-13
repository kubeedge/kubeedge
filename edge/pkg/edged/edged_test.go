/*
Copyright 2025 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package edged

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/config"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
	kubeletutil "k8s.io/kubernetes/pkg/kubelet/util"
)

const testNodeName = "test-node"

func newTestPod(namespace, name, uid string, phase v1.PodPhase, hold bool) v1.Pod {
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       types.UID(uid),
		},
		Spec: v1.PodSpec{
			NodeName: testNodeName,
			Containers: []v1.Container{{
				Name:  "c",
				Image: "busybox",
			}},
		},
		Status: v1.PodStatus{Phase: phase},
	}
	if hold {
		pod.Annotations = map[string]string{holdUpgradeLabel: "true"}
	}
	return pod
}

func marshalPodList(t *testing.T, pods ...v1.Pod) []byte {
	t.Helper()

	lists := make([]string, 0, len(pods))
	for _, pod := range pods {
		data, err := json.Marshal(pod)
		if err != nil {
			t.Fatalf("failed to marshal pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
		lists = append(lists, string(data))
	}

	content, err := json.Marshal(lists)
	if err != nil {
		t.Fatalf("failed to marshal pod list: %v", err)
	}
	return content
}

// TestHandlePodListFromMetaManagerHoldsPodAsUpdate makes sure a held pod is kept out of the
// pod list snapshot and is queued as an UPDATE, like the holdUpgrade path does.
func TestHandlePodListFromMetaManagerHoldsPodAsUpdate(t *testing.T) {
	heldPod := newTestPod("default", "held", "held-uid", v1.PodPending, true)
	runningPod := newTestPod("default", "running", "running-uid", v1.PodRunning, false)
	otherNodePod := newTestPod("default", "other", "other-uid", v1.PodRunning, false)
	otherNodePod.Spec.NodeName = "another-node"

	e := &edged{
		nodeName:       testNodeName,
		heldPodUpdates: make(map[string][]kubelettypes.PodUpdate),
	}

	updatesChan := make(chan interface{}, 1)
	if err := e.handlePodListFromMetaManager(marshalPodList(t, heldPod, runningPod, otherNodePod), updatesChan); err != nil {
		t.Fatalf("handlePodListFromMetaManager failed: %v", err)
	}

	update := (<-updatesChan).(kubelettypes.PodUpdate)
	if update.Op != kubelettypes.SET {
		t.Errorf("expected the pod list to be sent as SET, got %v", update.Op)
	}
	if len(update.Pods) != 1 || update.Pods[0].Name != runningPod.Name {
		t.Errorf("expected the pod list to contain only %q, got %v", runningPod.Name, update.Pods)
	}

	held, exists := e.heldPodUpdates["default/held"]
	if !exists {
		t.Fatal("expected the held pod to be cached in heldPodUpdates")
	}
	if len(held) != 1 {
		t.Fatalf("expected 1 held update, got %d", len(held))
	}
	if held[0].Op != kubelettypes.UPDATE {
		t.Errorf("expected the held pod to be queued as UPDATE, got %v, kubelet reads a SET as the full pod list of the source", held[0].Op)
	}
	if held[0].Source != kubelettypes.ApiserverSource {
		t.Errorf("expected source %q, got %q", kubelettypes.ApiserverSource, held[0].Source)
	}
	if len(held[0].Pods) != 1 || held[0].Pods[0].Name != heldPod.Name {
		t.Errorf("expected the held update to carry %q, got %v", heldPod.Name, held[0].Pods)
	}
}

// TestUnholdHeldPodFromPodListKeepsOtherPods replays a held pod through kubelet's PodConfig,
// the way the unhold-upgrade path does, and makes sure the pods already known to kubelet are
// not removed while the held pod is still delivered.
func TestUnholdHeldPodFromPodListKeepsOtherPods(t *testing.T) {
	heldPod := newTestPod("default", "held", "held-uid", v1.PodPending, true)
	runningPod := newTestPod("default", "running", "running-uid", v1.PodRunning, false)

	podCfg := config.NewPodConfig(config.PodConfigNotificationIncremental,
		record.NewFakeRecorder(100), kubeletutil.NewPodStartupLatencyTracker())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rawUpdateChan := podCfg.Channel(ctx, kubelettypes.ApiserverSource)

	e := &edged{
		nodeName:       testNodeName,
		heldPodUpdates: make(map[string][]kubelettypes.PodUpdate),
	}

	if err := e.handlePodListFromMetaManager(marshalPodList(t, heldPod, runningPod), rawUpdateChan); err != nil {
		t.Fatalf("handlePodListFromMetaManager failed: %v", err)
	}

	add := receivePodUpdate(t, podCfg)
	if add.Op != kubelettypes.ADD || len(add.Pods) != 1 || add.Pods[0].Name != runningPod.Name {
		t.Fatalf("expected %q to be added, got %v with pods %v", runningPod.Name, add.Op, add.Pods)
	}

	// Replay the held pod, as the unhold-upgrade path in syncPod does.
	for _, update := range e.heldPodUpdates["default/held"] {
		rawUpdateChan <- update
	}

	replayed := receivePodUpdate(t, podCfg)
	if replayed.Op == kubelettypes.REMOVE {
		t.Fatalf("replaying the held pod removed pods %v", replayed.Pods)
	}
	if replayed.Op != kubelettypes.ADD {
		t.Fatalf("expected the replayed held pod to be added, got %v", replayed.Op)
	}
	if len(replayed.Pods) != 1 || replayed.Pods[0].Name != heldPod.Name {
		t.Fatalf("expected the replay to carry %q, got %v", heldPod.Name, replayed.Pods)
	}

	if extra := receiveNoPodUpdate(t, podCfg); extra != nil {
		t.Fatalf("unexpected extra update after the replay: %v with pods %v", extra.Op, extra.Pods)
	}
}

func receivePodUpdate(t *testing.T, podCfg *config.PodConfig) kubelettypes.PodUpdate {
	t.Helper()

	select {
	case update := <-podCfg.Updates():
		return update
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for a pod update")
		return kubelettypes.PodUpdate{}
	}
}

func receiveNoPodUpdate(t *testing.T, podCfg *config.PodConfig) *kubelettypes.PodUpdate {
	t.Helper()

	select {
	case update := <-podCfg.Updates():
		return &update
	case <-time.After(time.Second):
		return nil
	}
}
