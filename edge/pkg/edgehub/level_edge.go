package edgehub

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/klog/v2"
	"math"

	"github.com/kubeedge/beehive/pkg/core"
)

const inf = math.MaxInt32 // 使用32位的最大整数表示无穷大

type EdgeConfig struct {
	Cloud struct {
		IP          string `yaml:"ip"`
		SignalRange int    `yaml:"signal_range"`
	} `yaml:"cloud"`
	Nodes []struct {
		IP              string `yaml:"ip"`
		DistanceToCloud int    `yaml:"distance_to_cloud"`
	} `yaml:"nodes"`
	Distances []struct {
		Node1    string `yaml:"node1"`
		Node2    string `yaml:"node2"`
		Distance int    `yaml:"distance"`
	} `yaml:"distances"`
}

type LevelMessage struct {
	IP    string `json:"ip"`
	Level int    `json:"level"`
}

// 读取配置文件
func loadEdgeConfig(path string) EdgeConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		klog.Fatalf("Failed to read config file: %v", err)
	}

	var config EdgeConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		//配置文件解析错误
		klog.Fatalf("配Failed to unmarshal config file: %v", err)
	}

	return config
}

// 初始化节点层级
func initializeNodeLevels(config EdgeConfig) map[string]int {
	nodeLevels := make(map[string]int)
	nodeLevels[config.Cloud.IP] = 0 // 云节点层级为0

	// 初始化边缘节点层级为无穷大
	for _, node := range config.Nodes {
		nodeLevels[node.IP] = inf
	}

	return nodeLevels
}

// 广播和更新层级
func broadcastAndUpdateLevels(config EdgeConfig, nodeLevels map[string]int) {
	stable := false
	for !stable {
		stable = true
		// 遍历每个节点并更新层级
		for _, node := range config.Nodes {
			currentLevel := nodeLevels[node.IP]
			updated := false

			// 检查与云节点的距离
			if node.DistanceToCloud <= config.Cloud.SignalRange {
				newLevel := nodeLevels[config.Cloud.IP] + 1
				if newLevel < currentLevel {
					nodeLevels[node.IP] = newLevel
					klog.Infof("Sent level message to neighbor: %s, {IP:%s Level:%d}", config.Cloud.IP, node.IP, newLevel)
					updated = true
				}
			}

			// 检查与其他节点的距离并更新
			for _, dist := range config.Distances {
				if dist.Node1 == node.IP || dist.Node2 == node.IP {
					var neighborIP string
					if dist.Node1 == node.IP {
						neighborIP = dist.Node2
					} else {
						neighborIP = dist.Node1
					}
					neighborLevel := nodeLevels[neighborIP]
					if neighborLevel != inf { // 邻居节点的层级已经确定
						newLevel := neighborLevel + 1
						if newLevel < currentLevel && dist.Distance <= config.Cloud.SignalRange {
							nodeLevels[node.IP] = newLevel
							klog.Infof("Sent level message to neighbor: %s, {IP:%s Level:%d}", neighborIP, node.IP, newLevel)
							updated = true
						}
					}
				}
			}
			if updated {
				stable = false
			}
		}
	}
}

// EdgeHubModule 是 EdgeHub 的模块定义，实现 Module 接口
type EdgeHubModule struct {
	edgeIP     string
	finalLevel int // 节点最终的层级
}

// Name 返回模块名
func (eh *EdgeHubModule) Name() string {
	return "edgehub"
}

// Group 返回模块组名
func (eh *EdgeHubModule) Group() string {
	return "hub"
}

// Enable 表示模块是否启用
func (eh *EdgeHubModule) Enable() bool {
	return true
}

// Start 启动 EdgeHub 模块，运行层级确认方法
func (eh *EdgeHubModule) Start() {
	klog.Infof("Starting EdgeHub for edge: %s", eh.edgeIP)

	// 加载配置文件
	config := loadEdgeConfig("config.yaml")

	// 初始化层级
	nodeLevels := initializeNodeLevels(config)

	// 广播并更新层级
	broadcastAndUpdateLevels(config, nodeLevels)

	// 输出最终层级信息
	klog.Infof("最终层级信息：")
	klog.Infof("Cloud (IP: %s): %d级", config.Cloud.IP, nodeLevels[config.Cloud.IP])
	for _, node := range config.Nodes {
		klog.Infof("Node (IP: %s): %d级", node.IP, nodeLevels[node.IP])
	}
}

// 注册 EdgeHub 模块
func RegisterEdgeHub(edgeIP string) {
	core.Register(&EdgeHubModule{edgeIP: edgeIP})
}
