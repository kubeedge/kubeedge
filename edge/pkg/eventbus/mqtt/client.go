package mqtt

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
)

// disconnectQuiesce is the milliseconds paho waits to finish in-flight work
// when disconnecting a client that is being replaced on reconnect.
const disconnectQuiesce = 250

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
	// GroupAuthGet get temporary aksk from cloudhub
	GroupAuthGet = "$hw/events/edgeGroup/%s/authInfo/get"
	// GroupAuthGetRes temporary aksk from cloudhub
	GroupAuthGetRes = "$hw/events/edgeGroup/%s/authInfo/get/result"
	// SubTopics which edge-client should be sub
	SubTopics = []string{
		"$hw/events/upload/#",
		"$hw/events/device/+/+/state/update",
		"$hw/events/device/+/+/twin/+",
		"$hw/events/node/+/membership/get",
		UploadTopic,
		"+/user/#",
	}

	// EventBusServiceFactory is a function variable that can be mocked in tests
	EventBusServiceFactory = func() interface {
		InsertTopics(topic string) error
		DeleteTopicsByKey(key string) error
		QueryAllTopics() (*[]string, error)
	} {
		return dbclient.NewEventBusService()
	}
)

// Client struct
type Client struct {
	MQTTUrl     string
	PubClientID string
	SubClientID string
	Username    string
	Password    string

	// cliLock guards pubCli and subCli, which are reassigned from the
	// connection-lost callbacks while the eventbus loop uses them.
	cliLock sync.RWMutex
	pubCli  MQTT.Client
	subCli  MQTT.Client

	// pubInitLock and subInitLock serialize the full reconnect lifecycle
	// (build, swap, disconnect the replaced client, connect) so concurrent
	// connection-lost callbacks cannot interleave and reconnect a client that
	// has already been replaced.
	pubInitLock sync.Mutex
	subInitLock sync.Mutex
}

// Publish publishes on the current publish client while holding the read lock,
// so the reconnect path cannot replace and disconnect the client during the
// call.
func (mq *Client) Publish(topic string, qos byte, retained bool, payload interface{}) MQTT.Token {
	mq.cliLock.RLock()
	defer mq.cliLock.RUnlock()
	return mq.pubCli.Publish(topic, qos, retained, payload)
}

// Subscribe subscribes on the current subscribe client while holding the read
// lock; see Publish for the rationale.
func (mq *Client) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) MQTT.Token {
	mq.cliLock.RLock()
	defer mq.cliLock.RUnlock()
	return mq.subCli.Subscribe(topic, qos, callback)
}

// Unsubscribe unsubscribes on the current subscribe client while holding the
// read lock; see Publish for the rationale.
func (mq *Client) Unsubscribe(topics ...string) MQTT.Token {
	mq.cliLock.RLock()
	defer mq.cliLock.RUnlock()
	return mq.subCli.Unsubscribe(topics...)
}

// isActivePubClient reports whether client is the current publish client.
func (mq *Client) isActivePubClient(client MQTT.Client) bool {
	mq.cliLock.RLock()
	defer mq.cliLock.RUnlock()
	return mq.pubCli == client
}

// isActiveSubClient reports whether client is the current subscribe client.
func (mq *Client) isActiveSubClient(client MQTT.Client) bool {
	mq.cliLock.RLock()
	defer mq.cliLock.RUnlock()
	return mq.subCli == client
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
	// Ignore the event when it comes from a client that has already been
	// replaced; otherwise a superseded client would reconnect and displace the
	// current active one, starting an extra connection loop.
	if MQTTHub == nil || !MQTTHub.isActivePubClient(client) {
		klog.Warning("ignore connection lost event from a superseded pub client")
		return
	}
	go MQTTHub.InitPubClient()
}

func onSubConnectionLost(client MQTT.Client, err error) {
	klog.Errorf("onSubConnectionLost with error: %v", err)
	// Ignore the event when it comes from a client that has already been
	// replaced; otherwise a superseded client would reconnect and displace the
	// current active one, starting an extra connection loop.
	if MQTTHub == nil || !MQTTHub.isActiveSubClient(client) {
		klog.Warning("ignore connection lost event from a superseded sub client")
		return
	}
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
	topics, err := EventBusServiceFactory().QueryAllTopics()
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
func OnSubMessageReceived(_ MQTT.Client, msg MQTT.Message) {
	klog.Infof("OnSubMessageReceived receive msg from topic: %s", msg.Topic())

	NewMessageMux().Dispatch(msg.Topic(), msg.Payload())
}

// InitSubClient init sub client
func (mq *Client) InitSubClient() {
	// Serialize the whole lifecycle so concurrent connection-lost callbacks
	// cannot interleave and reconnect a client that has already been replaced.
	mq.subInitLock.Lock()
	defer mq.subInitLock.Unlock()
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
	cli := MQTT.NewClient(subOpts)

	mq.cliLock.Lock()
	old := mq.subCli
	mq.subCli = cli
	mq.cliLock.Unlock()
	// release the client being replaced, otherwise its goroutines leak on every
	// broker reconnect.
	if old != nil {
		old.Disconnect(disconnectQuiesce)
	}

	util.LoopConnect(mq.SubClientID, cli)
	klog.Info("finish hub-client sub")
}

// InitPubClient init pub client
func (mq *Client) InitPubClient() {
	// Serialize the whole lifecycle so concurrent connection-lost callbacks
	// cannot interleave and reconnect a client that has already been replaced.
	mq.pubInitLock.Lock()
	defer mq.pubInitLock.Unlock()
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
	cli := MQTT.NewClient(pubOpts)

	mq.cliLock.Lock()
	old := mq.pubCli
	mq.pubCli = cli
	mq.cliLock.Unlock()
	// release the client being replaced, otherwise its goroutines leak on every
	// broker reconnect.
	if old != nil {
		old.Disconnect(disconnectQuiesce)
	}

	util.LoopConnect(mq.PubClientID, cli)
	klog.Info("finish hub-client pub")
}
