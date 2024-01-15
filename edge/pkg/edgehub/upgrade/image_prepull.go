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

package upgrade

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/msghandler"
	metaclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
)

func init() {
	handler := &prepullHandler{}
	msghandler.RegisterHandler(handler)
}

type prepullHandler struct{}

func (uh *prepullHandler) Filter(message *model.Message) bool {
	name := message.GetGroup()
	return name == cloudmodules.ImagePrePullControllerModuleGroup
}

func (uh *prepullHandler) Process(message *model.Message, clientHub clients.Adapter) error {
	// get edgecore config
	edgecoreconfig := options.GetEdgeCoreConfig()
	if edgecoreconfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint == "" {
		edgecoreconfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint = edgecoreconfig.Modules.Edged.RemoteRuntimeEndpoint
	}
	nodeName := edgecoreconfig.Modules.Edged.HostnameOverride

	var errmsg, jobName string
	defer func() {
		if errmsg != "" {
			sendResponseMessage(nodeName, jobName, errmsg, nil, v1alpha1.PrePullFailed)
		}
	}()

	jobName, err := parseJobName(message.GetResource())
	if err != nil {
		errmsg = fmt.Sprintf("failed to parse prepull resource, err: %v", err)
		return fmt.Errorf(errmsg)
	}

	// parse message request
	var prePullReq commontypes.ImagePrePullJobRequest
	data, err := message.GetContentData()
	if err != nil {
		errmsg = fmt.Sprintf("failed to get message content, err: %v", err)
		return fmt.Errorf(errmsg)
	}
	err = json.Unmarshal(data, &prePullReq)
	if err != nil {
		errmsg = fmt.Sprintf("failed to unmarshal message content, err: %v", err)
		return fmt.Errorf(errmsg)
	}
	if nodeName != prePullReq.NodeName {
		errmsg = fmt.Sprintf("request node name %s is not match with the edge node %s", prePullReq.NodeName, nodeName)
		return fmt.Errorf(errmsg)
	}

	// todo check items such as disk according to req.checkitem config

	// pull images
	container, err := util.NewContainerRuntime(edgecoreconfig.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint, edgecoreconfig.Modules.Edged.TailoredKubeletConfig.CgroupDriver)
	if err != nil {
		errmsg = fmt.Sprintf("failed to new container runtime: %v", err)
		return fmt.Errorf(errmsg)
	}

	go func() {
		pullMsg, resp, state := prePullImages(prePullReq, container)
		sendResponseMessage(nodeName, jobName, pullMsg, resp, state)
	}()
	return nil
}

func sendResponseMessage(nodeName, jobName, pullMsg string, imageStatus []v1alpha1.ImageStatus, state v1alpha1.PrePullState) {
	resp := commontypes.ImagePrePullJobResponse{
		NodeName:    nodeName,
		State:       state,
		Reason:      pullMsg,
		ImageStatus: imageStatus,
	}
	msg := model.NewMessage("").SetRoute(modules.EdgeHubModuleName, modules.HubGroup).
		SetResourceOperation(fmt.Sprintf("node/%s/prepull/%s", nodeName, jobName), "prepull").FillBody(resp)
	beehiveContext.Send(modules.EdgeHubModuleName, *msg)
}

func prePullImages(prePullReq commontypes.ImagePrePullJobRequest, container util.ContainerRuntime) (string, []v1alpha1.ImageStatus, v1alpha1.PrePullState) {
	var errmsg string
	authConfig, err := makeAuthConfig(prePullReq.Secret)
	if err != nil {
		errmsg = fmt.Sprintf("failed to get prepull secret, err: %v", err)
		return errmsg, nil, v1alpha1.PrePullFailed
	}

	state := v1alpha1.PrePullSuccessful
	var imageStatus []v1alpha1.ImageStatus
	for _, image := range prePullReq.Images {
		prePullStatus := v1alpha1.ImageStatus{
			Image: image,
		}
		for i := 0; i < int(prePullReq.RetryTimes); i++ {
			err = container.PullImage(image, authConfig, nil)
			if err == nil {
				break
			}
		}
		if err != nil {
			klog.Errorf("pull image %s failed, err: %v", image, err)
			errmsg = "prepull images failed"
			state = v1alpha1.PrePullFailed
			prePullStatus.State = v1alpha1.PrePullFailed
			prePullStatus.Reason = err.Error()
		} else {
			klog.Infof("pull image %s successfully!", image)
			prePullStatus.State = v1alpha1.PrePullSuccessful
		}
		imageStatus = append(imageStatus, prePullStatus)
	}

	return errmsg, imageStatus, state
}

func parseJobName(resource string) (string, error) {
	var jobName string
	sli := strings.Split(resource, constants.ResourceSep)
	if len(sli) != 2 {
		return jobName, fmt.Errorf("the resource %s is not the standard type", resource)
	}
	return sli[1], nil
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
