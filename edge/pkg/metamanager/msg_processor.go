package metamanager

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"beehive/pkg/common/log"
	"beehive/pkg/common/util"
	"beehive/pkg/core"
	"beehive/pkg/core/context"
	"beehive/pkg/core/model"
	messagepkg "edge-core/pkg/common/message"
	"edge-core/pkg/common/modules"
	"edge-core/pkg/metamanager/dao"
	"edge-core/pkg/metamanager/nodegroup"
)

type VersionState uint16

const (
	ResourceSeparator = "/"
	OK                = "OK"

	DefaultSyncInterval = 60
	GroupResource       = "resource"
	OperationMetaSync   = "meta-internal-sync"

	OperationFunctionAction = "action"

	OperationFunctionActionResult = "action_result"

	OperationReloadFunction = "reloadFunction"
	OperationImagePrepull   = "prepull"

	ResourceTypeNodeGroup      = "nodegroup"
	ResourceTypeService        = "service"
	ResourceTypeServiceList    = "servicelist"
	ResourceTypeEndpoints      = "endpoints"
	ResourceTypeEndpointsList  = "endpointslist"
	ResourceTypeListener       = "Listener"
	ResourceTypeBackend        = "backend"
	ResourceTypeBackendList    = "backendlist"
	ResourceTypeMeshMeta       = "meshmetas"
	ResourceTypeGateway        = "gateway"
	ResourceTypeVirtualService = "virtualservice"

	EdgeFunctionModel   = "edgefunction"
	CloudFunctionModel  = "funcmgr"
	CloudControlerModel = "controller"
	EdgeLoggerModel     = "logger_remote"
	EdgeMonitorModel    = "monitor_remote"
	AlarmModel          = "alarm"

	UnknownVersion = VersionState(0)
	SameVersion = VersionState(1)
	OldVersion  = VersionState(2)
	NewVersion  = VersionState(3)
)

var connected = false

func feedbackError(err error, info string, request model.Message, c *context.Context) {
	errInfo := "Something wrong"
	if err != nil {
		errInfo = fmt.Sprintf(info+": %v", err)
	}
	errResponse := model.NewErrorMessage(&request, errInfo).SetRoute(MetaManagerModuleName, request.GetGroup())
	if request.GetSource() == core.EdgedModuleName {
		send2Edged(errResponse, request.IsSync(), c)
	} else {
		send2Cloud(errResponse, c)
	}
}

func send2Edged(message *model.Message, sync bool, c *context.Context) {
	if sync {
		c.SendResp(*message)
	} else {
		c.Send(core.EdgedModuleName, *message)
	}
}

//send virtualservice or gateway to eventbus
func send2EventBus(message *model.Message, sync bool, c *context.Context) {
	operation := message.GetOperation()
	resourceString := ResourceSend2EventBus(operation, message.GetResource())
	message.BuildRouter(message.Router.Source, core.BusGroup, resourceString, "publish")
	if sync {
		c.SendResp(*message)
	} else {
		c.Send("eventbus", *message)
	}
}

//the resource send to eventbus
func ResourceSend2EventBus(operation string, str string) string {
	_, resourceType, resourceId := parseResource(str)
	resource := "$hw/events/" + resourceType + "/" + resourceId + "/" + operation
	return resource
}

func send2EdgeMesh(message *model.Message, sync bool, c *context.Context) {
	if sync {
		c.SendResp(*message)
	} else {
		c.Send(modules.EdgeMeshModuleName, *message)
	}
}

func send2Alarm(message *model.Message, sync bool, c *context.Context) {
	if sync {
		c.SendResp(*message)
	} else {
		c.Send2Group(AlarmModel, *message)
	}
}

func send2Cloud(message *model.Message, c *context.Context) {
	c.Send2Group(core.HubGroup, *message)
}

func send2Remote(module string, message *model.Message, sync bool, c *context.Context) {
	var err error
	if sync {
		if message.GetSource() == EdgeLoggerModel || message.GetSource() == EdgeMonitorModel {
			message, err = pack(message)
			if err != nil {
				log.LOGGER.Errorf("Failed to pack message, err: %v", err)
				return
			}
		}

		c.SendResp(*message)
	} else {
		if module == EdgeLoggerModel || module == EdgeMonitorModel {
			message, err = pack(message)
			if err != nil {
				log.LOGGER.Errorf("Failed to pack message, err: %v", err)
				return
			}
		}

		c.Send(module, *message)
	}
}

