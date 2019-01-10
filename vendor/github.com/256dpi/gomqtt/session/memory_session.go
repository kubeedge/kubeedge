// Package session implements session objects to be used with MQTT clients and
// brokers.
package session

import (
	"github.com/256dpi/gomqtt/packet"
)

// Direction denotes a packets direction.
type Direction int

const (
	// Incoming packets are being received.
	Incoming Direction = iota

	// Outgoing packets are being be sent.
	Outgoing
)

// A MemorySession stores packets in memory.
type MemorySession struct {
	Counter  *IDCounter
	Incoming *PacketStore
	Outgoing *PacketStore
}

// NewMemorySession returns a new MemorySession.
func NewMemorySession() *MemorySession {
	return &MemorySession{
		Counter:  NewIDCounter(),
		Incoming: NewPacketStore(),
		Outgoing: NewPacketStore(),
	}
}

// NextID will return the next id for outgoing packets.
func (s *MemorySession) NextID() packet.ID {
	return s.Counter.NextID()
}

// SavePacket will store a packet in the session. An eventual existing
// packet with the same id gets quietly overwritten.
func (s *MemorySession) SavePacket(dir Direction, pkt packet.Generic) error {
	s.storeForDirection(dir).Save(pkt)
	return nil
}

// LookupPacket will retrieve a packet from the session using a packet id.
func (s *MemorySession) LookupPacket(dir Direction, id packet.ID) (packet.Generic, error) {
	return s.storeForDirection(dir).Lookup(id), nil
}

// DeletePacket will remove a packet from the session. The method must not
// return an error if no packet with the specified id does exists.
func (s *MemorySession) DeletePacket(dir Direction, id packet.ID) error {
	s.storeForDirection(dir).Delete(id)
	return nil
}

// AllPackets will return all packets currently saved in the session.
func (s *MemorySession) AllPackets(dir Direction) ([]packet.Generic, error) {
	return s.storeForDirection(dir).All(), nil
}

// Reset will completely reset the session.
func (s *MemorySession) Reset() error {
	// reset counter and stores
	s.Counter.Reset()
	s.Incoming.Reset()
	s.Outgoing.Reset()

	return nil
}

func (s *MemorySession) storeForDirection(dir Direction) *PacketStore {
	if dir == Incoming {
		return s.Incoming
	} else if dir == Outgoing {
		return s.Outgoing
	}

	panic("unknown direction")
}
