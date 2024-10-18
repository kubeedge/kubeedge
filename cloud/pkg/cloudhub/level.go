package cloudhub

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
)

type CloudConfig struct {
	Cloud struct {
		IP          string `yaml:"ip"`
		SignalRange int    `yaml:"signal_range"`
	} `yaml:"cloud"`
}

type LevelMessage struct {
	IP    string `json:"ip"`
	Level int    `json:"level"`
}

// 读取配置文件
func loadCloudConfig() CloudConfig {
	data, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		klog.Fatalf("Failed to read config file: %v", err)
	}

	var config CloudConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		klog.Fatalf("Failed to unmarshal config file: %v", err)
	}

	return config
}

// CloudHubModule 是 CloudHub 的模块定义，实现 Module 接口
type CloudHubModule struct{}

// Name 返回模块名
func (ch *CloudHubModule) Name() string {
	return "cloudhub"
}

// Group 返回模块组名
func (ch *CloudHubModule) Group() string {
	return "hub"
}

// Enable 表示模块是否启用
func (ch *CloudHubModule) Enable() bool {
	return true
}

// Start 启动 CloudHub 模块
func (ch *CloudHubModule) Start() {
	klog.Infof("Starting CloudHub...")

	// 加载配置
	config := loadCloudConfig()

	// 云节点IP和初始层级
	cloudIP := config.Cloud.IP
	cloudLevel := 0

	// 定时广播云节点的层级信息
	go func() {
		for {
			time.Sleep(10 * time.Second)

			// 构建层级消息
			msg := LevelMessage{
				IP:    cloudIP,
				Level: cloudLevel,
			}

			// 发送广播给所有边缘节点
			broadcastLevelMessage(msg)
		}
	}()
}

// 发送层级消息给边缘节点
func broadcastLevelMessage(msg LevelMessage) {
	messageBody, err := json.Marshal(msg)
	if err != nil {
		klog.Errorf("Failed to marshal level message: %v", err)
		return
	}

	// 使用 BuildBody 设置消息内容
	cloudHubMessage := model.NewMessage("").FillBody(messageBody)

	// 通过 EdgeController 发送消息
	beehiveContext.Send("edgehub", *cloudHubMessage)
	klog.Infof("Broadcasted level message to edges: %+v", msg)
}

// 注册 CloudHub 模块
func RegisterCloudHub() {
	core.Register(&CloudHubModule{})
}
