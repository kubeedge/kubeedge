package metamanager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common/util"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	metaManagerConfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
)

//Constants to check metamanager processes
const (
	OK = "OK"

	GroupResource     = "resource"
	OperationMetaSync = "meta-internal-sync"

	OperationFunctionAction = "action"

	OperationFunctionActionResult = "action_result"

	EdgeFunctionModel   = "edgefunction"
	CloudFunctionModel  = "funcmgr"
	CloudControlerModel = "edgecontroller"
)

func feedbackError(err error, info string, request model.Message) {
	errInfo := "Something wrong"
	if err != nil {
		errInfo = fmt.Sprintf(info+": %v", err)
	}
	errResponse := model.NewErrorMessage(&request, errInfo).SetRoute(MetaManagerModuleName, request.GetGroup())
	if request.GetSource() == modules.EdgedModuleName {
		sendToEdged(errResponse, request.IsSync())
	} else {
		sendToCloud(errResponse)
	}
}

func sendToEdged(message *model.Message, sync bool) {
	if sync {
		beehiveContext.SendResp(*message)
	} else {
		beehiveContext.Send(modules.EdgedModuleName, *message)
	}
}

func sendToCloud(message *model.Message) {
	beehiveContext.SendToGroup(string(metaManagerConfig.Config.ContextSendGroup), *message)
}

// Resource format: <namespace>/<restype>[/resid]
// return <reskey, restype, resid>
func parseResource(resource string) (string, string, string) {
	tokens := strings.Split(resource, constants.ResourceSep)
	resType := ""
	resID := ""
	switch len(tokens) {
	case 2:
		resType = tokens[len(tokens)-1]
	case 3:
		resType = tokens[len(tokens)-2]
		resID = tokens[len(tokens)-1]
	default:
	}
	return resource, resType, resID
}

// is resource type require remote query
func requireRemoteQuery(resType string) bool {
	return resType == model.ResourceTypeConfigmap ||
		resType == model.ResourceTypeSecret ||
		resType == constants.ResourceTypePersistentVolume ||
		resType == constants.ResourceTypePersistentVolumeClaim ||
		resType == constants.ResourceTypeVolumeAttachment ||
		resType == model.ResourceTypeNode
}

func isConnected() bool {
	return metaManagerConfig.Connected
}

func msgDebugInfo(message *model.Message) string {
	return fmt.Sprintf("msgID[%s] resource[%s]", message.GetID(), message.GetResource())
}

func resourceUnchanged(resType string, resKey string, content []byte) bool {
	if resType == model.ResourceTypePodStatus {
		dbRecord, err := dao.QueryMeta("key", resKey)
		if err == nil && len(*dbRecord) > 0 && string(content) == (*dbRecord)[0] {
			return true
		}
	}

	return false
}

func (m *metaManager) processInsert(message model.Message) {
	var err error
	var content []byte
	switch message.GetContent().(type) {
	case []uint8:
		content = message.GetContent().([]byte)
	default:
		content, err = json.Marshal(message.GetContent())
		if err != nil {
			klog.Errorf("marshal update message content failed, %s", msgDebugInfo(&message))
			feedbackError(err, "Error to marshal message content", message)
			return
		}
	}
	imitator.DefaultV2Client.Inject(message)
	resKey, resType, _ := parseResource(message.GetResource())

	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.SaveMeta(meta)
	if err != nil {
		klog.Errorf("save meta failed, %s: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message)
		return
	}

	// Notify edged
	sendToEdged(&message, false)

	resp := message.NewRespByMessage(&message, OK)
	sendToCloud(resp)
}

func (m *metaManager) processUpdate(message model.Message) {
	var err error
	var content []byte
	switch message.GetContent().(type) {
	case []uint8:
		content = message.GetContent().([]byte)
	default:
		content, err = json.Marshal(message.GetContent())
		if err != nil {
			klog.Errorf("marshal update message content failed, %s", msgDebugInfo(&message))
			feedbackError(err, "Error to marshal message content", message)
			return
		}
	}
	imitator.DefaultV2Client.Inject(message)

	resKey, resType, _ := parseResource(message.GetResource())

	if resourceUnchanged(resType, resKey, content) {
		resp := message.NewRespByMessage(&message, OK)
		sendToEdged(resp, message.IsSync())
		klog.Infof("resource[%s] unchanged, no notice", resKey)
		return
	}

	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("update meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to update meta to DB", message)
		return
	}

	msgSource := message.GetSource()
	switch msgSource {
	//case core.EdgedModuleName:
	case modules.EdgedModuleName:
		sendToCloud(&message)
		resp := message.NewRespByMessage(&message, OK)
		sendToEdged(resp, message.IsSync())
	case cloudmodules.EdgeControllerModuleName, cloudmodules.DynamicControllerModuleName:
		sendToEdged(&message, message.IsSync())
		resp := message.NewRespByMessage(&message, OK)
		sendToCloud(resp)
	case CloudFunctionModel:
		beehiveContext.Send(EdgeFunctionModel, message)
	case EdgeFunctionModel:
		sendToCloud(&message)
	default:
		klog.Errorf("unsupport message source, %s", msgSource)
	}
}

