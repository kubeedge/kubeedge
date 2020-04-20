package metamanager

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"beehive/adoptions/common/api"
	"beehive/pkg/common/log"
	"beehive/pkg/core"
	"beehive/pkg/core/model"

	"edge-core/pkg/common/message"
	"edge-core/pkg/common/modules"
	"edge-core/pkg/metamanager/dao"
	"k8s.io/api/core/v1"
)

const (
	startIndex          = 0
	meshNameIndex       = startIndex
	meshNamespaceIndex  = 1
	meshResTypeIndex    = 2
	meshResVersionIndex = 3

	IgnoreMessage       = "ignore the message"
	ReTryMessage        = "send again"
)

func isMeshMeta(resType string) bool {
	if resType == ResourceTypeService || resType == ResourceTypeServiceList || resType == ResourceTypeEndpoints || resType == ResourceTypeBackend || resType == ResourceTypeBackendList || resType == ResourceTypeGateway || resType == ResourceTypeVirtualService {
		return true
	}
	return false
}

func (m *metaManager) handleMeshMeta(message model.Message, resKey, resType string, content []byte) (bool, string) {
	// check version for EdgeMesh metaData
	if isMeshMeta(resType) {
		log.LOGGER.Infof("[MESH] get an mesh meta %v", message)
		switch resType {
		case ResourceTypeService:
			var svc v1.Service
			err := json.Unmarshal(content, &svc)
			if err != nil {
				log.LOGGER.Errorf("invalid %s : %s", resType, string(content))
				return true, IgnoreMessage
			}
			status, err := checkAndUpdateMeshMeta(svc.Name, svc.Namespace, ResourceTypeService, svc.ResourceVersion)
			if status != NewVersion {
				log.LOGGER.Warnf("Update MeshMeta data get an error , the resourceVersion is %+v", status)
				return true, IgnoreMessage
			}
			meta := &dao.Meta{
				Key:   resKey,
				Type:  resType,
				Value: string(content)}
			err = dao.InsertOrUpdate(meta)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true, ReTryMessage
			}
			meshMeta := generateMeshMeta(svc.Name, svc.Namespace, ResourceTypeService, svc.ResourceVersion)
			err = dao.InsertOrUpdate(meshMeta)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true, ReTryMessage
			}
		case ResourceTypeBackend:
			var bkd api.Backend
			err := json.Unmarshal(content, &bkd)
			if err != nil {
				log.LOGGER.Errorf("invalid %s : %s", resType, string(content))
				return true, IgnoreMessage
			}
			status, err := checkAndUpdateMeshMeta(bkd.Name, bkd.Namespace, ResourceTypeBackend, bkd.ResourceVersion)
			if status != NewVersion {
				log.LOGGER.Warnf("Delete MeshMeta data get an error , the resourceVersion is %+v", status)
				return true, IgnoreMessage
			}
			meta := &dao.Meta{
				Key:   resKey,
				Type:  resType,
				Value: string(content)}
			err = dao.InsertOrUpdate(meta)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", bkd, err)
				return true, ReTryMessage
			}
			meshMeta := generateMeshMeta(bkd.Name, bkd.Namespace, ResourceTypeBackend, bkd.ResourceVersion)
			err = dao.InsertOrUpdate(meshMeta)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true, ReTryMessage
			}
		case ResourceTypeServiceList:
			var svcList []v1.Service
			err := json.Unmarshal(content, &svcList)
			if err != nil {
				log.LOGGER.Errorf("Unmarshal update message content failed, %+v", svcList)
				return true, IgnoreMessage
			}
			for idx, svc := range svcList {
				status, err := checkAndUpdateMeshMeta(svc.Name, svc.Namespace, ResourceTypeService, svc.ResourceVersion)
				if status != NewVersion {
					log.LOGGER.Warnf("Delete MeshMeta data get an error , the resourceVersion is %+v", status)
					continue
				}
				data, err := json.Marshal(svc)
				if err != nil {
					log.LOGGER.Errorf("Marshal endpoints content failed, %v", svc)
					continue
				}
				meta := &dao.Meta{
					Key:   fmt.Sprintf("%s/%s/%s", svcList[idx].Namespace, ResourceTypeService, svcList[idx].Name),
					Type:  ResourceTypeService,
					Value: string(data)}
				err = dao.InsertOrUpdate(meta)
				if err != nil {
					log.LOGGER.Errorf("Update meta %s failed, svc: %v, err: %v", string(data), svc, err)
					return true, ReTryMessage
				}
				meshMeta := generateMeshMeta(svcList[idx].Name, svcList[idx].Namespace, ResourceTypeService, svc.ResourceVersion)
				err = dao.InsertOrUpdate(meshMeta)
				if err != nil {
					log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
					return true, ReTryMessage
				}
			}
		case ResourceTypeBackendList:
			var bkdList []api.Backend
			err := json.Unmarshal(content, &bkdList)
			if err != nil {
				log.LOGGER.Errorf("Unmarshal update message content failed, %+v", bkdList)
				return true, IgnoreMessage
			}
			for _, bkd := range bkdList {
				status, err := checkAndUpdateMeshMeta(bkd.Name, bkd.Namespace, ResourceTypeBackend, bkd.ResourceVersion)
				if status != NewVersion {
					log.LOGGER.Warnf("Delete MeshMeta data get an error , the resourceVersion is %+v", status)
					continue
				}
				data, err := json.Marshal(bkd)
				if err != nil {
					log.LOGGER.Errorf("Marshal endpoints content failed, %v", bkd)
					continue
				}
				meta := &dao.Meta{
					Key:   fmt.Sprintf("%s/%s/%s", bkd.Namespace, ResourceTypeBackend, bkd.Name),
					Type:  ResourceTypeBackend,
					Value: string(data)}
				err = dao.InsertOrUpdate(meta)
				if err != nil {
					log.LOGGER.Errorf("Update meta failed, %v", bkd)
					return true, ReTryMessage
				}
				meshMeta := generateMeshMeta(bkd.Name, bkd.Namespace, ResourceTypeBackend, bkd.ResourceVersion)
				err = dao.InsertOrUpdate(meshMeta)
				if err != nil {
					log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
					return true, ReTryMessage
				}
			}
		case ResourceTypeGateway, ResourceTypeVirtualService:
			meta := &dao.Meta{
				Key: resKey,
				Type: resType,
				Value: string(content)}
			err := dao.InsertOrUpdate(meta)
			if err != nil{
				log.LOGGER.Errorf("save meta failed, %s, err: %v",string(content), err)
				return true, ReTryMessage
			}
		default:
			log.LOGGER.Infof("get other mesh meta data , %s", resType)
			return true,IgnoreMessage
		}
		if resType != ResourceTypeGateway && resType != ResourceTypeVirtualService {
			send2EdgeMesh(&message, false, m.context)
		}
		message.FillBody(content)
		send2EventBus(&message, false, m.context)
		resp := message.NewRespByMessage(&message, OK)
		send2Cloud(resp, m.context)
		return true, ""
	}

	return false, ""
}

