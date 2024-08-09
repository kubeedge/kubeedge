/*
Copyright 2023 The KubeEdge Authors.

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

package taskexecutor

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	edgeutil "github.com/kubeedge/kubeedge/edge/pkg/common/util"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

const (
	TaskPrePull = "prepull"
)

type PrePull struct {
	*BaseExecutor
}

func (p *PrePull) Name() string {
	return p.name
}

func NewPrePullExecutor() Executor {
	methods := map[string]func(types.NodeTaskRequest) fsm.Event{
		string(api.TaskChecking): preCheck,
		string(api.TaskInit):     emptyInit,
		"":                       emptyInit,
		string(api.PullingState): pullImages,
	}
	return &PrePull{
		BaseExecutor: NewBaseExecutor(TaskPrePull, methods),
	}
}

func pullImages(taskReq types.NodeTaskRequest) fsm.Event {
	event := fsm.Event{
		Type:   "Pull",
		Action: api.ActionSuccess,
	}

	// get edgecore config
	edgeCoreConfig := options.GetEdgeCoreConfig()

	// parse message request
	prePullReq, err := getImagePrePullJobRequest(taskReq)
	if err != nil {
		event.Msg = err.Error()
		event.Action = api.ActionFailure
		return event
	}

	// pull images
	container, err := util.NewContainerRuntime(edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint, edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		event.Msg = err.Error()
		event.Action = api.ActionFailure
		return event
	}

	go func() {
		errorStr, imageStatus := prePullImages(*prePullReq, container)
		if errorStr != "" {
			event.Action = api.ActionFailure
			event.Msg = errorStr
		}

		data, err := json.Marshal(imageStatus)
		if err != nil {
			klog.Warningf("marshal imageStatus failed: %v", err)
		}
		resp := commontypes.NodeTaskResponse{
			NodeName:        edgeCoreConfig.Modules.Edged.HostnameOverride,
			Event:           event.Type,
			Action:          event.Action,
			Reason:          event.Msg,
			ExternalMessage: string(data),
		}
		edgeutil.ReportTaskResult(taskReq.Type, taskReq.TaskID, resp)
	}()
	return fsm.Event{}
}

func getImagePrePullJobRequest(taskReq commontypes.NodeTaskRequest) (*commontypes.ImagePrePullJobRequest, error) {
	var prePullReq commontypes.ImagePrePullJobRequest
	data, err := json.Marshal(taskReq.Item)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &prePullReq)
	if err != nil {
		return nil, err
	}
	return &prePullReq, err
}

func prePullImages(prePullReq commontypes.ImagePrePullJobRequest, container util.ContainerRuntime) (string, []v1alpha1.ImageStatus) {
	errorStr := ""
	authConfig, err := makeAuthConfig(prePullReq.Secret)
	if err != nil {
		return errorStr, []v1alpha1.ImageStatus{}
	}

	var imageStatus []v1alpha1.ImageStatus
	for _, image := range prePullReq.Images {
		prePullStatus := v1alpha1.ImageStatus{
			Image: image,
		}
		for i := 0; i <= int(prePullReq.RetryTimes); i++ {
			err = container.PullImage(image, authConfig, nil)
			if err == nil {
				break
			}
		}
		if err != nil {
			klog.Errorf("pull image %s failed, err: %v", image, err)
			errorStr = fmt.Sprintf("pull image failed, err: %v", err)
			prePullStatus.State = api.TaskFailed
			prePullStatus.Reason = err.Error()
		} else {
			klog.Infof("pull image %s successfully!", image)
			prePullStatus.State = api.TaskSuccessful
		}
		imageStatus = append(imageStatus, prePullStatus)
	}

	return errorStr, imageStatus
}

func makeAuthConfig(pullsecret string) (*runtimeapi.AuthConfig, error) {
	if pullsecret == "" {
		return nil, nil
	}

	secretSli := strings.Split(pullsecret, constants.ResourceSep)
	if len(secretSli) != 2 {
		return nil, fmt.Errorf("pull secret format is not correct")
	}

	client := metaclient.New()
	secret, err := client.Secrets(secretSli[0]).Get(secretSli[1])
	if err != nil {
		return nil, fmt.Errorf("get secret %s failed, %v", secretSli[1], err)
	}

	var auth runtimeapi.AuthConfig
	err = json.Unmarshal(secret.Data[v1.DockerConfigJsonKey], &auth)
	if err != nil {
		return nil, fmt.Errorf("unmarshal secret %s to auth file failed, %v", secretSli[1], err)
	}

	return &auth, nil
}
