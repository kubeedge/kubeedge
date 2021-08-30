package processor

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	metaManagerConfig "github.com/kubeedge/kubeedge/edge/pkg/metamanager/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
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

// insertProcessor process insert into database
type insertProcessor struct{}

func (m *insertProcessor) Process(message model.Message) error {
	content, err := message.GetContentData()
	if err != nil {
		klog.Errorf("get insert message content data failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to get insert message content data", message)
		return err
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
		return err
	}
	return nil
}

// updateProcessor process update database
type updateProcessor struct{}

func (m *updateProcessor) Process(message model.Message) error {
	content, err := message.GetContentData()
	if err != nil {
		klog.Errorf("get update message content data failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to get update message content data", message)
		return err
	}

	resKey, resType, _ := parseResource(message.GetResource())

	if resourceUnchanged(resType, resKey, content) {
		resp := message.NewRespByMessage(&message, OK)
		sendToEdged(resp, message.IsSync())
		klog.Infof("resource[%s] unchanged, no notice", resKey)
		return fmt.Errorf("resource not changed, no need to process")
	}

	meta := &dao.Meta{
		Key:   resKey,
		Type:  resType,
		Value: string(content)}
	err = dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("update meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to update meta to DB", message)
		return err
	}
	return nil
}

// deleteProcessor process delete database
type deleteProcessor struct{}

func (m *deleteProcessor) Process(message model.Message) error {
	err := dao.DeleteMetaByKey(message.GetResource())
	if err != nil {
		klog.Errorf("delete meta failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to delete meta to DB", message)
		return err
	}
	return nil
}

// queryProcessor process query
type queryProcessor struct{}

func (m *queryProcessor) Process(message model.Message) {
	resKey, resType, resID := parseResource(message.GetResource())
	var metas *[]string
	var err error
	if requireRemoteQuery(resType) && isConnected() {
		metas, err = dao.QueryMeta("key", resKey)
		if err != nil || len(*metas) == 0 || resType == model.ResourceTypeNode || resType == constants.ResourceTypeVolumeAttachment {
			processRemoteQuery(message)
		} else {
			resp := message.NewRespByMessage(&message, *metas)
			resp.SetRoute(modules.MetaManagerModuleName, resp.GetGroup())
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
		resp.SetRoute(modules.MetaManagerModuleName, resp.GetGroup())
		sendToEdged(resp, message.IsSync())
	}
}

// responseProcessor process response
type responseProcessor struct{}

func (m *responseProcessor) Process(message model.Message) error {
	content, err := message.GetContentData()
	if err != nil {
		klog.Errorf("get response message content data failed, %s", msgDebugInfo(&message))
		feedbackError(err, "Error to get response message content data", message)
		return err
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
		return err
	}
	return nil
}

// processRemoteQuery query info from cloud
func processRemoteQuery(message model.Message) {
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

		content, err := resp.GetContentData()
		if err != nil {
			klog.Errorf("get remote query response content data failed, %s", msgDebugInfo(&resp))
			feedbackError(err, "Error to get remote query response message content data", message)
			return
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

func isConnected() bool {
	return metaManagerConfig.Connected
}

// is resource type require remote query
func requireRemoteQuery(resType string) bool {
	return resType == model.ResourceTypeConfigmap ||
		resType == model.ResourceTypeSecret ||
		resType == constants.ResourceTypePersistentVolume ||
		resType == constants.ResourceTypePersistentVolumeClaim ||
		resType == constants.ResourceTypeVolumeAttachment ||
		resType == model.ResourceTypeNode ||
		resType == model.ResourceTypeServiceAccountToken
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

func msgDebugInfo(message *model.Message) string {
	return fmt.Sprintf("msgID[%s] source[%s] resource[%s] operation[%s]", message.GetID(), message.GetSource(), message.GetResource(), message.GetOperation())
}

func feedbackError(err error, info string, request model.Message) {
	errInfo := "Something wrong"
	if err != nil {
		errInfo = fmt.Sprintf(info+": %v", err)
	}

	errResponse := model.NewErrorMessage(&request, errInfo).SetRoute(modules.MetaManagerModuleName, request.GetGroup())
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
