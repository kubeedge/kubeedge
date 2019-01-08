package packet

import "fmt"

// A Message bundles data that is published between brokers and clients.
type Message struct {
	// The Topic of the message.
	Topic string

	// The Payload of the message.
	Payload []byte

	// The QOS indicates the level of assurance for delivery.
	QOS QOS

	// If the Retain flag is set to true, the server must store the message,
	// so that it can be delivered to future subscribers whose subscriptions
	// match its topic name.
	Retain bool
}

// String returns a string representation of the message.
func (m *Message) String() string {
	return fmt.Sprintf("<Message Topic=%q QOS=%d Retain=%t Payload=%v>",
		m.Topic, m.QOS, m.Retain, m.Payload)
}

// Copy returns a copy of the message.
func (m Message) Copy() *Message {
	return &m
}
