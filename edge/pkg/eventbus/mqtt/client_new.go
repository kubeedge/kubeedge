package mqtt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"

	"beehive/pkg/common/log"
	"beehive/pkg/core"
	"beehive/pkg/core/context"
	"beehive/pkg/core/model"
	"edge-core/pkg/common/alarm"
	"edge-core/pkg/eventbus/app"
	"edge-core/pkg/eventbus/common/util"
)

var (
	// MQTTHub client
	MQTTHub *MQTTClient
	// ModuleContext variable
	ModuleContext *context.Context
	// NodeID stands for node id
	NodeID string
	// GroupID stands for group id
	GroupID string
	// ProjectID stands for project id
	ProjectID string
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
	// Sub apps topic prefix
	TopicAppPrefix = "$hw/events/apps"
	// SubTopics which edge-client should be sub
	SubTopics = []string{
		"$hw/events/upload/#",
		"$hw/events/device/+/twin/+",
		"$hw/events/node/+/membership/get",
		//properties/get topic is used by edge-connector to upload all devicetwins to DIS/APIG by router
		"$hw/devices/+/events/properties/get",
		//nodes/user topic is the custom topic for users to upload their msg to DIS/APIG by router
		"+/nodes/+/user/#",
		"SYS/dis/upload_records",
		"$hw/+/encryptdatas/+/properties/+/decrypt",
		// support edge tsdb
		"$hw/events/tsdb/receive/#",
		// Support alarm
		"$hw/alarm/+/add",
		"$hw/alarm/+/clear",
		// Support app control
		"$hw/events/apps/get",
		"$hw/events/apps/+/restart",
		"$hw/edge/v1/hub/report/#",
		// support system status
		"$hw/edge/v1/monitor/report/sys_status",
		// support atlas alarm
		"$hw/edge/v1/alarm/report/alarm",
		// support for edgegateway
		"$hw/events/reseller/sync/gateway",
		"$hw/events/reseller/sync/virtualservice",
		"$hw/events/reseller/sync/service",
		"$hw/events/reseller/sync/backend",
	}
	// BlackTopics to forbid the topic to edge
	BlackTopics = []*string{&MemberGet, &MemberDetail}
	// RegisterTopics topic
	RegisterTopics = make(map[string]string)
)

// MQTTClient struct
type MQTTClient struct {
	MQTTUrl string
	PubCli  MQTT.Client
	SubCli  MQTT.Client
}

// AccessInfo that deliever between edge-hub and cloud-hub
type AccessInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Topic   string `json:"topic"`
	Content []byte `json:"content"`
}

// TopicInit init topic
func TopicInit() {
	GroupID = os.Getenv("GROUP_ID")
	NodeID = os.Getenv("NODE_ID")
	if GroupID == "" || NodeID == "" {
		panic("env variables GROUP_ID and NODE_ID shouldn't be null")
	}
	log.LOGGER.Infof("groupID: %s, node id: %s", GroupID, NodeID)
	ConnectedTopic = fmt.Sprintf(ConnectedTopic, NodeID)
	DisconnectedTopic = fmt.Sprintf(DisconnectedTopic, NodeID)
	MemberGet = fmt.Sprintf(MemberGet, GroupID)
	MemberGetRes = fmt.Sprintf(MemberGetRes, GroupID)
	MemberDetail = fmt.Sprintf(MemberDetail, GroupID)
	MemberDetailRes = fmt.Sprintf(MemberDetailRes, GroupID)
	MemberUpdate = fmt.Sprintf(MemberUpdate, GroupID)
	GroupUpdate = fmt.Sprintf(GroupUpdate, GroupID)
	GroupAuthGet = fmt.Sprintf(GroupAuthGet, GroupID)
	GroupAuthGetRes = fmt.Sprintf(GroupAuthGetRes, GroupID)
}

func onPubConnectionLost(client MQTT.Client, err error) {
	log.LOGGER.Errorf("onPubConnectionLost with error! Error:[%v]", err)
	go MQTTHub.InitPubClient()
}

func onSubConnectionLost(client MQTT.Client, err error) {
	log.LOGGER.Errorf("onSubConnectionLost with error, [%v]", err)
	go MQTTHub.InitSubClient()
}

func onSubConnect(client MQTT.Client) {
	for _, t := range SubTopics {
		token := client.Subscribe(t, 1, OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			log.LOGGER.Errorf("edge-hub-cli subscribe topic:%s! Error:[%v]", t, err)
			return
		}
		log.LOGGER.Infof("edge-hub-cli subscribe topic to %s", t)
	}
}

