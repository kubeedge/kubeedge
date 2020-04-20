package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"beehive/adoptions/common/api"
	"beehive/pkg/common/log"
	"beehive/pkg/core"
	"beehive/pkg/core/context"
	"beehive/pkg/core/model"
	"edge-core/pkg/metamanager/dao"
)

const (
	typePodStatus = "podstatus"

	resourcePod        = "edge/pod/%s"
	resourceAppsGet    = "$hw/events/apps/result"
	resourceAppRestart = "$hw/events/apps/%s/restart/result"

	operationPublish = "publish"
	operationRestart = "restart"
	operationGet     = "get"

	resultCompleted = "ok"
	resultFailed    = "failed"

	moduleEventBus = "eventbus"
)

type AppsRestartReply struct {
	Result string `json:"result"`
}

type AppsQueryReply struct {
	Apps []PodInfo `json:"apps"`
}

type PodInfo struct {
	ID     string `json:"ID"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func HandleMessageForAppsControl(topic, pyload string, context *context.Context) {
	log.LOGGER.Infof("receive message for apps control ,topic [%s]", topic)
	segments := strings.Split(topic, "/")
	if len(segments) < 4 {
		log.LOGGER.Warnf("not support apps topic ")
		return
	}
	if len(segments) == 4 && segments[3] == operationGet {
		handleMessageForAppsQuery(context)
	} else if len(segments) == 5 && segments[4] == operationRestart {
		handleMessageForAppsRestart(segments[3], context)
	}
	return
}

func handleMessageForAppsQuery(context *context.Context) {
	podInfoList := transformPodStatusListToPodInfoList(getPodStatusListFromDB())
	out := AppsQueryReply{Apps: podInfoList}
	go context.Send(moduleEventBus, buildReplyMessageForAppsQuery(out))
}

func handleMessageForAppsRestart(podName string, context *context.Context) {
	podInfoList := transformPodStatusListToPodInfoList(getPodStatusListFromDB())
	for _, pod := range podInfoList {
		if pod.Name != podName {
			continue
		}

		go context.Send(core.EdgedModuleName, buildMessageForAppRestart(podName))
		go context.Send(moduleEventBus, buildReplyMessageForAppRestart(podName, AppsRestartReply{Result: resultCompleted}))
		return
	}
	go context.Send(moduleEventBus, buildReplyMessageForAppRestart(podName, AppsRestartReply{Result: resultFailed}))
}

func getPodStatusListFromDB() []api.PodStatusRequest {
	var podstatusList []api.PodStatusRequest
	podStatusDBList, err := dao.QueryMeta("type", typePodStatus)
	if err != nil {
		log.LOGGER.Warnf("get podstatus from db failed. error:%s", err.Error())
		return podstatusList
	}

	for _, value := range *podStatusDBList {
		var podstatus api.PodStatusRequest
		err = json.Unmarshal([]byte(value), &podstatus)
		if err != nil {
			return podstatusList
		}
		podstatusList = append(podstatusList, podstatus)
	}
	return podstatusList
}

func transformPodStatusListToPodInfoList(podstatusList []api.PodStatusRequest) []PodInfo {
	var podInfoList []PodInfo
	for _, value := range podstatusList {
		podInfoList = append(podInfoList, transformPodStatusToPodInfo(value))
	}
	return podInfoList
}

func transformPodStatusToPodInfo(podstatus api.PodStatusRequest) PodInfo {
	return PodInfo{
		ID:     string(podstatus.UID),
		Name:   podstatus.Name,
		Status: string(podstatus.Status.Phase),
	}
}

func buildReplyMessageForAppsQuery(reply AppsQueryReply) model.Message {
	msg := model.NewMessage("").BuildRouter("", core.BusGroup, resourceAppsGet, operationPublish)

	content, err := json.Marshal(reply)
	if err != nil {
		log.LOGGER.Warnf("marshal apps get result failed. error:%s", err.Error())
	}
	msg.FillBody(content)
	return *msg
}

func buildMessageForAppRestart(podName string) model.Message {
	return *model.NewMessage("").BuildRouter("", core.EdgedGroup, fmt.Sprintf(resourcePod, podName), operationRestart).FillBody("")
}

func buildReplyMessageForAppRestart(podName string, result AppsRestartReply) model.Message {
	msg := model.NewMessage("").BuildRouter("", core.BusGroup, fmt.Sprintf(resourceAppRestart, podName), operationPublish)

	content, err := json.Marshal(result)
	if err != nil {
		log.LOGGER.Warnf("marshal apps get result failed. error:%s", err.Error())
	}
	msg.FillBody(content)
	return *msg
}

// HandleEdgeGatewayRequest get edgegateway msg
func HandleEdgeGatewayRequest(gatewayType string) *[]string {
	gatewayDBList, err := dao.QueryMeta("type", gatewayType)
	if err != nil {
		log.LOGGER.Warnf("get gatewayList from db failed. error:%s", err.Error())
		return nil
	}

	return gatewayDBList
}
