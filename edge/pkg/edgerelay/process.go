package edgerelay

import (
	"bytes"
	"encoding/json"
	"fmt"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	hubConfig "github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/config"
	"github.com/kubeedge/kubeedge/edge/pkg/edgerelay/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/viaduct/pkg/mux"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NodeAddress struct {
	// Required
	IP string `json:"ip,omitempty"`
	// Required
	Port int64 `json:"port,omitempty"`
}

type RelayData struct {
	//
	AddrData map[string]NodeAddress `json:"addrdata,omitempty"`
}

// 接收cloud-crd传来的信息
func (er *EdgeRelay) RelayFromCenter() {
	relayID := ""
	// 提供接收信息服务
	// 从地面中心接收信息

	// 如果edgerelay.enable，调用HandleRelayMark
	if config.Config.Enable {
		er.HandleRelayMark(relayID)
	}

}

func (er *EdgeRelay) HandleRelayMark(relayID string) {
	// 调用SaveRelayID
	er.SaveRelayID(relayID)
	// if nodeID==relayID 创建一个MessageContainer类型信息，head头插入（"relayID"：relayID），调用MsgToOtherEdge
	if config.Config.NodeID == relayID {
		// 给EdgeHub的控制按钮标为true，允许EdgeHub与CloudHub通信

		// 调用MsgToEdgeHub，发送一个Operation为RelayMark的Message
		msg := &model.Message{}
		er.MsgToEdgeHub(msg)

		// 通知其他节点
		msgcontainer := &mux.MessageContainer{
			Header: map[string][]string{},
		}
		msgcontainer.Header.Add(constants.RelayID, relayID)
		nodes := er.GetAllAddress()
		for _, v := range nodes {
			er.MsgToOtherEdge(v, msgcontainer)
		}

	}

}

// finished
func (er *EdgeRelay) SaveRelayID(relayID string) {

	// 更新config和数据库
	config.Config.RelayID = relayID
	// 判断数据库中能不能查到，不能查到就insert，能查到就update
	meta := &dao.Meta{
		Key:   constants.RelayID,
		Type:  constants.RelayType,
		Value: string(relayID)}
	err := dao.InsertOrUpdate(meta)
	if err != nil {
		klog.Errorf("save relayId failed", err)
		return
	}
}

// finished
func (er *EdgeRelay) LoadRelayID() string {
	// 读取数据库中的中继信息，在每次启动的时候进行读取
	metas, err := dao.QueryMeta("key", constants.RelayID)
	if err != nil {
		klog.Errorf("query relayID failed")
	}
	if metas == nil {
		return ""
	}
	var result = *metas
	return result[0]
	// 如果能查到，返回值，如果查不到返回""
}

func (er *EdgeRelay) HandleMsgFromOtherEdge(container *mux.MessageContainer) {
	// if 检查container的头部是否有"relayID"字段，如果有，调用SaveRelayID
	relayID := container.Header.Get(constants.RelayID)
	if relayID != "" {
		er.SaveRelayID(relayID)
		// 当打开中继的时候，关闭EdgeHub与CloudHub直接通信的功能(后续会由MsgFromEdgeHub处理)
		// 发送一个通知建立链路的通知
		// 通知edgehub链路已经建立，可以开始让各个组件活跃起来发布消息
	} else {
		// else if(nodeID==relayID) 对信息进行封装，发送一个Operation为uploadrelay的Message，调用MsgToEdgeHub
		// 		else 对消息进行拆解，调用MsgToEdgeHub
		var msg *model.Message
		if config.Config.NodeID == config.Config.RelayID {
			msg = container.Message.Clone(container.Message)
			msg.SetResourceOperation(msg.GetResource(), constants.OpUploadRelayMessage)
			contentMsg, err := json.Marshal(*msg)

			if err != nil {
				fmt.Errorf("EdgeRelay SealMessage failed")
			}
			msg.Content = contentMsg
			er.MsgToEdgeHub(msg)
		} else {
			msg = container.Message
			er.MsgToEdgeHub(msg)
		}
	}

}

// finished
func (er *EdgeRelay) HandleMsgFromEdgeHub(msg *model.Message) {
	// 肯定是关于中继类型的信息，才会由EdgeHub发给Relay处理

	// if(nodeID==relayID) 先提取message的nodeID，再封层container格式,获取GetAddressThroughFile
	if config.Config.NodeID == config.Config.RelayID {
		container := &mux.MessageContainer{
			Header:  map[string][]string{},
			Message: msg,
		}
		nodeID, _ := GetNodeID(msg)
		nodeAddr := er.GetAddress(nodeID)
		er.MsgToOtherEdge(nodeAddr, container)
	} else {
		// else 封层container格式，添加自身nodeID和projectID，目标nodeID标为relayID
		container := &mux.MessageContainer{
			Header:  map[string][]string{},
			Message: msg,
		}
		container.Header.Add("node_id", config.Config.NodeID)
		container.Header.Add("project_id", hubConfig.Config.ProjectID)
		relayAddr := er.GetAddress(config.Config.RelayID)
		// 调用MsgToOtherEdge
		er.MsgToOtherEdge(relayAddr, container)
	}

}

func (er *EdgeRelay) MsgToEdgeHub(msg *model.Message) {
	// ch <- message
	beehiveContext.Send(modules.EdgeHubModuleName, *msg)
}

// 时刻接收其他组件传来的信息
func (er *EdgeRelay) MsgFromEdgeHub() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("EdgeRelay MsgFromEdgeHub stop")
			return
		default:
		}
		// ch <- message
		message, err := beehiveContext.Receive(modules.EdgeRelayModuleName)
		if err != nil {
			klog.Errorf("edgerelay failed to receive message from edgehub: %v", err)
			time.Sleep(time.Second)
		}
		// 调用HandleMsgFromEdgeHub
		er.HandleMsgFromEdgeHub(&message)
	}

}

