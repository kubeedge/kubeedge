package mqtt

import (
	"fmt"
	"strconv"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/dao"
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
		"+/user/#",
	}
)

// Client struct
type Client struct {
	MQTTUrl     string
	PubClientID string
	SubClientID string
	Username    string
	Password    string
	PubCli      MQTT.Client
	SubCli      MQTT.Client
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
	topics, err := dao.QueryAllTopics()
	if err != nil {
		klog.Errorf("list edge-hub-cli-topics failed: %v", err)
		return
	}
	if len(*topics) <= 0 {
		klog.Infof("list edge-hub-cli-topics status, no record, skip sync")
		return
	}
	for _, t := range *topics {
		token := client.Subscribe(t, 1, OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			klog.Errorf("edge-hub-cli subscribe topic: %s, %v", t, err)
			return
		}
		klog.Infof("edge-hub-cli subscribe topic to %s", t)
	}
}

// OnSubMessageReceived msg received callback
func OnSubMessageReceived(client MQTT.Client, msg MQTT.Message) {
	klog.Infof("OnSubMessageReceived receive msg from topic: %s", msg.Topic())

	NewMessageMux().Dispatch(msg.Topic(), msg.Payload())
}

// InitSubClient init sub client
func (mq *Client) InitSubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}
	// if SubClientID is NOT set, we need to generate it by ourselves.
	if mq.SubClientID == "" {
		mq.SubClientID = fmt.Sprintf("hub-client-sub-%s", timeStr[0:right])
	}
	subOpts := util.HubClientInit(mq.MQTTUrl, mq.SubClientID, mq.Username, mq.Password)
	subOpts.OnConnect = onSubConnect
	subOpts.AutoReconnect = false
	subOpts.OnConnectionLost = onSubConnectionLost
	mq.SubCli = MQTT.NewClient(subOpts)
	util.LoopConnect(mq.SubClientID, mq.SubCli)
	klog.Info("finish hub-client sub")
}

// InitPubClient init pub client
func (mq *Client) InitPubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}
	// if PubClientID is NOT set, we need to generate it by ourselves.
	if mq.PubClientID == "" {
		mq.PubClientID = fmt.Sprintf("hub-client-pub-%s", timeStr[0:right])
	}
	pubOpts := util.HubClientInit(mq.MQTTUrl, mq.PubClientID, mq.Username, mq.Password)
	pubOpts.OnConnectionLost = onPubConnectionLost
	pubOpts.AutoReconnect = false
	mq.PubCli = MQTT.NewClient(pubOpts)
	util.LoopConnect(mq.PubClientID, mq.PubCli)
	klog.Info("finish hub-client pub")
}