func (m *metaManager) handleDeleteMeshMeta(message model.Message) (bool,string) {
	resKey, resType, _ := parseResource(message.GetResource())
	if isMeshMeta(resType) {
		log.LOGGER.Infof("[MESH] get an mesh meta %v", message)
		content, err := json.Marshal(message.GetContent())
		if err != nil {
			log.LOGGER.Errorf("marshal update message content failed, %s, err: %v", msgDebugInfo(&message), err)
			return true,ReTryMessage
		}
		switch resType {
		case ResourceTypeService:
			var svc v1.Service
			err := json.Unmarshal(content, &svc)
			if err != nil {
				log.LOGGER.Errorf("invalid %s : %s", resType, string(content))
				return true,IgnoreMessage
			}
			status, err := checkAndUpdateMeshMeta(svc.Name, svc.Namespace, ResourceTypeService, svc.ResourceVersion)
			if status != SameVersion && status != NewVersion{
				log.LOGGER.Warnf("Delete MeshMeta data get an error , the resourceVersion is %+v", status)
				return true,IgnoreMessage
			}
			err = dao.DeleteMetaByKey(resKey)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true,ReTryMessage
			}
			meshMetaKey := generateMeshMetaKey(svc.Name, svc.Namespace, resType)
			err = dao.DeleteMetaByKey(meshMetaKey)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true,ReTryMessage
			}
		case ResourceTypeBackend:
			var bkd api.Backend
			err := json.Unmarshal(content, &bkd)
			if err != nil {
				log.LOGGER.Errorf("invalid %s : %s", resType, string(content))
				return true,IgnoreMessage
			}
			status, err := checkAndUpdateMeshMeta(bkd.Name, bkd.Namespace, ResourceTypeBackend, bkd.ResourceVersion)
			if status != SameVersion && status != NewVersion {
				log.LOGGER.Warnf("Delete MeshMeta data get an error , the resourceVersion is %+v", status)
				return true,IgnoreMessage
			}

			err = dao.DeleteMetaByKey(resKey)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true,ReTryMessage
			}
			meshMetaKey := generateMeshMetaKey(bkd.Name, bkd.Namespace, resType)
			err = dao.DeleteMetaByKey(meshMetaKey)
			if err != nil {
				log.LOGGER.Errorf("save meta failed, %s, err: %v", string(content), err)
				return true,ReTryMessage
			}
		case ResourceTypeGateway, ResourceTypeVirtualService:
			err = dao.DeleteMetaByKey(resKey)
			if err != nil {
				log.LOGGER.Errorf("delete meta failed, err:%v", err)
				return true,ReTryMessage
			}
		default:
			return true,IgnoreMessage
		}
		if resType != ResourceTypeGateway && resType != ResourceTypeVirtualService {
			send2EdgeMesh(&message, false, m.context)
		}
		message.FillBody(content)
		send2EventBus(&message, false, m.context)
		resp := message.NewRespByMessage(&message, OK)
		send2Cloud(resp, m.context)
		return true,""
	}
	return false,""
}