// OnSubMessageReceived msg received callback
func OnSubMessageReceived(client MQTT.Client, message MQTT.Message) {
	log.LOGGER.Infof("OnSubMessageReceived recevie msg from topic: %s", message.Topic())
	onSubMessageReceived(message.Topic(), message.Payload())
}

func onSubMessageReceived(topic string, payload []byte) {
	// for "$hw/events/device/+/twin/+", "$hw/events/node/+/membership/get", send to twin
	// for other, send to hub
	// for "SYS/dis/upload_records", no need to base64 topic
	var target string
	var msg *model.Message
	resource := base64.URLEncoding.EncodeToString([]byte(topic))
	if strings.HasPrefix(topic, "$hw/events/device") || strings.HasPrefix(topic, "$hw/events/node") {
		target = core.TwinGroup
	} else {
		target = core.HubGroup
		if topic == "SYS/dis/upload_records" {
			resource = "SYS/dis/upload_records"
		}
	}

	if strings.HasPrefix(topic, "$hw/events/reseller") {
		handleEdgeGateway(topic, payload)
		return
	}

	segments := strings.Split(topic, "/")

	// for app control topic
	if isApplicationControlMessage(topic) {
		app.HandleMessageForAppsControl(topic, string(payload), ModuleContext)
		return
	}

	// for custom topic {project_id}/nodes/{node_id}/user/{custom_topic}
	if len(segments) > 4 && segments[1] == "nodes" && segments[3] == "user" {
		target = core.HubGroup
		resource = topic
		msg = model.NewMessage("").BuildRouter("eventbus", "user",
			resource, "upload").FillBody(string(payload))
	} else if strings.HasPrefix(topic, "$hw/devices/") && strings.HasSuffix(topic, "/events/properties/get") {
		var defaultMsg model.Message
		err := json.Unmarshal(payload, &defaultMsg)
		if err != nil {
			log.LOGGER.Errorf("unmarshal device get msg failed, err: %v", err)
			return
		}
		msg = &defaultMsg
		//$hw/{project_id}/encryptdatas/{encryptdata_name}/properties/{properties_name}/decrypt
	} else if len(segments) > 6 && segments[0] == "$hw" && segments[2] == "encryptdatas" && segments[4] == "properties" && segments[6] == "decrypt" {
		target = "encryptdata"
		resource = topic
		opr := "query_property"
		msg = model.NewMessage("").BuildRouter("eventbus", "user",
			resource, opr).FillBody(string(payload))
		// $hw/alarm/+/add  $hw/alarm/+/clear
	} else if strings.HasPrefix(topic, "$hw/alarm/") {
		content := alarm.Alarm{}
		err := json.Unmarshal(payload, &content)
		if err != nil {
			log.LOGGER.Errorf("unmarshal alarm msg failed, err: %v", err)
			return
		}
		target = "alarm"
		resource = target
		opr := segments[len(segments)-1]
		msg = model.NewMessage("").BuildRouter(core.BusGroup, target,
			resource, opr).FillBody(content)
	} else if len(segments) == 6 && strings.HasPrefix(topic, "$hw/edge/") && segments[3] == "alarm" && segments[5] == "alarm" {
		util.HandleAlarm(ModuleContext, payload)
		return
	} else if len(segments) >= 6 && segments[1] == "edge" && segments[3] == "hub" && segments[4] == "report" {
		// from IBMA-Edge, $hw/edge/v1/hub/report/+
		target = core.HubGroup
		resource = fmt.Sprintf("%s/%s", ProjectID, strings.Join(segments[5:], "/"))
		msg = model.NewMessage("").BuildRouter("hardware", segments[3],
			resource, model.UpdateOperation).FillBody(string(payload))
	} else if len(segments) >= 6 && segments[1] == "edge" && segments[3] == "monitor" && segments[4] == "report" {
		target = "monitor_remote"
		resource = topic
		content := make(map[string]interface{})
		err := json.Unmarshal(payload, &content)
		if err != nil {
			log.LOGGER.Errorf("unmarshal sys_status msg failed, err: %v", err)
			return
		}
		msg = model.NewMessage("").BuildRouter("eventbus", target, topic, "update").FillBody(content)
	} else {
		// add for edge tsdb
		if strings.HasPrefix(topic, "$hw/events/tsdb/receive/upwrite") {
			msg = model.NewMessage("").BuildRouter("edgedb", "user", NodeID, topic).FillBody(payload)
		} else if strings.HasPrefix(topic, "$hw/events/tsdb/receive/downwrite") {
			msg = model.NewMessage("").BuildRouter("edgedb", "user", NodeID, topic).FillBody(payload)
		} else if strings.HasPrefix(topic, "$hw/events/tsdb/receive/dialupload") {
			msg = model.NewMessage("").BuildRouter("edgedb", "user", NodeID, topic).FillBody(payload)
		} else if strings.HasPrefix(topic, "$hw/events/tsdb/receive/dialdown") {
			msg = model.NewMessage("").BuildRouter("edgedb", "user", NodeID, topic).FillBody(payload)
		} else {
			// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
			msg = model.NewMessage("").BuildRouter(core.BusGroup, "user",
				resource, "response").FillBody(string(payload))
		}
	}
	log.LOGGER.Info(fmt.Sprintf("received msg from mqttserver, deliver to %s with resource %s", target, resource))
	ModuleContext.Send2Group(target, *msg)
}