func (m *metaManager) processResponse(message model.Message) {
	var err error
	var content []byte
	switch message.GetContent().(type) {
	case []uint8:
		content = message.GetContent().([]byte)
	default:
		content, err = json.Marshal(message.GetContent())
		if err != nil {
			klog.Errorf("marshal response message content failed, %s", msgDebugInfo(&message))
			feedbackError(err, "Error to marshal message content", message)
			return
		}
	}

	resKey, resType, _ := parseResource(message.GetResource())
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("update meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to update meta to DB", message)
		return
	}

	// Notify edged if the data is coming from cloud
	if message.GetSource() == CloudControlerModel {
		sendToEdged(&message, message.IsSync())
	} else {
		// Send to cloud if the update request is coming from edged
		sendToCloud(&message)
	}
}

func (m *metaManager) processDelete(message model.Message) {
	imitator.DefaultV2Client.Inject(message)
	err := dao.DeleteMetaByKey(message.GetResource())
	if err != nil {
		klog.Errorf("delete meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to delete meta to DB", message)
		return
	}

	_, resType, _ := parseResource(message.GetResource())

	if resType == model.ResourceTypePod && message.GetSource() == modules.EdgedModuleName {
		sendToCloud(&message)
		return
	}

	// Notify edged
	sendToEdged(&message, false)
	resp := message.NewRespByMessage(&message, OK)
	sendToCloud(resp)
}

func (m *metaManager) processQuery(message model.Message) {
	resKey, resType, resID := parseResource(message.GetResource())
	var metas *[]string
	var err error
	if requireRemoteQuery(resType) && isConnected() {
		metas, err = dao.QueryMeta("key", resKey)
		if err != nil || len(*metas) == 0 || resType == model.ResourceTypeNode || resType == constants.ResourceTypeVolumeAttachment {
			m.processRemoteQuery(message)
		} else {
			resp := message.NewRespByMessage(&message, *metas)
			resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
			sendToEdged(resp, message.IsSync())
		}
		return
	}

	if resID == "" {
		// Get specific type resources
		metas, err = dao.QueryMeta("type", resType)
	} else {
		metas, err = dao.QueryMeta("key", resKey)
	}
	if err != nil {
		klog.Errorf("query meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to query meta in DB", message)
	} else {
		resp := message.NewRespByMessage(&message, *metas)
		resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
		sendToEdged(resp, message.IsSync())
	}
}

func (m *metaManager) processRemoteQuery(message model.Message) {
	go func() {
		// TODO: retry
		originalID := message.GetID()
		message.UpdateID()
		resp, err := beehiveContext.SendSync(
			string(metaManagerConfig.Config.ContextSendModule),
			message,
			time.Duration(metaManagerConfig.Config.RemoteQueryTimeout)*time.Second)
		klog.Infof("########## process get: req[%+v], resp[%+v], err[%+v]", message, resp, err)
		if err != nil {
			klog.Errorf("remote query failed: %v", err)
			feedbackError(err, "Error to query meta in DB", message)
			return
		}

		var content []byte
		switch resp.GetContent().(type) {
		case []uint8:
			content = resp.GetContent().([]byte)
		default:
			content, err = json.Marshal(resp.GetContent())
			if err != nil {
				klog.Errorf("marshal remote query response content failed, %s", msgDebugInfo(&resp))
				feedbackError(err, "Error to marshal message content", message)
				return
			}
		}

		resKey, resType, _ := parseResource(message.GetResource())
		meta := &dao.Meta{
			Key:   resKey,
			Type:  resType,
			Value: string(content)}
		err = dao.InsertOrUpdate(meta)
		if err != nil {
			klog.Errorf("update meta failed, %s", msgDebugInfo(&resp))
		}
		resp.BuildHeader(resp.GetID(), originalID, resp.GetTimestamp())

		sendToEdged(&resp, message.IsSync())

		respToCloud := message.NewRespByMessage(&resp, OK)
		sendToCloud(respToCloud)
	}()
}

func (m *metaManager) processNodeConnection(message model.Message) {
	content, _ := message.GetContent().(string)
	klog.Infof("node connection event occur: %s", content)
	if content == connect.CloudConnected {
		metaManagerConfig.Connected = true
	} else if content == connect.CloudDisconnected {
		metaManagerConfig.Connected = false
	}
}

func (m *metaManager) processSync() {
	m.syncPodStatus()
}

