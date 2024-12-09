package metamanager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
	metaManagerConfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
)

// Constants to check metamanager processes
const (
	OK                   = "OK"
	GroupResource        = "resource"
	CloudControllerModel = "edgecontroller"
	errNotConnected      = "not connected"
)

func feedbackError(err error, request model.Message) {
	errResponse := model.NewErrorMessage(&request, err.Error()).SetRoute(modules.MetaManagerModuleName, request.GetGroup())
	if request.GetSource() == modules.EdgedModuleName {
		sendToEdged(errResponse, request.IsSync())
	} else {
		sendToCloud(errResponse)
	}
}

func feedbackResponse(message *model.Message, parentID string, resp *model.Message) {
	resp.BuildHeader(resp.GetID(), parentID, resp.GetTimestamp())
	sendToEdged(resp, message.IsSync())
	respToCloud := message.NewRespByMessage(resp, OK)
	sendToCloud(respToCloud)
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

func sendToTwin(message *model.Message) {
	beehiveContext.Send(modules.DeviceTwinModuleName, *message)
}

func sentToMetamanager(message *model.Message, sync bool) {
	if sync {
		beehiveContext.SendResp(*message)
	} else {
		beehiveContext.Send(modules.MetaManagerModuleName, *message)
	}
}

// Resource format: <namespace>/<restype>[/resid]
// return <reskey, restype, resid>
func parseResource(message *model.Message) (string, string, string) {
	resource := message.GetResource()
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
	if resType != model.ResourceTypeServiceAccountToken {
		return resource, resType, resID
	}
	var tokenReq authenticationv1.TokenRequest
	content, err := message.GetContentData()
	if err != nil {
		klog.Errorf("failed to get token request from message %s, error %s", message.GetID(), err)
		return "", "", ""
	}
	if err = json.Unmarshal(content, &tokenReq); err != nil {
		klog.Errorf("failed to unmarshal token request from message %s, error %s", message.GetID(), err)
		return "", "", ""
	}

	trTokens := strings.Split(resource, constants.ResourceSep)
	if len(trTokens) != 3 {
		klog.Errorf("failed to get resource %s name and namespace", resource)
		return "", "", ""
	}
	return client.KeyFunc(trTokens[2], trTokens[0], &tokenReq), resType, ""
}

// is resource type require remote query
func requireRemoteQuery(resType string) bool {
	return resType == model.ResourceTypeConfigmap ||
		resType == model.ResourceTypeSecret ||
		resType == constants.ResourceTypePersistentVolume ||
		resType == constants.ResourceTypePersistentVolumeClaim ||
		resType == constants.ResourceTypeVolumeAttachment ||
		resType == model.ResourceTypeNode ||
		resType == model.ResourceTypeServiceAccountToken ||
		resType == model.ResourceTypeLease ||
		resType == model.ResourceTypeCSR ||
		resType == model.ResourceTypeK8sCA
}

func msgDebugInfo(message *model.Message) string {
	return fmt.Sprintf("msgID[%s] resource[%s]", message.GetID(), message.GetResource())
}

func (m *metaManager) handleMessage(message *model.Message) error {
	resKey, resType, _ := parseResource(message)
	switch message.GetOperation() {
	case model.InsertOperation, model.UpdateOperation, model.PatchOperation, model.ResponseOperation:
		content, err := message.GetContentData()
		if err != nil {
			klog.Errorf("get message content data failed, message: %s, error: %s", msgDebugInfo(message), err)
			return fmt.Errorf("get message content data failed, error: %s", err)
		}
		meta := &dao.Meta{
			Key:   resKey,
			Type:  resType,
			Value: string(content)}
		err = dao.InsertOrUpdate(meta)
		if err != nil {
			klog.Errorf("insert or update meta failed, message: %s, error: %v", msgDebugInfo(message), err)
			return fmt.Errorf("insert or update meta failed, %s", err)
		}
	case model.DeleteOperation:
		if resType == model.ResourceTypePod {
			err := processDeletePodDB(*message)
			if err != nil {
				klog.Errorf("delete pod meta failed, message %s, err: %v", msgDebugInfo(message), err)
				return fmt.Errorf("failed to delete pod meta to DB: %s", err)
			}
		} else {
			err := dao.DeleteMetaByKey(resKey)
			if err != nil {
				klog.Errorf("delete meta failed, %s", msgDebugInfo(message))
				return fmt.Errorf("delete meta failed, %s", err)
			}
		}
	}
	return nil
}

func processDeletePodDB(message model.Message) error {
	var msgPod corev1.Pod
	msgContent, err := message.GetContentData()
	if err != nil {
		return err
	}

	err = json.Unmarshal(msgContent, &msgPod)
	if err != nil {
		return err
	}

	num, err := dao.DeleteMetaByKeyAndPodUID(message.GetResource(), string(msgPod.UID))
	if err != nil {
		return err
	}
	if num == 0 {
		klog.V(2).Infof("don't need to delete pod DB")
		return nil
	}

	podPatchKey := strings.Replace(message.GetResource(),
		constants.ResourceSep+model.ResourceTypePod+constants.ResourceSep,
		constants.ResourceSep+model.ResourceTypePodPatch+constants.ResourceSep, 1)
	err = dao.DeleteMetaByKey(podPatchKey)
	if err != nil {
		return err
	}

	return nil
}

func (m *metaManager) processInsert(message model.Message) {
	if _, resType, _ := parseResource(&message); resType == model.ResourceTypeEvent {
		sendToCloud(&message)
		return
	}
	imitator.DefaultV2Client.Inject(message)

	msgSource := message.GetSource()
	if msgSource == modules.EdgedModuleName {
		if !connect.IsConnected() {
			klog.Warningf("process remote failed, req[%s], err: %v", msgDebugInfo(&message), errNotConnected)
			feedbackError(fmt.Errorf("failed to process remote: %s", errNotConnected), message)
			return
		}
		m.processRemote(message)
		return
	}
	if err := m.handleMessage(&message); err != nil {
		feedbackError(err, message)
		return
	}
	if msgSource == cloudmodules.DeviceControllerModuleName {
		message.SetRoute(modules.MetaGroup, modules.DeviceTwinModuleName)
		beehiveContext.Send(modules.DeviceTwinModuleName, message)
	} else if msgSource != cloudmodules.PolicyControllerModuleName {
		// Notify edged
		sendToEdged(&message, false)
	}

	resp := message.NewRespByMessage(&message, OK)
	sendToCloud(resp)
}

func (m *metaManager) processUpdate(message model.Message) {
	if _, resType, _ := parseResource(&message); resType == model.ResourceTypeEvent {
		sendToCloud(&message)
		return
	}
	imitator.DefaultV2Client.Inject(message)

	msgSource := message.GetSource()
	_, resType, _ := parseResource(&message)
	if msgSource == modules.EdgedModuleName && resType == model.ResourceTypeLease {
		if !connect.IsConnected() {
			klog.Warningf("process remote failed, req[%s], err: %v", msgDebugInfo(&message), errNotConnected)
			feedbackError(fmt.Errorf("failed to process remote: %s", errNotConnected), message)
			return
		}
		m.processRemote(message)
		return
	}
	if err := m.handleMessage(&message); err != nil {
		feedbackError(err, message)
		return
	}
	switch msgSource {
	case modules.EdgedModuleName:
		// For pod status update message, we need to wait for the response message
		// to ensure that the pod status is correctly reported to the kube-apiserver
		sendToCloud(&message)
		resp := message.NewRespByMessage(&message, OK)
		sendToEdged(resp, message.IsSync())
	case cloudmodules.EdgeControllerModuleName, cloudmodules.DynamicControllerModuleName:
		sendToEdged(&message, message.IsSync())
		resp := message.NewRespByMessage(&message, OK)
		sendToCloud(resp)
	case cloudmodules.DeviceControllerModuleName:
		resp := message.NewRespByMessage(&message, OK)
		sendToCloud(resp)
		message.SetRoute(modules.MetaGroup, modules.DeviceTwinModuleName)
		beehiveContext.Send(modules.DeviceTwinModuleName, message)
	case cloudmodules.PolicyControllerModuleName:
		resp := message.NewRespByMessage(&message, OK)
		sendToCloud(resp)
	case modules.MetaManagerModuleName:
		// Process the update message from MetaManager(MetaServer)
		// which is used to update the device in edge node.
		sendToTwin(&message)
		resp := message.NewRespByMessage(&message, OK)
		sentToMetamanager(resp, message.IsSync())
	default:
		klog.Errorf("unsupport message source, %s", msgSource)
	}
}

func (m *metaManager) processPatch(message model.Message) {
	if _, resType, _ := parseResource(&message); resType == model.ResourceTypeEvent {
		sendToCloud(&message)
		return
	}
	if err := m.handleMessage(&message); err != nil {
		feedbackError(err, message)
		return
	}

	if connect.IsConnected() {
		sendToCloud(&message)
	} else {
		feedbackError(connect.ErrConnectionLost, message)
	}
}

func (m *metaManager) processResponse(message model.Message) {
	if err := m.handleMessage(&message); err != nil {
		feedbackError(err, message)
		return
	}
	// Notify edged if the data is coming from cloud
	if message.GetSource() == CloudControllerModel {
		sendToEdged(&message, message.IsSync())
	} else {
		// Send to cloud if the update request is coming from edged
		sendToCloud(&message)
	}
}

func (m *metaManager) processDelete(message model.Message) {
	imitator.DefaultV2Client.Inject(message)
	_, resType, _ := parseResource(&message)
	if resType == model.ResourceTypePod && message.GetSource() == modules.EdgedModuleName {
		// if pod is deleted in K8s, then a new delete message will be sent to edge
		sendToCloud(&message)
		return
	}

	if err := m.handleMessage(&message); err != nil {
		feedbackError(err, message)
		return
	}
	msgSource := message.GetSource()
	if msgSource == cloudmodules.DeviceControllerModuleName {
		message.SetRoute(modules.MetaGroup, modules.DeviceTwinModuleName)
		beehiveContext.Send(modules.DeviceTwinModuleName, message)
	}

	if msgSource != cloudmodules.PolicyControllerModuleName {
		// Notify edged
		sendToEdged(&message, false)
	}
	resp := message.NewRespByMessage(&message, OK)
	sendToCloud(resp)
}

func (m *metaManager) processQuery(message model.Message) {
	resKey, resType, resID := parseResource(&message)
	var metas *[]string
	var err error
	if requireRemoteQuery(resType) && connect.IsConnected() {
		m.processRemote(message)
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
		feedbackError(fmt.Errorf("failed to query meta in DB: %s", err), message)
	} else {
		resp := message.NewRespByMessage(&message, *metas)
		resp.SetRoute(modules.MetaManagerModuleName, resp.GetGroup())
		sendToEdged(resp, message.IsSync())
	}
}

func (m *metaManager) processRemote(message model.Message) {
	go func() {
		// TODO: retry
		originalID := message.GetID()
		message.UpdateID()
		resp, err := beehiveContext.SendSync(
			string(metaManagerConfig.Config.ContextSendModule),
			message,
			time.Duration(metaManagerConfig.Config.RemoteQueryTimeout)*time.Second)
		if err != nil {
			klog.Errorf("process remote failed, req[%s], err: %v", msgDebugInfo(&message), err)
			feedbackError(fmt.Errorf("failed to process remote: %s", err), message)
			return
		}
		klog.V(4).Infof("process remote: req[%s], resp[%s]", msgDebugInfo(&message), msgDebugInfo(&resp))
		content, ok := resp.GetContent().(string)
		if ok && content == constants.MessageSuccessfulContent {
			klog.V(4).Infof("process remote successfully")
			feedbackResponse(&message, originalID, &resp)
			return
		}
		errContent, ok := resp.GetContent().(error)
		if ok {
			klog.V(4).Infof("process remote err: %v", errContent)
			feedbackResponse(&message, originalID, &resp)
			return
		}
		mapContent, ok := resp.GetContent().(map[string]interface{})
		respDB := resp
		if ok && isObjectResp(mapContent) {
			if mapContent["Err"] != nil {
				klog.V(4).Infof("process remote objResp err: %v", mapContent["Err"])
				feedbackResponse(&message, originalID, &resp)
				return
			}
			klog.V(4).Infof("process remote objResp: %+v", mapContent["Object"])
			respDB.Content = mapContent["Object"]
		}
		if err := m.handleMessage(&respDB); err != nil {
			feedbackError(err, message)
			return
		}
		feedbackResponse(&message, originalID, &resp)
	}()
}

func isObjectResp(data map[string]interface{}) bool {
	_, ok := data["Object"]
	if !ok {
		return false
	}
	_, ok = data["Err"]
	return ok
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
	case model.PatchOperation:
		m.processPatch(message)
	case model.DeleteOperation:
		m.processDelete(message)
	case model.QueryOperation:
		m.processQuery(message)
	case model.ResponseOperation:
		m.processResponse(message)
	case constants.CSIOperationTypeCreateVolume,
		constants.CSIOperationTypeDeleteVolume,
		constants.CSIOperationTypeControllerPublishVolume,
		constants.CSIOperationTypeControllerUnpublishVolume:
		m.processVolume(message)
	default:
		klog.Errorf("metamanager not supported operation: %v", operation)
	}
}

func (m *metaManager) runMetaManager() {
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
}
