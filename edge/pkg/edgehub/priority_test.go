package edgehub

import (
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestPriorityQueue(t *testing.T) {
	// Create a new priority queue
	pq := NewMessagePriorityQueue()

	// Create test messages with different priorities
	lowMsg := model.NewMessage("").BuildRouter("test", "group", "resource", "operation").FillBody("low priority")
	normalMsg := model.NewMessage("").BuildRouter("metamanager", "group", "resource", "operation").FillBody("normal priority")
	importantMsg := model.NewMessage("").BuildRouter("eventbus", "group", "resource", "operation").FillBody("important priority")
	emergencyMsg := model.NewMessage("").BuildRouter(modules.EdgeHubModuleName, "resource", "node", message.OperationKeepalive).FillBody("emergency priority")

	// Add messages to queue in random order
	pq.Push(lowMsg, PriorityLow)
	pq.Push(emergencyMsg, PriorityEmergency)
	pq.Push(importantMsg, PriorityImportant)
	pq.Push(normalMsg, PriorityNormal)

	// Verify queue size
	if pq.Size() != 4 {
		t.Errorf("Expected queue size 4, got %d", pq.Size())
	}

	// Pop messages and verify they come out in priority order
	expectedOrder := []int{PriorityEmergency, PriorityImportant, PriorityNormal, PriorityLow}

	for i, expectedPriority := range expectedOrder {
		msg := pq.Pop()
		if msg == nil {
			t.Errorf("Expected message at index %d, got nil", i)
			continue
		}

		actualPriority := GetPriorityForMessage(msg)
		if actualPriority != expectedPriority {
			t.Errorf("Expected priority %d at index %d, got %d", expectedPriority, i, actualPriority)
		}
	}

	// Verify queue is empty
	if !pq.IsEmpty() {
		t.Errorf("Expected empty queue, got size %d", pq.Size())
	}
}

func TestGetPriorityForMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  *model.Message
		expected int
	}{
		{
			name:     "Heartbeat message should be emergency",
			message:  model.NewMessage("").BuildRouter(modules.EdgeHubModuleName, "resource", "node", message.OperationKeepalive),
			expected: PriorityEmergency,
		},
		{
			name:     "Eventbus message should be important",
			message:  model.NewMessage("").BuildRouter("eventbus", "group", "resource", "operation"),
			expected: PriorityImportant,
		},
		{
			name:     "Metamanager message should be normal",
			message:  model.NewMessage("").BuildRouter("metamanager", "group", "resource", "operation"),
			expected: PriorityNormal,
		},
		{
			name:     "Unknown source should be low",
			message:  model.NewMessage("").BuildRouter("unknown", "group", "resource", "operation"),
			expected: PriorityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := GetPriorityForMessage(tt.message)
			if priority != tt.expected {
				t.Errorf("Expected priority %d, got %d", tt.expected, priority)
			}
		})
	}
}

func TestPriorityQueueConcurrency(t *testing.T) {
	pq := NewMessagePriorityQueue()

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Add messages
	go func() {
		for i := 0; i < 100; i++ {
			msg := model.NewMessage("").BuildRouter("test", "group", "resource", "operation").FillBody("test")
			pq.Push(msg, i%4) // Different priorities
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Pop messages
	go func() {
		for i := 0; i < 100; i++ {
			msg := pq.Pop()
			if msg != nil {
				// Just consume the message
			}
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Queue should be empty or have very few items
	if pq.Size() > 10 {
		t.Errorf("Expected queue to be mostly empty, got size %d", pq.Size())
	}
}