func compareResourceVersion(n, o string) (int64, error) {
	remote, err := strconv.ParseInt(n, 10, 64)
	if err != nil {
		return 0, err
	}
	local, err := strconv.ParseInt(o, 10, 64)
	if err != nil {
		return 0, err
	}
	return remote - local, err
}

func checkAndUpdateMeshMeta(name, namespace, resType, resourceVersion string) (VersionState, error) {
	metaKey := generateMeshMetaKey(name, namespace, resType)
	data, err := dao.QueryMeta("key", metaKey)
	if err != nil {
		log.LOGGER.Errorf("query meshMeta error :%s", err.Error())
		return UnknownVersion, err
	}
	if len(*data) != 0 {
		//if exist
		oldResourceVersion := getResourceVersionFromDB(*data)
		if oldResourceVersion == "" {
			oldResourceVersion = "0"
		}
		ret, err := compareResourceVersion(resourceVersion, oldResourceVersion)
		if err != nil {
			return UnknownVersion, err
		}
		if ret < 0 {
			return OldVersion, nil
		} else if ret == 0 {
			return SameVersion, nil
		}
	}
	//not exist,maybe a new one
	return NewVersion, nil
}

func getResourceVersionFromDB(content []string) string {
	if len(content) == 0 {
		return ""
	}
	value := strings.Split(content[startIndex], ResourceSeparator)
	if len(value) != meshResVersionIndex+1 {
		return ""
	}
	return value[meshResVersionIndex]
}

func generateMeshMeta(name, namespace, resType, resVersion string) *dao.Meta {
	return &dao.Meta{
		Key:   generateMeshMetaKey(name, namespace, resType),
		Type:  ResourceTypeMeshMeta,
		Value: name + ResourceSeparator + namespace + ResourceSeparator + resType + ResourceSeparator + resVersion,
	}
}

func generateMeshMetaKey(name, namespace, resType string) string {
	return name + ResourceSeparator + resType + ResourceSeparator + namespace
}

func (m *metaManager) syncMeshMeta() {
	//get all service
	content, err := dao.QueryMeta("type", ResourceTypeMeshMeta)
	if err != nil {
		return
	}
	syncData := &api.MeshMeta{
		ServiceMeta: make(map[string]string),
		BackendMeta: make(map[string]string),
	}
	//get all meshMeta
	for _, meshMeta := range *content {
		resName, resNamespace, resType, resVersion := parseMeshMeta(meshMeta)
		syncData.Namespace = resNamespace
		switch resType {
		case ResourceTypeService:
			syncData.ServiceMeta[resName] = resVersion
		case ResourceTypeBackend:
			syncData.BackendMeta[resName] = resVersion
		default:
			log.LOGGER.Warnf("invalid meshMeta type %s", resType)
		}
	}
	if syncData.Namespace == "" {
		syncData.Namespace = "null"
	}
	log.LOGGER.Infof("[MESH] Sync Mesh %v", syncData)
	resourceSync := fmt.Sprintf("%s/%s/%s", syncData.Namespace, model.ResourceTypeMeshMetas, "all")
	meshMetaSync := message.BuildMsg(core.MetaGroup, "", modules.EdgeMeshModuleName, resourceSync, model.UpdateOperation, syncData)
	send2Cloud(meshMetaSync, m.context)
}

func parseMeshMeta(data string) (string, string, string, string) {
	s := strings.Split(data, ResourceSeparator)
	if len(s) == meshResVersionIndex + 1 {
		return s[meshNameIndex], s[meshNamespaceIndex], s[meshResTypeIndex], s[meshResVersionIndex]
	}
	return "", "", "", ""
}
