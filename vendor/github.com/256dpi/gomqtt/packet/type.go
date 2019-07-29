package packet

import "errors"

// ErrInvalidPacketType is returned by New if the packet type is invalid.
var ErrInvalidPacketType = errors.New("invalid packet type")

// Type represents the MQTT packet types.
type Type byte

// All packet types.
const (
	_ Type = iota
	CONNECT
	CONNACK
	PUBLISH
	PUBACK
	PUBREC
	PUBREL
	PUBCOMP
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ
	PINGRESP
	DISCONNECT
)

// String returns the type as a string.
func (t Type) String() string {
	switch t {
	case CONNECT:
		return "Connect"
	case CONNACK:
		return "Connack"
	case PUBLISH:
		return "Publish"
	case PUBACK:
		return "Puback"
	case PUBREC:
		return "Pubrec"
	case PUBREL:
		return "Pubrel"
	case PUBCOMP:
		return "Pubcomp"
	case SUBSCRIBE:
		return "Subscribe"
	case SUBACK:
		return "Suback"
	case UNSUBSCRIBE:
		return "Unsubscribe"
	case UNSUBACK:
		return "Unsuback"
	case PINGREQ:
		return "Pingreq"
	case PINGRESP:
		return "Pingresp"
	case DISCONNECT:
		return "Disconnect"
	}

	return "Unknown"
}

// DefaultFlags returns the default flag values for the packet type, as defined
// by the MQTT spec, except for PUBLISH.
func (t Type) defaultFlags() byte {
	switch t {
	case CONNECT:
		return 0
	case CONNACK:
		return 0
	case PUBACK:
		return 0
	case PUBREC:
		return 0
	case PUBREL:
		return 2 // 00000010
	case PUBCOMP:
		return 0
	case SUBSCRIBE:
		return 2 // 00000010
	case SUBACK:
		return 0
	case UNSUBSCRIBE:
		return 2 // 00000010
	case UNSUBACK:
		return 0
	case PINGREQ:
		return 0
	case PINGRESP:
		return 0
	case DISCONNECT:
		return 0
	}

	return 0
}

// New creates a new packet based on the type. It is a shortcut to call one of
// the New*Packet functions. An error is returned if the type is invalid.
func (t Type) New() (Generic, error) {
	switch t {
	case CONNECT:
		return NewConnect(), nil
	case CONNACK:
		return NewConnack(), nil
	case PUBLISH:
		return NewPublish(), nil
	case PUBACK:
		return NewPuback(), nil
	case PUBREC:
		return NewPubrec(), nil
	case PUBREL:
		return NewPubrel(), nil
	case PUBCOMP:
		return NewPubcomp(), nil
	case SUBSCRIBE:
		return NewSubscribe(), nil
	case SUBACK:
		return NewSuback(), nil
	case UNSUBSCRIBE:
		return NewUnsubscribe(), nil
	case UNSUBACK:
		return NewUnsuback(), nil
	case PINGREQ:
		return NewPingreq(), nil
	case PINGRESP:
		return NewPingresp(), nil
	case DISCONNECT:
		return NewDisconnect(), nil
	}

	return nil, ErrInvalidPacketType
}

// Valid returns a boolean indicating whether the type is valid or not.
func (t Type) Valid() bool {
	return t >= CONNECT && t <= DISCONNECT
}
