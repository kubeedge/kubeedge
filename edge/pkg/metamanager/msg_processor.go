package metamanager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/common/util"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"

	"k8s.io/api/core/v1"
)

//Constants to check metamanager processes
const (
	OK = "OK"

	DefaultSyncInterval = 60
	GroupResource       = "resource"
	OperationMetaSync   = "meta-internal-sync"

	OperationFunctionAction = "action"

	OperationFunctionActionResult = "action_result"

	EdgeFunctionModel   = "edgefunction"
	CloudFunctionModel  = "funcmgr"
	CloudControlerModel = "controller"
)

var connected = false

// sendModuleGroupName is the name of the group to which we send the message
var sendModuleGroupName = modules.HubGroup

// sendModuleName is the name of send module for remote query
var sendModuleName = "websocket"

func init() {
	var err error
	groupName, err := config.CONFIG.GetValue("metamanager.context-send-group").ToString()
	if err == nil && groupName != "" {
		sendModuleGroupName = groupName
	}

	edgeSite, err := config.CONFIG.GetValue("metamanager.edgesite").ToBool()
	if err == nil && edgeSite == true {
		connected = true
	}

	moduleName, err := config.CONFIG.GetValue("metamanager.context-send-module").ToString()
	if err == nil && moduleName != "" {
		sendModuleName = moduleName
	}
}

func feedbackError(err error, info string, request model.Message, c *context.Context) {
	errInfo := "Something wrong"
	if err != nil {
		errInfo = fmt.Sprintf(info+": %v", err)
	}
	errResponse := model.NewErrorMessage(&request, errInfo).SetRoute(MetaManagerModuleName, request.GetGroup())
	if request.GetSource() == modules.EdgedModuleName {
		send2Edged(errResponse, request.IsSync(), c)
	} else {
		send2Cloud(errResponse, c)
	}
}

func send2Edged(message *model.Message, sync bool, c *context.Context) {
	if sync {
		c.SendResp(*message)
	} else {
		c.Send(modules.EdgedModuleName, *message)
	}
}

func send2EdgeMesh(message *model.Message, sync bool, c *context.Context) {
	if sync {
		c.SendResp(*message)
	} else {
		c.Send(modules.EdgeMeshModuleName, *message)
	}
}

func send2Cloud(message *model.Message, c *context.Context) {
	c.Send2Group(sendModuleGroupName, *message)
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
	return resType == model.ResourceTypeConfigmap || resType == model.ResourceTypeSecret || resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints
}

