package session

import (
	"sync"

	"github.com/256dpi/gomqtt/packet"
)

// An IDCounter continuously counts packet ids.
type IDCounter struct {
	next  packet.ID
	mutex sync.Mutex
}

// NewIDCounter returns a new counter.
func NewIDCounter() *IDCounter {
	return NewIDCounterWithNext(1)
}

// NewIDCounterWithNext returns a new counter that will emit the specified if
// id as the next id.
func NewIDCounterWithNext(next packet.ID) *IDCounter {
	return &IDCounter{
		next: next,
	}
}

// NextID will return the next id.
func (c *IDCounter) NextID() packet.ID {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// ignore zeroes
	if c.next == 0 {
		c.next++
	}

	// cache next id
	id := c.next

	// increment id
	c.next++

	return id
}

// Reset will reset the counter.
func (c *IDCounter) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.next = 1
}