func pack(message *model.Message) (*model.Message, error) {
	content := message.GetContent()
	byteContent, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("Marshal failed, err: %v", err)
	}

	data, err := compress(byteContent)
	if err != nil {
		return nil, err
	}

	base64content := base64.StdEncoding.EncodeToString(data)
	message = message.FillBody(base64content)

	return message, nil
}

func compress(content []byte) ([]byte, error) {
	var data bytes.Buffer
	w := zlib.NewWriter(&data)
	defer w.Close()

	n, err := w.Write(content)
	if n <= 0 || err != nil {
		return nil, err
	}

	err = w.Flush()
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

// Resource format: <namespace>/<restype>[/resid]
// return <reskey, restype, resid>
func parseResource(resource string) (string, string, string) {
	tokens := strings.Split(resource, ResourceSeparator)
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
	return resType == model.ResourceTypeConfigmap || resType == model.ResourceTypeSecret || resType == ResourceTypeEndpoints
}

func isConnected() bool {
	return connected
}

func msgDebugInfo(message *model.Message) string {
	return fmt.Sprintf("msgID[%s] source[%s] resource[%s]", message.GetID(), message.GetSource(), message.GetResource())
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
	content, err := json.Marshal(message.GetContent())
	if err != nil {
		log.LOGGER.Errorf("marshal save message content failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to marshal message content", message, m.context)
		return
	}

	resKey, resType, _ := parseResource(message.GetResource())
	isMeshMeta,errString := m.handleMeshMeta(message,resKey,resType,content)

	if isMeshMeta {
		if errString == IgnoreMessage {
			log.LOGGER.Errorf("marshal save message content failed, %s, err: %v", msgDebugInfo(&message), err)
			resp := message.NewRespByMessage(&message, OK)
			send2Cloud(resp, m.context)
		}
		return
	}
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		log.LOGGER.Errorf("save meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message, m.context)
		return
	}

	if resType == ResourceTypeListener {
		// Notify edgemesh
		resp := message.NewRespByMessage(&message, nil)
		send2EdgeMesh(resp, true, m.context)
		return
	}
	// Notify edged
	send2Edged(&message, false, m.context)

	if strings.Contains(message.GetResource(), model.ResourceTypeConfigmap) {
		send2Remote(EdgeLoggerModel, &message, false, m.context)
	}
	resp := message.NewRespByMessage(&message, OK)
	send2Cloud(resp, m.context)
}

func (m *metaManager) processUpdate(message model.Message) {
	resKey, resType, _ := parseResource(message.GetResource())
	switch resType {
	case ResourceTypeNodeGroup:
		m.processNodeGroupMessage(message)
	default:
		m.processMetaMessage(resKey, resType, message)
	}
}

func (m *metaManager) processNodeGroupMessage(message model.Message) {
	err := nodegroup.GetManager().Handle(&message)
	if err != nil {
		log.LOGGER.Errorf("failed to handle group config message, error: %s", err.Error())
	}
	resp := message.NewRespByMessage(&message, OK)
	send2Cloud(resp, m.context)
}

func (m *metaManager) processMetaMessage(resKey, resType string, message model.Message) {
	content, err := json.Marshal(message.GetContent())
	if err != nil {
		log.LOGGER.Errorf("marshal update message content failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to marshal message content", message, m.context)
		return
	}
	isMeshMeta,errString := m.handleMeshMeta(message,resKey,resType,content)
	if isMeshMeta {
		if errString == IgnoreMessage {
			log.LOGGER.Errorf("marshal save message content failed, %s, err: %v", msgDebugInfo(&message), err)
			resp := message.NewRespByMessage(&message, OK)
			send2Cloud(resp, m.context)
		}
		return
	}
	if resourceUnchanged(resType, resKey, content) {
		resp := message.NewRespByMessage(&message, OK)
		if message.IsSync() {
			send2Edged(resp, message.IsSync(), m.context)
		}
		log.LOGGER.Debugf("resouce[%s] unchanged, no notice", resKey)
		return
	}

	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		log.LOGGER.Errorf("update meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to update meta to DB", message, m.context)
	}

	switch message.GetSource() {
	case core.EdgedModuleName:
		send2Cloud(&message, m.context)
		resp := message.NewRespByMessage(&message, OK)
		if message.IsSync() {
			send2Edged(resp, message.IsSync(), m.context)
		}
	case CloudControlerModel:
		send2Edged(&message, message.IsSync(), m.context)
		if strings.Contains(message.GetResource(), model.ResourceTypeConfigmap) {
			send2Remote(EdgeLoggerModel, &message, false, m.context)
		}
		resp := message.NewRespByMessage(&message, OK)
		send2Cloud(resp, m.context)
	case CloudFunctionModel:
		m.context.Send(EdgeFunctionModel, message)
	case EdgeFunctionModel:
		send2Cloud(&message, m.context)
	case AlarmModel:
		resp := message.NewRespByMessage(&message, OK)
		send2Alarm(resp, message.IsSync(), m.context)
	}
}

