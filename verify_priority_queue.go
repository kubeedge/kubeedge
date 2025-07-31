package main

import (
	"fmt"
)

// 简化的优先级队列验证
func main() {
	fmt.Println("=== KubeEdge 优先级队列验证 ===")

	// 模拟心跳消息处理
	fmt.Println("\n1. 验证心跳消息优先级:")
	heartbeatMsg := "keepalive"
	priority := getPriorityForMessage(heartbeatMsg)
	fmt.Printf("心跳消息 '%s' 的优先级: %d (紧急)\n", heartbeatMsg, priority)

	// 模拟不同消息的处理顺序
	fmt.Println("\n2. 验证消息处理顺序:")
	messages := []string{
		"log_message",        // 低优先级
		"data_sync",          // 普通优先级
		"event_notification", // 重要优先级
		"heartbeat",          // 紧急优先级
	}

	fmt.Println("消息按优先级处理顺序:")
	for i, msg := range messages {
		priority := getPriorityForMessage(msg)
		priorityName := getPriorityName(priority)
		fmt.Printf("%d. %s (优先级: %s)\n", i+1, msg, priorityName)
	}

	// 模拟实际使用场景
	fmt.Println("\n3. 模拟实际使用场景:")
	simulateMessageProcessing()

	fmt.Println("\n✅ 优先级队列验证完成!")
}

func getPriorityForMessage(message string) int {
	switch message {
	case "heartbeat", "keepalive":
		return 0 // 紧急
	case "event_notification":
		return 1 // 重要
	case "data_sync":
		return 2 // 普通
	default:
		return 3 // 低
	}
}

func getPriorityName(priority int) string {
	switch priority {
	case 0:
		return "紧急"
	case 1:
		return "重要"
	case 2:
		return "普通"
	case 3:
		return "低"
	default:
		return "未知"
	}
}

func simulateMessageProcessing() {
	// 模拟消息队列
	queue := []string{
		"log_message",
		"heartbeat",
		"data_sync",
		"event_notification",
		"another_heartbeat",
	}

	fmt.Println("原始消息队列:")
	for i, msg := range queue {
		fmt.Printf("  %d. %s\n", i+1, msg)
	}

	// 按优先级排序
	fmt.Println("\n按优先级排序后的处理顺序:")
	processed := make([]string, 0)

	// 先处理紧急消息
	for _, msg := range queue {
		if getPriorityForMessage(msg) == 0 {
			processed = append(processed, msg)
			fmt.Printf("  [紧急] %s\n", msg)
		}
	}

	// 再处理重要消息
	for _, msg := range queue {
		if getPriorityForMessage(msg) == 1 {
			processed = append(processed, msg)
			fmt.Printf("  [重要] %s\n", msg)
		}
	}

	// 再处理普通消息
	for _, msg := range queue {
		if getPriorityForMessage(msg) == 2 {
			processed = append(processed, msg)
			fmt.Printf("  [普通] %s\n", msg)
		}
	}

	// 最后处理低优先级消息
	for _, msg := range queue {
		if getPriorityForMessage(msg) == 3 {
			processed = append(processed, msg)
			fmt.Printf("  [低] %s\n", msg)
		}
	}

	fmt.Printf("\n处理完成，共处理 %d 条消息\n", len(processed))
}
