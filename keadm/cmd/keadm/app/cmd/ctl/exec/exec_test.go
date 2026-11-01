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

package exec

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func TestEdgePodExec(t *testing.T) {
	cmd := NewEdgePodExec()
	assert.Equal(t, "exec", cmd.Use)
	assert.Error(t, cmd.RunE(cmd, []string{}))

	opts := NewEdgePodExecOpts()
	outBuf, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	opts.Out, opts.ErrOut = outBuf, errBuf

	fakeClient := fake.NewSimpleClientset()
	p := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p.Reset()

	p.ApplyFunc(podExec, func(_ context.Context, _ kubernetes.Interface, _ string, _ []string, _ *PodExecOptions) (*types.ExecResponse, error) {
		return &types.ExecResponse{
			RunOutMessages: []string{"out\n"},
			RunErrMessages: []string{"err1\n"},
			ErrMessages:    []string{"err2"},
		}, nil
	})
	assert.NoError(t, opts.execPod([]string{"pod1", "ls"}))
	assert.Equal(t, "out\n", outBuf.String())
	assert.Equal(t, "err1\nerr2\n", errBuf.String())
}

func TestExecPodNilResponse(t *testing.T) {
	opts := NewEdgePodExecOpts()
	fakeClient := fake.NewSimpleClientset()
	p := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p.Reset()

	p.ApplyFunc(podExec, func(_ context.Context, _ kubernetes.Interface, _ string, _ []string, _ *PodExecOptions) (*types.ExecResponse, error) {
		return nil, nil
	})
	assert.NoError(t, opts.execPod([]string{"pod1", "ls"}))
}

func TestExecPodErrors(t *testing.T) {
	opts := NewEdgePodExecOpts()

	// Test KubeClient error
	p1 := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return nil, fmt.Errorf("kubeclient error")
	})
	assert.Error(t, opts.execPod([]string{"pod1", "ls"}))
	p1.Reset()

	// Test podExec error
	fakeClient := fake.NewSimpleClientset()
	p2 := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p2.Reset()

	p2.ApplyFunc(podExec, func(_ context.Context, _ kubernetes.Interface, _ string, _ []string, _ *PodExecOptions) (*types.ExecResponse, error) {
		return nil, fmt.Errorf("podExec error")
	})
	assert.Error(t, opts.execPod([]string{"pod1", "ls"}))
}
