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

package logs

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

func TestEdgePodLogs(t *testing.T) {
	cmd := NewEdgePodLogs()
	assert.Equal(t, "logs", cmd.Use)
	assert.Error(t, cmd.RunE(cmd, []string{}))

	opts := NewLogsPodOpts()
	outBuf, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	opts.Out, opts.ErrOut = outBuf, errBuf

	fakeClient := fake.NewSimpleClientset()
	p := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p.Reset()

	p.ApplyFunc(logsPod, func(_ context.Context, _ kubernetes.Interface, _ string, _ *PodLogsOptions) (*types.LogsResponse, error) {
		return &types.LogsResponse{LogMessages: []string{"line1\n"}, ErrMessages: []string{"warn"}}, nil
	})
	assert.NoError(t, opts.getPodLogs([]string{"pod1"}))
	assert.Equal(t, "line1\n", outBuf.String())
	assert.Equal(t, "warn\n", errBuf.String())
}

func TestLogsPodFollow(t *testing.T) {
	outBuf := &bytes.Buffer{}
	opts := NewLogsPodOpts()
	opts.Out, opts.Follow = outBuf, true

	fakeClient := fake.NewSimpleClientset()
	p := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p.Reset()

	p.ApplyFunc(logsPod, func(_ context.Context, _ kubernetes.Interface, _ string, o *PodLogsOptions) (*types.LogsResponse, error) {
		fmt.Fprint(o.Out, "streamed\n")
		return nil, nil
	})

	assert.NoError(t, opts.getPodLogs([]string{"pod1"}))
	assert.Equal(t, "streamed\n", outBuf.String())
}

func TestGetPodLogsNilResponse(t *testing.T) {
	opts := NewLogsPodOpts()
	fakeClient := fake.NewSimpleClientset()
	p := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p.Reset()

	p.ApplyFunc(logsPod, func(_ context.Context, _ kubernetes.Interface, _ string, _ *PodLogsOptions) (*types.LogsResponse, error) {
		return nil, nil
	})
	assert.NoError(t, opts.getPodLogs([]string{"pod1"}))
}

func TestGetPodLogsErrors(t *testing.T) {
	opts := NewLogsPodOpts()

	// Test KubeClient error
	p1 := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) {
		return nil, fmt.Errorf("kubeclient error")
	})
	assert.Error(t, opts.getPodLogs([]string{"pod1"}))
	p1.Reset()

	// Test logsPod error
	fakeClient := fake.NewSimpleClientset()
	p2 := gomonkey.ApplyFunc(metaclient.KubeClient, func() (kubernetes.Interface, error) { return fakeClient, nil })
	defer p2.Reset()

	p2.ApplyFunc(logsPod, func(_ context.Context, _ kubernetes.Interface, _ string, _ *PodLogsOptions) (*types.LogsResponse, error) {
		return nil, fmt.Errorf("logsPod error")
	})
	assert.Error(t, opts.getPodLogs([]string{"pod1"}))
}