func isConnected() bool {
	return connected
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
			log.LOGGER.Errorf("marshal update message content failed, %s", msgDebugInfo(&message))
			feedbackError(err, "Error to marshal message content", message, m.context)
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
		log.LOGGER.Errorf("save meta failed, %s: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message, m.context)
		return
	}

	if resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints {
		// Notify edgemesh
		send2EdgeMesh(&message, false, m.context)
	} else {
		// Notify edged
		send2Edged(&message, false, m.context)
	}

	resp := message.NewRespByMessage(&message, OK)
	send2Cloud(resp, m.context)
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
			log.LOGGER.Errorf("marshal update message content failed, %s", msgDebugInfo(&message))
			feedbackError(err, "Error to marshal message content", message, m.context)
			return
		}
	}

	resKey, resType, _ := parseResource(message.GetResource())
	if resType == constants.ResourceTypeServiceList || resType == constants.ResourceTypeEndpointsList || resType == model.ResourceTypePodlist {
		switch resType {
		case constants.ResourceTypeEndpointsList:
			var epsList []v1.Endpoints
			err = json.Unmarshal(content, &epsList)
			if err != nil {
				log.LOGGER.Errorf("Unmarshal update message content failed, %s", msgDebugInfo(&message))
				feedbackError(err, "Error to unmarshal", message, m.context)
				return
			}
			for _, eps := range epsList {
				data, err := json.Marshal(eps)
				if err != nil {
					log.LOGGER.Errorf("Marshal endpoints content failed, %v", eps)
					continue
				}

				meta := &dao.Meta{
					Key:   fmt.Sprintf("%s/%s/%s", eps.Namespace, constants.ResourceTypeEndpoints, eps.Name),
					Type:  constants.ResourceTypeEndpoints,
					Value: string(data)}
				err = dao.InsertOrUpdate(meta)
				if err != nil {
					log.LOGGER.Errorf("Update meta failed, %v", eps)
					continue
				}
			}
			send2EdgeMesh(&message, false, m.context)
			resp := message.NewRespByMessage(&message, OK)
			send2Cloud(resp, m.context)
			return
		case constants.ResourceTypeServiceList:
			var svcList []v1.Service
			err = json.Unmarshal(content, &svcList)
			if err != nil {
				log.LOGGER.Errorf("Unmarshal update message content failed, %s", msgDebugInfo(&message))
				feedbackError(err, "Error to unmarshal", message, m.context)
				return
			}
			for _, svc := range svcList {
				data, err := json.Marshal(svc)
				if err != nil {
					log.LOGGER.Errorf("Marshal service content failed, %v", svc)
					continue
				}

				meta := &dao.Meta{
					Key:   fmt.Sprintf("%s/%s/%s", svc.Namespace, constants.ResourceTypeService, svc.Name),
					Type:  constants.ResourceTypeService,
					Value: string(data)}
				err = dao.InsertOrUpdate(meta)
				if err != nil {
					log.LOGGER.Errorf("Update meta failed, %v", svc)
					continue
				}
			}
			send2EdgeMesh(&message, false, m.context)
			resp := message.NewRespByMessage(&message, OK)
			send2Cloud(resp, m.context)
			return
		case model.ResourceTypePodlist:
			meta := &dao.Meta{
				Key:   resKey,
				Type:  resType,
				Value: string(content)}
			err = dao.InsertOrUpdate(meta)
			if err != nil {
				log.LOGGER.Errorf("Update meta failed, %s", msgDebugInfo(&message))
				feedbackError(err, "Error to update meta to DB", message, m.context)
				return
			}
			send2EdgeMesh(&message, false, m.context)
			resp := message.NewRespByMessage(&message, OK)
			send2Cloud(resp, m.context)
			return
		default:
			log.LOGGER.Warnf("Resource type %s unknown", resType)
			return
		}
	}

	if resourceUnchanged(resType, resKey, content) {
		resp := message.NewRespByMessage(&message, OK)
		send2Edged(resp, message.IsSync(), m.context)
		log.LOGGER.Infof("resouce[%s] unchanged, no notice", resKey)
		return
	}

	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		log.LOGGER.Errorf("update meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to update meta to DB", message, m.context)
		return
	}

	switch message.GetSource() {
	//case core.EdgedModuleName:
	case modules.EdgedModuleName:
		send2Cloud(&message, m.context)
		resp := message.NewRespByMessage(&message, OK)
		send2Edged(resp, message.IsSync(), m.context)
	case CloudControlerModel:
		if resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints {
			send2EdgeMesh(&message, message.IsSync(), m.context)
		} else {
			send2Edged(&message, message.IsSync(), m.context)
		}
		resp := message.NewRespByMessage(&message, OK)
		send2Cloud(resp, m.context)
	case CloudFunctionModel:
		m.context.Send(EdgeFunctionModel, message)
	case EdgeFunctionModel:
		send2Cloud(&message, m.context)
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
			log.LOGGER.Errorf("marshal response message content failed, %s", msgDebugInfo(&message))
			feedbackError(err, "Error to marshal message content", message, m.context)
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
		log.LOGGER.Errorf("update meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to update meta to DB", message, m.context)
		return
	}

	// Notify edged or edgemesh if the data if coming from cloud
	if message.GetSource() == CloudControlerModel {
		if resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints {
			send2EdgeMesh(&message, message.IsSync(), m.context)
		} else {
			send2Edged(&message, message.IsSync(), m.context)
		}
	} else {
		// Send to cloud if the update request is coming from edged
		send2Cloud(&message, m.context)
	}
}

func (m *metaManager) processDelete(message model.Message) {
	err := dao.DeleteMetaByKey(message.GetResource())
	if err != nil {
		log.LOGGER.Errorf("delete meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to delete meta to DB", message, m.context)
		return
	}

	_, resType, _ := parseResource(message.GetResource())
	if resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints {
		// Notify edgemesh
		send2EdgeMesh(&message, false, m.context)
		resp := message.NewRespByMessage(&message, OK)
		send2Cloud(resp, m.context)
		return
	}
	// Notify edged
	send2Edged(&message, false, m.context)
	resp := message.NewRespByMessage(&message, OK)
	send2Cloud(resp, m.context)
}