// InitSubClient init sub client
func (mq *MQTTClient) InitSubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}
	subID := fmt.Sprintf("hub-client-sub-%s", timeStr[0:right])
	subOpts := util.HubclientInit(mq.MQTTUrl, subID, "", "")
	subOpts.OnConnect = onSubConnect
	subOpts.AutoReconnect = false
	subOpts.OnConnectionLost = onSubConnectionLost
	mq.SubCli = MQTT.NewClient(subOpts)
	util.LoopConnect(subID, mq.SubCli)
	log.LOGGER.Info("finish hub-client sub")
}

// InitPubClient init pub client
func (mq *MQTTClient) InitPubClient() {
	timeStr := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	right := len(timeStr)
	if right > 10 {
		right = 10
	}
	pubID := fmt.Sprintf("hub-client-pub-%s", timeStr[0:right])
	pubOpts := util.HubclientInit(mq.MQTTUrl, pubID, "", "")
	pubOpts.OnConnectionLost = onPubConnectionLost
	pubOpts.AutoReconnect = false
	mq.PubCli = MQTT.NewClient(pubOpts)
	util.LoopConnect(pubID, mq.PubCli)
	log.LOGGER.Info("finish hub-client pub")
}

// PubMQTTMsg pub msg to mqtt broker
func (mq *MQTTClient) PubMQTTMsg(topic string, qos byte, retained bool, payload interface{}) error {
	token := mq.PubCli.Publish(topic, qos, retained, payload)
	if token.WaitTimeout(util.TokenWaitTime) && token.Error() != nil {
		log.LOGGER.Errorf("error in pubCloudMsgToEdge with topic: %s! Error:[%v]", topic, token.Error())
		return fmt.Errorf("pubmsg err")
	}
	log.LOGGER.Infof("success in pubCloudMsgToEdge with topic: %s", topic)
	return nil
}

func isApplicationControlMessage(topic string) bool {
	return strings.HasPrefix(topic, TopicAppPrefix)
}

func handleEdgeGateway(topic string, payload []byte) {
	var gateway *[]string
	var sendTopic string
	log.LOGGER.Infof("received edgegateway request from topic: %s", topic)
	switch topic {
	case "$hw/events/reseller/sync/gateway":
		gateway = app.HandleEdgeGatewayRequest("gateway")
		sendTopic = "$hw/events/gateways/insert"
	case "$hw/events/reseller/sync/virtualservice":
		gateway = app.HandleEdgeGatewayRequest("virtualservice")
		sendTopic = "$hw/events/virtualservices/insert"
	case "$hw/events/reseller/sync/service":
		gateway = app.HandleEdgeGatewayRequest("service")
		sendTopic = "$hw/events/services/insert"
	case "$hw/events/reseller/sync/backend":
		gateway = app.HandleEdgeGatewayRequest("backend")
		sendTopic = "$hw/events/backends/insert"
	default:
		log.LOGGER.Errorf("edgegateway topic: %s not support", topic)
		return
	}
	if gateway == nil || len(*gateway) == 0 {
		log.LOGGER.Errorf("edgegateway topic: %s get nil", topic)
		return
	}
	payload, err := json.Marshal(gateway)
	if err != nil {
		log.LOGGER.Errorf("marshal edgegateway msg failed, err: %v", err)
		return
	}
	msg := model.NewMessage("").BuildRouter("gateway", core.BusGroup, sendTopic, "publish").FillBody(payload)
	ModuleContext.Send("eventbus", *msg)
	log.LOGGER.Infof("send edgegateway request from topic: %s success", topic)
}