func (m *metaManager) processResponse(message model.Message) {
	content, err := json.Marshal(message.GetContent())
	if err != nil {
		log.LOGGER.Errorf("marshal response message content failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to marshal message content", message, m.context)
		return
	}

	resKey, resType, _ := parseResource(message.GetResource())
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		log.LOGGER.Errorf("update meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to update meta to DB", message, m.context)
		return
	}

	// Notify edged if the data if coming from cloud
	if message.GetSource() != core.EdgedModuleName {
		if resType == ResourceTypeService || resType == ResourceTypeEndpoints {
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
	isMeshMeta,errString := m.handleDeleteMeshMeta(message)
	if isMeshMeta {
		if errString != IgnoreMessage {
			resp := message.NewRespByMessage(&message, OK)
			send2Cloud(resp, m.context)
		}
		return
	}
	err := dao.DeleteMetaByKey(message.GetResource())
	if err != nil {
		log.LOGGER.Errorf("delete meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to delete meta to DB", message, m.context)
		return
	}

	_, resType, _ := parseResource(message.GetResource())
	if resType == ResourceTypeListener {
		resp := message.NewRespByMessage(&message, OK)
		send2EdgeMesh(resp, true, m.context)
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
			if message.GetSource() == EdgeLoggerModel || message.GetSource() == EdgeMonitorModel {
				send2Remote(message.GetSource(), resp, message.IsSync(), m.context)
			} else if message.GetSource() == AlarmModel {
				resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
				send2Alarm(resp, message.IsSync(), m.context)
			} else {
				resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
				send2Edged(resp, message.IsSync(), m.context)
			}
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
		log.LOGGER.Errorf("query meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to query meta in DB", message, m.context)
	} else {
		resp := message.NewRespByMessage(&message, *metas)
		if message.GetSource() == EdgeLoggerModel || message.GetSource() == EdgeMonitorModel {
			send2Remote(message.GetSource(), resp, message.IsSync(), m.context)
		} else if message.GetSource() == AlarmModel {
			resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
			send2Alarm(resp, message.IsSync(), m.context)
		} else {
			resp.SetRoute(MetaManagerModuleName, resp.GetGroup())
			if resType == ResourceTypeService || resType == ResourceTypeEndpoints || resType == model.ResourceTypePodlist || resType == ResourceTypeListener {
				send2EdgeMesh(resp, message.IsSync(), m.context)
			} else {
				send2Edged(resp, message.IsSync(), m.context)
			}
		}
	}
}

func (m *metaManager) processRemoteQuery(message model.Message) {
	go func() {
		// TODO: retry
		originalID := message.GetID()
		message.UpdateID()
		resp, err := m.context.SendSync("websocket", message, 60*time.Second) // TODO: configurable
		log.LOGGER.Infof("########## process get: req[%+v], resp[%+v, %+v], err[%+v]", message, resp.Header, resp.Router, err)
		if err != nil {
			log.LOGGER.Errorf("remote query failed, err: %v", err)
			feedbackError(err, "Error to query meta in DB", message, m.context)
			return
		}

		content, err := json.Marshal(resp.GetContent())
		if err != nil {
			log.LOGGER.Errorf("marshal remote query response content failed, %s, err: %v", msgDebugInfo(&resp), err)
		}

		resKey, resType, _ := parseResource(message.GetResource())
		meta := &dao.Meta{
			Key:   resKey,
			Type:  resType,
			Value: string(content)}
		err = dao.InsertOrUpdate(meta)
		if err != nil {
			log.LOGGER.Errorf("update meta failed, %s, err: %v", msgDebugInfo(&resp), err)
		}
		resp.BuildHeader(resp.GetID(), originalID, resp.GetTimestamp())
		if resType == ResourceTypeService || resType == ResourceTypeEndpoints {
			send2EdgeMesh(&resp, message.IsSync(), m.context)
		} else {
			send2Edged(&resp, message.IsSync(), m.context)
		}
	}()
}

func (m *metaManager) processNodeConnection(message model.Message) {
	content, _ := message.GetContent().(string)
	log.LOGGER.Infof("node connection event occur: %s", content)
	if content == model.CLOUD_CONNECTED {
		connected = true
		// notify cloud to refresh node groups id to edge.
		go nodegroup.GetManager().RefreshNodeGroupsId()
		go m.syncMeshMeta()
	} else if content == model.CLOUD_DISCONNECTED {
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
		log.LOGGER.Errorf("list pod status failed, err: %v", err)
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
		podKey := strings.Replace(v.Key, ResourceSeparator+model.ResourceTypePodStatus+ResourceSeparator, ResourceSeparator+model.ResourceTypePod+ResourceSeparator, 1)
		podRecord, err := dao.QueryMeta("key", podKey)
		if err != nil {
			log.LOGGER.Errorf("query pod[%s] failed, err: %v", podKey, err)
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
			log.LOGGER.Errorf("unmarshal podstatus[%s] failed, content[%s], err: %v", v.Key, v.Value, err)
			continue
		}
		content = append(content, podStatus)
	}

	msg := model.NewMessage("").BuildRouter(MetaManagerModuleName, GroupResource, namespace+ResourceSeparator+model.ResourceTypePodStatus, model.UpdateOperation).FillBody(content)
	send2Cloud(msg, m.context)
	log.LOGGER.Infof("sync pod status successful, %s", msgDebugInfo(msg))
}

func (m *metaManager) processFunctionAction(message model.Message) {
	content, err := json.Marshal(message.GetContent())
	if err != nil {
		log.LOGGER.Errorf("marshal save message content failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to marshal message content", message, m.context)
		return
	}

	resKey, resType, _ := parseResource(message.GetResource())
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.SaveMeta(meta)
	if err != nil {
		log.LOGGER.Errorf("save meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message, m.context)
		return
	}

	m.context.Send(EdgeFunctionModel, message)
}

func (m *metaManager) processFunctionActionResult(message model.Message) {
	content, err := json.Marshal(message.GetContent())
	if err != nil {
		log.LOGGER.Errorf("marshal save message content failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to marshal message content", message, m.context)
		return
	}

	resKey, resType, _ := parseResource(message.GetResource())
	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		log.LOGGER.Errorf("save meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to save meta to DB", message, m.context)
		return
	}

	send2Cloud(&message, m.context)

}

func (m *metaManager) processGetDeployment(message model.Message) {
	resKey, _, _ := parseResource(message.GetResource())
	results, err := dao.QueryFuzzyMeta("key", resKey)
	if err != nil {
		log.LOGGER.Errorf("query meta failed, %s, err: %v", msgDebugInfo(&message), err)
		feedbackError(err, "Error to query meta in DB", message, m.context)
	} else {
		msg := model.NewMessage(message.GetID()).BuildRouter("", "", message.GetResource(), OperationReloadFunction).FillBody(results)
		m.context.SendResp(*msg)
	}

}

func (m *metaManager) processImagePrepull(message model.Message) {
	send2Edged(&message, message.IsSync(), m.context)
	resp := message.NewRespByMessage(&message, OK)
	send2Cloud(resp, m.context)
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
	case OperationReloadFunction:
		m.processGetDeployment(message)
	case OperationImagePrepull:
		m.processImagePrepull(message)
	}
}

func (m *metaManager) mainLoop() {
	go func() {
		for {
			if msg, err := m.context.Receive(m.Name()); err == nil {

				m.process(msg)
			} else {
				log.LOGGER.Errorf("get a message: %v, err: %v", msgDebugInfo(&msg), err)
			}
		}
	}()
}
