package processor

import (
	"encoding/json"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

func init() {
	key := queryKey{
		operation: OperationMetaSync,
	}

	processors[key] = &metaSyncProcessor{}
}

// metaSyncProcessor process "meta-internal-sync"
type metaSyncProcessor struct {
}

func (m *metaSyncProcessor) Process(message model.Message) {
	syncPodStatus()
}

func syncPodStatus() {
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
		msg := model.NewMessage("").BuildRouter(modules.MetaManagerModuleName, GroupResource, namespace+constants.ResourceSep+model.ResourceTypePodStatus, model.UpdateOperation).FillBody(content)
		sendToCloud(msg)
		klog.V(3).Infof("sync pod status successfully for namespaces %s, %s", namespace, msgDebugInfo(msg))
	}
}
