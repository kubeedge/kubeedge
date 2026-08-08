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

package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"

	edgecoreconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/image"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestImagePrePullJobCheckItems(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				CheckItems: []string{"cpu", "mem", "disk"},
			},
		},
	}
	h := imagePrePullJobActionHandler{
		logger: klog.Background(),
	}

	t.Run("check items failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(PreCheck, func([]string) error {
			return errors.New("test error")
		})

		resp := h.checkItems(ctx, "", "", specser)
		require.EqualError(t, resp.Error(), "test error")
	})

	t.Run("check items success", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(PreCheck, func([]string) error {
			return nil
		})

		resp := h.checkItems(ctx, "", "", specser)
		require.NoError(t, resp.Error())
	})
}

func TestImagePrePullJobPullImages(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				Images: []string{"image1", "image2"},
			},
		},
	}
	h := imagePrePullJobActionHandler{
		logger: klog.Background(),
	}

	imagert := &image.RuntimeImpl{}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(options.GetEdgeCoreConfig, func() *edgecoreconfig.EdgeCoreConfig {
		return edgecoreconfig.NewDefaultEdgeCoreConfig()
	})
	patches.ApplyFunc(image.NewImageRuntime, func(_endpoint string, _timeout time.Duration,
	) (*image.RuntimeImpl, error) {
		return imagert, nil
	})
	patches.ApplyMethodFunc(reflect.TypeOf(imagert), "PullImage",
		func(_ctx context.Context,
			image string,
			_authConfig *runtimeapi.AuthConfig,
			_sandboxConfig *runtimeapi.PodSandboxConfig,
		) error {
			if image == "image2" {
				return errors.New("test error")
			}
			return nil
		})

	resp := h.pullImages(ctx, "", "", specser)
	require.ErrorContains(t, resp.Error(), pullImageFailureMessage)
	imagePrePullResp, ok := resp.(*imagePrePullJobActionResponse)
	assert.True(t, ok)
	assert.Equal(t, metav1.ConditionTrue, imagePrePullResp.imageStatus[0].Status)
	assert.Equal(t, metav1.ConditionFalse, imagePrePullResp.imageStatus[1].Status)
	assert.Equal(t, "test error", imagePrePullResp.imageStatus[1].Reason)
}

func TestImagePrePullJobReportActionStatus(t *testing.T) {
	var (
		jobName  = "test-job"
		nodeName = "node1"
		status   = []operationsv1alpha2.ImageStatus{
			{
				Image:  "image1",
				Status: metav1.ConditionTrue,
			},
			{
				Image:  "image2",
				Status: metav1.ConditionFalse,
				Reason: "test error",
			},
		}
	)

	cases := []struct {
		name       string
		action     string
		resp       ActionResponse
		wantExtend bool
	}{
		{
			name:   "check action does not report image status",
			action: string(operationsv1alpha2.ImagePrePullJobActionCheck),
			resp: &imagePrePullJobActionResponse{
				imageStatus: status,
			},
		},
		{
			name:   "pull action reports image status",
			action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			resp: &imagePrePullJobActionResponse{
				imageStatus: status,
			},
			wantExtend: true,
		},
		{
			name:   "failed pull action reports image status",
			action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			resp: &imagePrePullJobActionResponse{
				imageStatus: status,
				baseActionResponse: baseActionResponse{
					err: errors.New("test error"),
				},
			},
			wantExtend: true,
		},
	}

	h := imagePrePullJobActionHandler{
		logger: klog.Background(),
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			called := false
			patches.ApplyFunc(message.ReportNodeTaskStatus, func(res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
				called = true
				assert.Equal(t, operationsv1alpha2.SchemeGroupVersion.String(), res.APIVersion)
				assert.Equal(t, operationsv1alpha2.ResourceImagePrePullJob, res.ResourceType)
				assert.Equal(t, jobName, res.JobName)
				assert.Equal(t, nodeName, res.NodeName)
				assert.Equal(t, c.action, msgbody.Action)
				assert.NotEmpty(t, msgbody.FinishTime)
				if c.wantExtend {
					require.NotEmpty(t, msgbody.Extend)
					gotStatus, err := taskmsg.ParseImagePrePullJobExtend(msgbody.Extend)
					require.NoError(t, err)
					assert.Equal(t, status, gotStatus)
				} else {
					assert.Empty(t, msgbody.Extend)
				}
				if c.resp.Error() == nil {
					assert.True(t, msgbody.Succ)
				} else {
					assert.False(t, msgbody.Succ)
					assert.Equal(t, c.resp.Error().Error(), msgbody.Reason)
				}
			})

			h.reportActionStatus(jobName, nodeName, c.action, c.resp)
			assert.True(t, called)
		})
	}
}
