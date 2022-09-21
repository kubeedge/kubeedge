package cloudrelay

import (
	"context"
	"encoding/json"
	"fmt"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	relayconstants "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/cloudrelay/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/cloudrelay/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/viaduct/pkg/mux"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sync"
)

//1 能够保存relay信息
//2 能够处理消息，再up or down stream

var once sync.Once

type CloudRelay struct {
	enable     bool
	relayID    string
	kubeClient kubernetes.Interface
}

var RelayHandle *CloudRelay

func InitCloudRelay() {
	// init的后去k8s的config查询有没有存储下来的中继数据，这部分在server.go
	once.Do(func() {
		RelayHandle = &CloudRelay{
			enable:     true,
			relayID:    "",
			kubeClient: client.GetKubeClient(),
		}
	})
}

func (relayHandle *CloudRelay) SaveRelayMark(container *mux.MessageContainer) {
	relayID := container.Header.Get("node_id")

	relayHandle.SetRelayID(relayID)
	// 存储到configmap，需要启动时候查询
	relayConfigMap := &v1.ConfigMap{}
	relayConfigMap.Name = relayconstants.CloudRelayConfigMap
	relayConfigMap.Namespace = "default"
	relayConfigMap.Data = make(map[string]string)

	if _, err := relayHandle.kubeClient.CoreV1().ConfigMaps(relayconstants.DefaultNameSpace).Get(context.Background(), relayConfigMap.Name, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			if _, err := relayHandle.kubeClient.CoreV1().ConfigMaps(relayconstants.DefaultNameSpace).Create(context.Background(), relayConfigMap, metav1.CreateOptions{}); err != nil {
				klog.Errorf("Failed to create config map for relay, error %v", err)
			}
		}
		return
	}
	relayConfigMap.Data[relayconstants.RelayID] = relayID
	if _, err := relayHandle.kubeClient.CoreV1().ConfigMaps(relayconstants.DefaultNameSpace).Update(context.Background(), relayConfigMap, metav1.UpdateOptions{}); err != nil {
		klog.Errorf("Failed to update config map for relay, error %v", err)
		return
	}
	// relayHandle.configMapManager.ConfigMap.Store("relayID", relayHandle.relayID)

}

func (relayHandle *CloudRelay) LoadRelayID() {
	// relayID, ok := relayHandle.configMapManager.ConfigMap.Load("relayID")

	relayConfigMap, err := relayHandle.kubeClient.CoreV1().ConfigMaps(relayconstants.DefaultNameSpace).Get(context.Background(), relayconstants.CloudRelayConfigMap, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("read Relay ConfigMap failed, error %v", err)
		return
	}
	if err != nil && apierrors.IsNotFound(err) {
		klog.Errorf("Relay ConfigMap not found, error %v", err)
		return
	}
	relayHandle.SetRelayID(relayConfigMap.Data[relayconstants.RelayID])
}

func (relayHandle *CloudRelay) SealMessage(msg *beehiveModel.Message) (string, *beehiveModel.Message, error) {
	nodeID := relayHandle.GetRelayID()
	oldID, resource, err := messagelayer.BuildResource(nodeID, msg.Router.Resource)
	if err != nil {
		return oldID, msg, fmt.Errorf("build relay node resource failed")
	}
	relayMsg := *msg
	contentMsg, err := json.Marshal(msg)

	relayMsg.Router.Resource = resource
	relayMsg.Router.Group = relayconstants.RelayGroupName
	relayMsg.Content = contentMsg
	return oldID, &relayMsg, nil
}

func (relayHandle *CloudRelay) UnsealMessage(container *mux.MessageContainer) *mux.MessageContainer {
	var rcontainer mux.MessageContainer
	err := json.Unmarshal(container.Message.GetContent().([]byte), &rcontainer)
	if err != nil {
		klog.V(4).Infof("RelayHandleServer Unmarshal failed", err)
	}
	return &rcontainer
}

func (relayHandle *CloudRelay) FindAndEqualID(nodeID string) bool {
	if relayHandle.GetRelayID() == "" {
		return true
	}
	if nodeID == relayHandle.GetRelayID() {
		return true
	}
	return false
}

func (relayHandle *CloudRelay) GetRelayID() string {
	return relayHandle.relayID
}
func (relayHandle *CloudRelay) SetRelayID(id string) {
	relayHandle.relayID = id
}

func (relayHandle *CloudRelay) ChangeDesToRelay(msg *beehiveModel.Message) (string, *beehiveModel.Message) {

	oldID, rmsg, err := relayHandle.SealMessage(msg)
	if err != nil {
		fmt.Errorf("ChangeDesToRelay failed")
		return oldID, msg
	}

	return oldID, rmsg
}