func (m *metaManager) processQuery(message model.Message) {
	resKey, resType, resID := parseResource(message.GetResource())
	var metas *[]string
	var err error
	if requireRemoteQuery(resType) && isConnected() {
		metas, err = dao.QueryMeta("key", resKey)
		if err != nil || len(*metas) == 0 {
			m.processRemoteQuery(message)
		} else {
			resp := message.NewRespByMessage(&message, *metas)
			resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
			send2Edged(resp, message.IsSync(), m.context)
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
		log.LOGGER.Errorf("query meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to query meta in DB", message, m.context)
	} else {
		resp := message.NewRespByMessage(&message, *metas)
		resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
		if resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints || resType == model.ResourceTypePodlist {
			send2EdgeMesh(resp, message.IsSync(), m.context)
		} else {
			send2Edged(resp, message.IsSync(), m.context)
		}
	}
}

func (m *metaManager) processRemoteQuery(message model.Message) {
	go func() {
		// TODO: retry
		originalID := message.GetID()
		message.UpdateID()
		resp, err := m.context.SendSync(sendModuleName, message, 60*time.Second) // TODO: configurable
		log.LOGGER.Infof("########## process get: req[%+v], resp[%+v], err[%+v]", message, resp, err)
		if err != nil {
			log.LOGGER.Errorf("remote query failed: %v", err)
			feedbackError(err, "Error to query meta in DB", message, m.context)
			return
		}

		var content []byte
		switch resp.GetContent().(type) {
		case []uint8:
			content = resp.GetContent().([]byte)
		default:
			content, err = json.Marshal(resp.GetContent())
			if err != nil {
				log.LOGGER.Errorf("marshal remote query response content failed, %s", msgDebugInfo(&resp))
				feedbackError(err, "Error to marshal message content", message, m.context)
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
			log.LOGGER.Errorf("update meta failed, %s", msgDebugInfo(&resp))
		}
		resp.BuildHeader(resp.GetID(), originalID, resp.GetTimestamp())
		if resType == constants.ResourceTypeService || resType == constants.ResourceTypeEndpoints {
			send2EdgeMesh(&resp, message.IsSync(), m.context)
		} else {
			send2Edged(&resp, message.IsSync(), m.context)
		}
	}()
}

func (m *metaManager) processNodeConnection(message model.Message) {
	content, _ := message.GetContent().(string)
	log.LOGGER.Infof("node connection event occur: %s", content)
	if content == connect.CloudConnected {
		connected = true
	} else if content == connect.CloudDisconnected {
		connected = false
	}
}

func (m *metaManager) processSync(message model.Message) {
	m.syncPodStatus()
}

func (m *metaManager) syncPodStatus() {
	log.LOGGER.Infof("start to sync pod status")
	podStatusRecords, err := dao.QueryAllMeta("type", model.ResourceTypePodStatus)
	if err != nil {
		log.LOGGER.Errorf("list pod status failed: %v", err)
		return
	}
	if len(*podStatusRecords) <= 0 {
		log.LOGGER.Infof("list pod status, no record, skip sync")
		return
	}

	var namespace string
	content := make([]interface{}, 0, len(*podStatusRecords))
	for _, v := range *podStatusRecords {
		if namespace == "" {
			namespace, _, _, _ = util.ParseResourceEdge(v.Key, model.QueryOperation)
		}
		podKey := strings.Replace(v.Key, constants.ResourceSep+model.ResourceTypePodStatus+constants.ResourceSep, constants.ResourceSep+model.ResourceTypePod+constants.ResourceSep, 1)
		podRecord, err := dao.QueryMeta("key", podKey)
		if err != nil {
			log.LOGGER.Errorf("query pod[%s] failed: %v", podKey, err)
			return
		}

		if len(*podRecord) <= 0 {
			// pod already deleted, clear the corresponding podstatus record
			err = dao.DeleteMetaByKey(v.Key)
			log.LOGGER.Infof("pod[%s] already deleted, clear podstatus record, result:%v", podKey, err)
			continue
		}

		var podStatus interface{}
		err = json.Unmarshal([]byte(v.Value), &podStatus)
		if err != nil {
			log.LOGGER.Errorf("unmarshal podstatus[%s] failed, content[%s]: %v", v.Key, v.Value, err)
			continue
		}
		content = append(content, podStatus)
	}

	msg := model.NewMessage("").BuildRouter(MetaManagerModuleName, GroupResource, namespace+constants.ResourceSep+model.ResourceTypePodStatus, model.UpdateOperation).FillBody(content)
	send2Cloud(msg, m.context)
	log.LOGGER.Infof("sync pod status successful, %s", msgDebugInfo(msg))
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
			log.LOGGER.Errorf("marshal save message content failed, %s: %v", msgDebugInfo(&message), err)
			feedbackError(err, "Error to marshal message content", message, m.context)
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
		log.LOGGER.Errorf("save meta failed, %s: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message, m.context)
		return
	}

	m.context.Send(EdgeFunctionModel, message)
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
			log.LOGGER.Errorf("marshal save message content failed, %s: %v", msgDebugInfo(&message), err)
			feedbackError(err, "Error to marshal message content", message, m.context)
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
		log.LOGGER.Errorf("save meta failed, %s: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message, m.context)
		return
	}

	send2Cloud(&message, m.context)

}
func (m *metaManager) process(message model.Message) {
	resource := message.GetOperation()
	switch resource {
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
		m.processSync(message)
	case OperationFunctionAction:
		m.processFunctionAction(message)
	case OperationFunctionActionResult:
		m.processFunctionActionResult(message)
	}
}

func (m *metaManager) mainLoop() {
	go func() {
		for {
			if msg, err := m.context.Receive(m.Name()); err == nil {
				log.LOGGER.Infof("get a message %+v", msg)
				m.process(msg)
			} else {
				log.LOGGER.Errorf("get a message %+v: %v", msg, err)
			}
		}
	}()
}