// 给其他节点发送信息
func (er *EdgeRelay) MsgToOtherEdge(addr NodeAddress, container *mux.MessageContainer) {
	er.client(addr, container)

}

// 时刻接收其他节点的来信
func (er *EdgeRelay) MsgFromOtherEdge() {
	// Start Receive Msg Client
	// 调用HandleMsgFromOtherEdge
	er.server()
}
func (er *EdgeRelay) GetAddress(nodeID string) NodeAddress {
	return NodeAddress{}
}

// 从文件中读取所有的nodeID：addr键值对，返回一个map
func (er *EdgeRelay) GetAllAddress() map[string]NodeAddress {
	return nil
}

func (er *EdgeRelay) FindAndEqualID() bool {
	// relayID是否存在，nodeID与RelayID是否相同
	if config.Config.RelayID == "" {
		return true
	}
	if config.Config.RelayID == config.Config.NodeID {
		return true
	}
	return false
}

// finished, but need checked
func GetNodeID(msg *model.Message) (string, error) {
	resource := msg.Router.Resource
	tokens := strings.Split(resource, commonconst.ResourceSep)
	numOfTokens := len(tokens)
	for i, token := range tokens {
		if token == constants.ResourceNode && i+1 < numOfTokens && tokens[i+1] != "" {
			return tokens[i+1], nil
		}
	}

	return "", fmt.Errorf("no nodeID in Message.Router.Resource: %s", resource)
}

// server
func (er *EdgeRelay) server() {
	http.HandleFunc("/postMessage", er.receiveMessage)
	err := http.ListenAndServe("127.0.0.1:9090", nil)
	if err != nil {
		fmt.Println("net.Listen error :", err)
	}
}

func (er *EdgeRelay) receiveMessage(writer http.ResponseWriter, request *http.Request) {
	if request.Method == constants.POST {
		body, err := ioutil.ReadAll(request.Body)

		if err != nil {
			fmt.Println("Read failed:", err)
		}

		defer request.Body.Close()

		var container *mux.MessageContainer
		err = json.Unmarshal(body, container)

		if err != nil {
			fmt.Println("json format error:", err)
		}
		er.HandleMsgFromOtherEdge(container)

	}
}

// client
func (er *EdgeRelay) client(addr NodeAddress, container *mux.MessageContainer) {
	ip := addr.IP
	port := addr.Port

	var url string
	url = ip + strconv.FormatInt(port, 10) + "/postMessage"
	contentType := "application/json;charset=utf-8"

	b, err := json.Marshal(container)

	if err != nil {
		fmt.Println("json format error:", err)
	}

	body := bytes.NewBuffer(b)
	request, err := http.Post(url, contentType, body)
	if err != nil {
		fmt.Println("Post failed:", err)
	}
	defer request.Body.Close()

}