func (m *metaManager) syncPodStatus() {
	klog.Infof("start to sync pod status in edge-store to cloud")
	podStatusRecords, err := dao.QueryAllMeta("type", model.ResourceTypePodStatus)
	if err != nil {
		klog.Errorf("list pod status failed: %v", err)
		return
	}
	if len(*podStatusRecords) <= 0 {
		klog.Infof("list pod status, no record, skip sync")
		return
	}
	contents := make(map[string][]interface{})
	for _, v := range *podStatusRecords {
		namespaceParsed, _, _, _ := util.ParseResourceEdge(v.Key, model.QueryOperation)
		podKey := strings.Replace(v.Key, constants.ResourceSep+model.ResourceTypePodStatus+constants.ResourceSep, constants.ResourceSep+model.ResourceTypePod+constants.ResourceSep, 1)
		podRecord, err := dao.QueryMeta("key", podKey)
		if err != nil {
			klog.Errorf("query pod[%s] failed: %v", podKey, err)
			return
		}

		if len(*podRecord) <= 0 {
			// pod already deleted, clear the corresponding podstatus record
			err = dao.DeleteMetaByKey(v.Key)
			klog.Infof("pod[%s] already deleted, clear podstatus record, result:%v", podKey, err)
			continue
		}

		var podStatus interface{}
		err = json.Unmarshal([]byte(v.Value), &podStatus)
		if err != nil {
			klog.Errorf("unmarshal podstatus[%s] failed, content[%s]: %v", v.Key, v.Value, err)
			continue
		}
		contents[namespaceParsed] = append(contents[namespaceParsed], podStatus)
	}
	for namespace, content := range contents {
		msg := model.NewMessage("").BuildRouter(MetaManagerModuleName, GroupResource, namespace+constants.ResourceSep+model.ResourceTypePodStatus, model.UpdateOperation).FillBody(content)
		sendToCloud(msg)
		klog.V(3).Infof("sync pod status successfully for namespaces %s, %s", namespace, msgDebugInfo(msg))
	}
}

func (m *metaManager) processFunctionAction(message model.Message) {
	var err error
	var content []byte
	switch message.GetContent().(type) {
	case []uint8:
		content = message.GetContent().([]byte)
	default:
		content, err = json.Marshal(message.GetContent())
		if err != nil {
			klog.Errorf("marshal save message content failed, %s: %v", msgDebugInfo(&message), err)
			feedbackError(err, "Error to marshal message content", message)
			return
		}
	}

	resKey, resType, _ := parseResource(message.GetResource())
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.SaveMeta(meta)
	if err != nil {
		klog.Errorf("save meta failed, %s: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message)
		return
	}

	beehiveContext.Send(EdgeFunctionModel, message)
}

func (m *metaManager) processFunctionActionResult(message model.Message) {
	var err error
	var content []byte
	switch message.GetContent().(type) {
	case []uint8:
		content = message.GetContent().([]byte)
	default:
		content, err = json.Marshal(message.GetContent())
		if err != nil {
			klog.Errorf("marshal save message content failed, %s: %v", msgDebugInfo(&message), err)
			feedbackError(err, "Error to marshal message content", message)
			return
		}
	}

	resKey, resType, _ := parseResource(message.GetResource())
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.SaveMeta(meta)
	if err != nil {
		klog.Errorf("save meta failed, %s: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message)
		return
	}

	sendToCloud(&message)
}

func (m *metaManager) processVolume(message model.Message) {
	klog.Info("process volume started")
	back, err := beehiveContext.SendSync(modules.EdgedModuleName, message, constants.CSISyncMsgRespTimeout)
	klog.Infof("process volume get: req[%+v], back[%+v], err[%+v]", message, back, err)
	if err != nil {
		klog.Errorf("process volume send to edged failed: %v", err)
	}

	resp := message.NewRespByMessage(&message, back.GetContent())
	sendToCloud(resp)
	klog.Infof("process volume send to cloud resp[%+v]", resp)
}

func (m *metaManager) process(message model.Message) {
	operation := message.GetOperation()
	switch operation {
	case model.InsertOperation:
		m.processInsert(message)
	case model.UpdateOperation:
		m.processUpdate(message)
	case model.DeleteOperation:
		m.processDelete(message)
	case model.QueryOperation:
		m.processQuery(message)
	case model.ResponseOperation:
		m.processResponse(message)
	case messagepkg.OperationNodeConnection:
		m.processNodeConnection(message)
	case OperationMetaSync:
		m.processSync()
	case OperationFunctionAction:
		m.processFunctionAction(message)
	case OperationFunctionActionResult:
		m.processFunctionActionResult(message)
	case constants.CSIOperationTypeCreateVolume,
		constants.CSIOperationTypeDeleteVolume,
		constants.CSIOperationTypeControllerPublishVolume,
		constants.CSIOperationTypeControllerUnpublishVolume:
		m.processVolume(message)
	}
}

func (m *metaManager) runMetaManager() {
	go func() {
		for {
			select {
			case <-beehiveContext.Done():
				klog.Warning("MetaManager main loop stop")
				return
			default:
			}
			msg, err := beehiveContext.Receive(m.Name())
			if err != nil {
				klog.Errorf("get a message %+v: %v", msg, err)
				continue
			}
			klog.V(2).Infof("get a message %+v", msg)
			m.process(msg)
		}
	}()
}
