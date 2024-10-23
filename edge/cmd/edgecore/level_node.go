package main

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/klog/v2"
)

type EdgeConfig struct {
	Cloud struct {
		IP          string `yaml:"ip"`
		SignalRange int    `yaml:"signal_range"`
	} `yaml:"cloud"`
	Nodes []struct {
		IP              string `yaml:"ip"`
		DistanceToCloud int    `yaml:"distance_to_cloud"`
	} `yaml:"nodes"`
}

// 读取配置文件
func loadEdgeConfig(path string) EdgeConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		klog.Fatalf("无法读取配置文件: %v", err)
	}

	var config EdgeConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		klog.Fatalf("配置文件解析错误: %v", err)
	}

	return config
}

func main() {
	// 加载配置文件
	config := loadEdgeConfig("config.yaml")

	// 为每个边缘节点注册 EdgeHub 模块
	for _, node := range config.Nodes {
		edgehub.RegisterEdgeHub(node.IP) // 注册每个边缘节点
	}

	// 启动 Beehive 框架
	core.Run()
}
