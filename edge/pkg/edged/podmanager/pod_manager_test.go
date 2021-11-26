package podmanager

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetSetPods(t *testing.T) {
	testPod1 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("edged-test-pod1"),
			Name:      "edged-test-pod1",
			Namespace: metav1.NamespaceDefault,
		},
	}

	testPod2 := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("edged-test-pod2"),
			Name:      "edged-test-pod2",
			Namespace: metav1.NamespaceDefault,
		},
	}

	expectedPods := []*v1.Pod{
		testPod1,
		testPod2,
	}

	edgedManager := NewEdgedPodManager().(*edgedManager)
	edgedManager.SetPods(expectedPods)

	actualPods := edgedManager.GetPods()
	if len(actualPods) != len(expectedPods) {
		t.Errorf("expected %d pods, god %d pods; expected pods %#v, got pods %#v", len(expectedPods), len(actualPods),
			expectedPods, actualPods)
	}
	for _, expected := range expectedPods {
		found := false
		for _, actual := range actualPods {
			if actual.UID == expected.UID {
				if !reflect.DeepEqual(&expected, &actual) {
					t.Errorf("pod was recorded incorrectly. expect: %#v, got: %#v", expected, actual)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("pod %q was not found in %#v", expected.UID, actualPods)
		}
	}

	actualPod, ok := edgedManager.GetPodByFullName("edged-test-pod1_default")
	if !ok || !reflect.DeepEqual(actualPod, testPod1) {
		t.Errorf("unable to get pod by full name; expected: %#v, got: %#v", testPod1, actualPod)
	}
	actualPod, ok = edgedManager.GetPodByName("default", "edged-test-pod2")
	if !ok || !reflect.DeepEqual(actualPod, testPod2) {
		t.Errorf("unable to get pod by full name; expected: %#v, got: %#v", testPod2, actualPod)
	}
}

func TestDeletePods(t *testing.T) {
	pods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("edged-test-pod1"),
				Name:      "edged-test-pod1",
				Namespace: metav1.NamespaceDefault,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("edged-test-pod2"),
				Name:      "edged-test-pod2",
				Namespace: metav1.NamespaceDefault,
			},
		},
	}

	deletePod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID("edged-test-pod2"),
			Name:      "edged-test-pod2",
			Namespace: metav1.NamespaceDefault,
		},
	}

	expectedPods := []*v1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       types.UID("edged-test-pod1"),
				Name:      "edged-test-pod1",
				Namespace: metav1.NamespaceDefault,
			},
		},
	}

	edgedManager := NewEdgedPodManager().(*edgedManager)
	edgedManager.SetPods(pods)

	edgedManager.DeletePod(deletePod)

	actualPods := edgedManager.GetPods()
	if len(actualPods) != len(expectedPods) {
		t.Errorf("expected %d pods, god %d pods; expected pods %#v, got pods %#v", len(expectedPods), len(actualPods),
			expectedPods, actualPods)
	}
}
