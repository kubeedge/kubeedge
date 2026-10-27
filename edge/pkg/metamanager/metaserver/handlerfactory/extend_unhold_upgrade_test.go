/*
Copyright 2026 The KubeEdge Authors.

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

package handlerfactory

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

const (
	testNodeName = "edge-node"
	heldPodKey   = "default/held-pod"
)

func TestUnholdUpgrade(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		pods     []runtime.Object
		cliErr   error
		wantCode int
		wantBody string
		wantSent []string
	}{
		{
			name:     "body is not namespace/name",
			body:     "held-pod",
			wantCode: http.StatusBadRequest,
			wantBody: "invalid format, expected <namespace>/<name>",
		},
		{
			name:     "failed to init kube client",
			body:     heldPodKey,
			cliErr:   errors.New("test error"),
			wantCode: http.StatusInternalServerError,
			wantBody: "cannot init kube client: test error",
		},
		{
			name:     "pod not found",
			body:     heldPodKey,
			wantCode: http.StatusNotFound,
			wantBody: "pod not found",
		},
		{
			name:     "pod is not held",
			body:     "default/pod",
			pods:     []runtime.Object{newUnholdTestPod("default", "pod", testNodeName, "", v1.PodPending)},
			wantCode: http.StatusBadRequest,
			wantBody: "pod is not marked with hold-upgrade annotation",
		},
		{
			name:     "hold-upgrade annotation is not true",
			body:     "default/pod",
			pods:     []runtime.Object{newUnholdTestPod("default", "pod", testNodeName, "false", v1.PodPending)},
			wantCode: http.StatusBadRequest,
			wantBody: "pod is not marked with hold-upgrade annotation",
		},
		{
			name:     "pod is not pending",
			body:     heldPodKey,
			pods:     []runtime.Object{newUnholdTestPod("default", "held-pod", testNodeName, "true", v1.PodRunning)},
			wantCode: http.StatusBadRequest,
			wantBody: "pod is not in pending phase",
		},
		{
			name:     "unhold held pod",
			body:     "kube-system/held-pod",
			pods:     []runtime.Object{newUnholdTestPod("kube-system", "held-pod", testNodeName, "true", v1.PodPending)},
			wantCode: http.StatusOK,
			wantSent: []string{"kube-system/pod/held-pod"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sent := patchUnholdUpgrade(t, fake.NewSimpleClientset(c.pods...), c.cliErr)

			w := serveUnholdUpgrade(NewFactory().UnholdUpgrade(), c.body)
			assert.Equal(t, c.wantCode, w.Code)
			assert.Contains(t, w.Body.String(), c.wantBody)
			assert.Equal(t, c.wantSent, *sent)
		})
	}

	t.Run("body is too large", func(t *testing.T) {
		sent := patchUnholdUpgrade(t, fake.NewSimpleClientset(), nil)

		w := serveUnholdUpgrade(NewFactory().UnholdUpgrade(), strings.Repeat("a", 3*1024*1024+1))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "too large")
		assert.Empty(t, *sent)
	})
}

func TestUnholdUpgradeNode(t *testing.T) {
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: testNodeName}}

	cases := []struct {
		name     string
		body     string
		objs     []runtime.Object
		cliErr   error
		wantCode int
		wantBody string
		wantSent []string
	}{
		{
			name:     "node name is empty",
			body:     " \n",
			wantCode: http.StatusBadRequest,
			wantBody: "node name required in body",
		},
		{
			name:     "failed to init kube client",
			body:     testNodeName,
			cliErr:   errors.New("test error"),
			wantCode: http.StatusInternalServerError,
			wantBody: "cannot init kube client: test error",
		},
		{
			name:     "node not found",
			body:     "other-node",
			objs:     []runtime.Object{node},
			wantCode: http.StatusNotFound,
			wantBody: "node not found",
		},
		{
			name:     "no held pods on the node",
			body:     testNodeName,
			objs:     []runtime.Object{node, newUnholdTestPod("default", "pod", testNodeName, "", v1.PodPending)},
			wantCode: http.StatusOK,
		},
		{
			// Only pending pods with hold-upgrade=true that run on the node are unheld.
			// The node name in the body is trimmed.
			name: "unhold held pods on the node",
			body: "  " + testNodeName + "\n",
			objs: []runtime.Object{
				node,
				newUnholdTestPod("default", "held-1", testNodeName, "true", v1.PodPending),
				newUnholdTestPod("kube-system", "held-2", testNodeName, "true", v1.PodPending),
				newUnholdTestPod("default", "held-running", testNodeName, "true", v1.PodRunning),
				newUnholdTestPod("default", "not-held", testNodeName, "", v1.PodPending),
				newUnholdTestPod("default", "hold-false", testNodeName, "false", v1.PodPending),
				newUnholdTestPod("default", "held-other-node", "other-node", "true", v1.PodPending),
			},
			wantCode: http.StatusOK,
			wantSent: []string{"default/pod/held-1", "kube-system/pod/held-2"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sent := patchUnholdUpgrade(t, newPodsByNodeClientset(c.objs...), c.cliErr)

			w := serveUnholdUpgrade(NewFactory().UnholdUpgradeNode(), c.body)
			assert.Equal(t, c.wantCode, w.Code)
			assert.Contains(t, w.Body.String(), c.wantBody)
			assert.ElementsMatch(t, c.wantSent, *sent)
		})
	}

	t.Run("failed to list pods", func(t *testing.T) {
		cli := fake.NewSimpleClientset(node)
		cli.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("test error")
		})
		sent := patchUnholdUpgrade(t, cli, nil)

		w := serveUnholdUpgrade(NewFactory().UnholdUpgradeNode(), testNodeName)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "failed to list pods: test error")
		assert.Empty(t, *sent)
	})

	t.Run("body is too large", func(t *testing.T) {
		sent := patchUnholdUpgrade(t, newPodsByNodeClientset(node), nil)

		w := serveUnholdUpgrade(NewFactory().UnholdUpgradeNode(), strings.Repeat("a", 3*1024*1024+1))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "too large")
		assert.Empty(t, *sent)
	})
}

// patchUnholdUpgrade makes the handlers use cli, or fail to create a kube client with cliErr,
// and returns the resources of the unhold-upgrade messages sent to edged.
func patchUnholdUpgrade(t *testing.T, cli kubernetes.Interface, cliErr error) *[]string {
	var sent []string
	patches := gomonkey.NewPatches()
	t.Cleanup(patches.Reset)

	patches.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		if cliErr != nil {
			return nil, cliErr
		}
		return cli, nil
	})
	patches.ApplyFunc(beehiveContext.Send, func(module string, msg model.Message) {
		assert.Equal(t, modules.EdgedModuleName, module)
		assert.Equal(t, model.UnholdUpgradeOperation, msg.GetOperation())
		assert.Equal(t, modules.MetaManagerModuleName, msg.GetSource())
		sent = append(sent, msg.GetResource())
	})
	return &sent
}

func serveUnholdUpgrade(handler http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/unhold-upgrade", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// newPodsByNodeClientset returns a fake clientset that applies the spec.nodeName field selector
// when listing pods, which the fake object tracker ignores, so a missing or wrong selector shows up.
func newPodsByNodeClientset(objs ...runtime.Object) *fake.Clientset {
	cli := fake.NewSimpleClientset(objs...)
	cli.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		selector := action.(clienttesting.ListAction).GetListRestrictions().Fields
		list := &v1.PodList{}
		for _, obj := range objs {
			if pod, ok := obj.(*v1.Pod); ok && selector.Matches(fields.Set{"spec.nodeName": pod.Spec.NodeName}) {
				list.Items = append(list.Items, *pod)
			}
		}
		return true, list, nil
	})
	return cli
}

func newUnholdTestPod(namespace, name, nodeName, holdUpgrade string, phase v1.PodPhase) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec:       v1.PodSpec{NodeName: nodeName},
		Status:     v1.PodStatus{Phase: phase},
	}
	if holdUpgrade != "" {
		pod.Annotations = map[string]string{"edge.kubeedge.io/hold-upgrade": holdUpgrade}
	}
	return pod
}
