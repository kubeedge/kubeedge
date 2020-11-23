package mqtt

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
)

const UploadTopic = "SYS/dis/upload_records"

var (
	// MQTTHub client
	MQTTHub *Client
	// GroupID stands for group id
	GroupID string
	// ConnectedTopic to send connect event
	ConnectedTopic = "$hw/events/connected/%s"
	// DisconnectedTopic to send disconnect event
	DisconnectedTopic = "$hw/events/disconnected/%s"
	// MemberGet to get membership device
	MemberGet = "$hw/events/edgeGroup/%s/membership/get"
	// MemberGetRes to get membership device
	MemberGetRes = "$hw/events/edgeGroup/%s/membership/get/result"
	// MemberDetail which edge-client should be pub when service start
	MemberDetail = "$hw/events/edgeGroup/%s/membership/detail"
	// MemberDetailRes MemberDetail topic resp
	MemberDetailRes = "$hw/events/edgeGroup/%s/membership/detail/result"
	// MemberUpdate updating of the twin
	MemberUpdate = "$hw/events/edgeGroup/%s/membership/updated"
	// GroupUpdate updates a edgegroup
	GroupUpdate = "$hw/events/edgeGroup/%s/updated"
	// GroupAuthGet get temperary aksk from cloudhub
	GroupAuthGet = "$hw/events/edgeGroup/%s/authInfo/get"
	// GroupAuthGetRes temperary aksk from cloudhub
	GroupAuthGetRes = "$hw/events/edgeGroup/%s/authInfo/get/result"
	// SubTopics which edge-client should be sub
	SubTopics = []string{
		"$hw/events/upload/#",
		"$hw/events/device/+/state/update",
		"$hw/events/device/+/twin/+",
		"$hw/events/node/+/membership/get",
		UploadTopic,
	}
)

// Client struct
type Client struct {
	MQTTUrl string
	PubCli  MQTT.Client
	SubCli  MQTT.Client
}

// AccessInfo that deliver between edge-hub and cloud-hub
type AccessInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Topic   string `json:"topic"`
	Content []byte `json:"content"`
}

func onPubConnectionLost(client MQTT.Client, err error) {
	klog.Errorf("onPubConnectionLost with error: %v", err)
	go MQTTHub.InitPubClient()
}

func onSubConnectionLost(client MQTT.Client, err error) {
	klog.Errorf("onSubConnectionLost with error: %v", err)
	go MQTTHub.InitSubClient()
}

func onSubConnect(client MQTT.Client) {
	for _, t := range SubTopics {
		token := client.Subscribe(t, 1, OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			klog.Errorf("edge-hub-cli subscribe topic: %s, %v", t, err)
			return
		}
		klog.Infof("edge-hub-cli subscribe topic to %s", t)
	}
}

// OnSubMessageReceived msg received callback
func OnSubMessageReceived(client MQTT.Client, message MQTT.Message) {
	klog.Infof("OnSubMessageReceived receive msg from topic: %s", message.Topic())
	// for "$hw/events/device/+/twin/+", "$hw/events/node/+/membership/get", send to twin
	// for other, send to hub
	// for "SYS/dis/upload_records", no need to base64 topic
	var target string
	resource := base64.URLEncoding.EncodeToString([]byte(message.Topic()))
	if strings.HasPrefix(message.Topic(), "$hw/events/device") || strings.HasPrefix(message.Topic(), "$hw/events/node") {
		target = modules.TwinGroup
	} else {
		target = modules.HubGroup
		if message.Topic() == UploadTopic {
			resource = UploadTopic
		}
	}
	// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
	msg := model.NewMessage("").BuildRouter(modules.BusGroup, "user",
		resource, "response").FillBody(string(message.Payload()))
	klog.Info(fmt.Sprintf("received msg from mqttserver, deliver to %s with resource %s", target, resource))
	beehiveContext.SendToGroup(target, *msg)
}

// InitSubClient init sub client
func (mq *Client) InitSubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}
	subID := fmt.Sprintf("hub-client-sub-%s", timeStr[0:right])
	subOpts := util.HubClientInit(mq.MQTTUrl, subID, "", "")
	subOpts.OnConnect = onSubConnect
	subOpts.AutoReconnect = false
	subOpts.OnConnectionLost = onSubConnectionLost
	mq.SubCli = MQTT.NewClient(subOpts)
	util.LoopConnect(subID, mq.SubCli)
	klog.Info("finish hub-client sub")
}

// InitPubClient init pub client
func (mq *Client) InitPubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}
	pubID := fmt.Sprintf("hub-client-pub-%s", timeStr[0:right])
	pubOpts := util.HubClientInit(mq.MQTTUrl, pubID, "", "")
	pubOpts.OnConnectionLost = onPubConnectionLost
	pubOpts.AutoReconnect = false
	mq.PubCli = MQTT.NewClient(pubOpts)
	util.LoopConnect(pubID, mq.PubCli)
	klog.Info("finish hub-client pub")
}
