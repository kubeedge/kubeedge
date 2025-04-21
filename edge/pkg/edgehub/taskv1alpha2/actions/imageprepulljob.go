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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	klog "k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/pkg/image"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func newImagePrePullJobRunner() *ActionRunner {
	logger := klog.Background().WithName("image-prepull-job-runner")
	handler := imagePrePullJobActionHandler{
		logger: logger,
	}
	runner := &ActionRunner{
		Flow:               actionflow.FlowImagePrePullJob,
		ReportActionStatus: handler.reportActionStatus,
		GetSpecSerializer:  handler.getSpecSerializer,
		Logger:             logger,
	}
	runner.addAction(string(operationsv1alpha2.ImagePrePullJobActionCheck), handler.checkItems)
	runner.addAction(string(operationsv1alpha2.ImagePrePullJobActionPull), handler.pullImages)
	return runner
}

const (
	pullImageFailureMessage = "there were some failures when pulling images"
)

type imagePrePullJobActionResponse struct {
	imageStatus []operationsv1alpha2.ImageStatus
	baseActionResponse
}

// Check that imagePrePullJobActionResponse implements ActionResponse interface.
var _ ActionResponse = (*imagePrePullJobActionResponse)(nil)

// imagePrePullJobActionHandler defines action-related functions
type imagePrePullJobActionHandler struct {
	logger logr.Logger
}

func (imagePrePullJobActionHandler) checkItems(_ctx context.Context, specser SpecSerializer) ActionResponse {
	resp := new(imagePrePullJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.ImagePrePullJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to ImagePrePullJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	if err := PreCheck(spec.ImagePrePullTemplate.CheckItems); err != nil {
		resp.err = err
		return resp
	}
	resp.doNext = true
	return resp
}

func (h *imagePrePullJobActionHandler) pullImages(ctx context.Context, specser SpecSerializer) ActionResponse {
	const retryDelay = 500 * time.Millisecond
	resp := new(imagePrePullJobActionResponse)
	spec, ok := specser.GetSpec().(*operationsv1alpha2.ImagePrePullJobSpec)
	if !ok {
		resp.err = fmt.Errorf("failed to conv spec to ImagePrePullJobSpec, actual type %T", specser.GetSpec())
		return resp
	}
	var imageStatus []operationsv1alpha2.ImageStatus
	err := retry.Do(
		func() error {
			var err error
			imageStatus, err = h.tryPullImage(ctx, spec)
			return err
		},
		retry.Delay(retryDelay),
		// Indicates the total number of times the function is called
		retry.Attempts(uint(spec.ImagePrePullTemplate.RetryTimes)+1),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true))
	if err != nil {
		resp.err = err
	}
	resp.imageStatus = imageStatus
	resp.doNext = true
	return resp
}

func (imagePrePullJobActionHandler) tryPullImage(
	ctx context.Context,
	spec *operationsv1alpha2.ImagePrePullJobSpec,
) ([]operationsv1alpha2.ImageStatus, error) {
	edgecoreCfg := options.GetEdgeCoreConfig()
	imgrt, err := image.NewImageRuntime(
		edgecoreCfg.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint,
		edgecoreCfg.Modules.Edged.TailoredKubeletConfig.RuntimeRequestTimeout.Duration)
	if err != nil {
		return nil, err
	}
	var authcfg runtimeapi.AuthConfig
	if imgsec := spec.ImagePrePullTemplate.ImageSecret; imgsec != "" {
		named := strings.Split(imgsec, constants.ResourceSep)
		if len(named) != 2 {
			return nil, fmt.Errorf("pull secret format is not correct")
		}
		client := metaclient.New()
		secret, err := client.Secrets(named[0]).Get(named[1])
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %s/%s, err: %v", named[0], named[1], err)
		}
		if err := json.Unmarshal(secret.Data[corev1.DockerConfigJsonKey], &authcfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal secret %s/%s to auth config, err: %v",
				named[0], named[1], err)
		}
	}
	imageStatus := make([]operationsv1alpha2.ImageStatus, 0, len(spec.ImagePrePullTemplate.Images))
	var hasError bool
	for _, image := range spec.ImagePrePullTemplate.Images {
		st := operationsv1alpha2.ImageStatus{Image: image}
		if err := imgrt.PullImage(ctx, image, &authcfg, nil); err != nil {
			hasError = true
			st.Status = metav1.ConditionFalse
			st.Reason = err.Error()
		} else {
			st.Status = metav1.ConditionTrue
		}
		imageStatus = append(imageStatus, st)
	}
	if hasError {
		return imageStatus, errors.New(pullImageFailureMessage)
	}
	return imageStatus, nil
}

func (imagePrePullJobActionHandler) getSpecSerializer(specData []byte) (SpecSerializer, error) {
	return NewSpecSerializer(specData, func(d []byte) (any, error) {
		var spec operationsv1alpha2.ImagePrePullJobSpec
		if err := json.Unmarshal(d, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	})
}

func (h imagePrePullJobActionHandler) reportActionStatus(jobname, nodename, action string, resp ActionResponse) {
	res := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceImagePrePullJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	var extend string
	if resp, ok := resp.(*imagePrePullJobActionResponse); ok &&
		action == string(operationsv1alpha2.ImagePrePullJobActionPull) {
		bff, err := json.Marshal(resp.imageStatus)
		if err != nil {
			h.logger.Error(err, "failed to marshal image status")
		} else {
			extend = string(bff)
		}
	}
	body := taskmsg.UpstreamMessage{
		Action: action,
		Extend: extend,
	}
	if err := resp.Error(); err != nil {
		body.Succ = false
		body.Reason = err.Error()
	} else {
		body.Succ = true
	}
	message.ReportNodeTaskStatus(res, body)
}
