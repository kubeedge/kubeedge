package packet

import (
	"encoding/binary"
	"fmt"
)

// A Publish packet is sent from a client to a server or from server to a client
// to transport an application message.
type Publish struct {
	// The message to publish.
	Message Message

	// If the Dup flag is set to false, it indicates that this is the first
	// occasion that the client or server has attempted to send this
	// Publish packet. If the dup flag is set to true, it indicates that this
	// might be re-delivery of an earlier attempt to send the packet.
	Dup bool

	// The packet identifier.
	ID ID
}

// NewPublish creates a new Publish packet.
func NewPublish() *Publish {
	return &Publish{}
}

// Type returns the packets type.
func (pp *Publish) Type() Type {
	return PUBLISH
}

// String returns a string representation of the packet.
func (pp *Publish) String() string {
	return fmt.Sprintf("<Publish ID=%d Message=%s Dup=%t>",
		pp.ID, pp.Message.String(), pp.Dup)
}

// Len returns the byte length of the encoded packet.
func (pp *Publish) Len() int {
	ml := pp.len()
	return headerLen(ml) + ml
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Publish) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, flags, rl, err := headerDecode(src[total:], PUBLISH)
	total += hl
	if err != nil {
		return total, err
	}

	// read flags
	pp.Dup = ((flags >> 3) & 0x1) == 1
	pp.Message.Retain = (flags & 0x1) == 1
	pp.Message.QOS = QOS((flags >> 1) & 0x3)

	// check qos
	if !pp.Message.QOS.Successful() {
		return total, makeError(pp.Type(), "invalid QOS level (%d)", pp.Message.QOS)
	}

	// check buffer length
	if len(src) < total+2 {
		return total, makeError(pp.Type(), "insufficient buffer size, expected %d, got %d", total+2, len(src))
	}

	n := 0

	// read topic
	pp.Message.Topic, n, err = readLPString(src[total:], pp.Type())
	total += n
	if err != nil {
		return total, err
	}

	if pp.Message.QOS != 0 {
		// check buffer length
		if len(src) < total+2 {
			return total, makeError(pp.Type(), "insufficient buffer size, expected %d, got %d", total+2, len(src))
		}

		// read packet id
		pp.ID = ID(binary.BigEndian.Uint16(src[total:]))
		total += 2

		// check packet id
		if !pp.ID.Valid() {
			return total, makeError(pp.Type(), "packet id must be grater than zero")
		}
	}

	// calculate payload length
	l := int(rl) - (total - hl)

	// read payload
	if l > 0 {
		pp.Message.Payload = make([]byte, l)
		copy(pp.Message.Payload, src[total:total+l])
		total += len(pp.Message.Payload)
	}

	return total, nil
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Publish) Encode(dst []byte) (int, error) {
	total := 0

	// check topic length
	if len(pp.Message.Topic) == 0 {
		return total, makeError(pp.Type(), "topic name is empty")
	}

	flags := byte(0)

	// set dup flag
	if pp.Dup {
		flags |= 0x8 // 00001000
	} else {
		flags &= 247 // 11110111
	}

	// set retain flag
	if pp.Message.Retain {
		flags |= 0x1 // 00000001
	} else {
		flags &= 254 // 11111110
	}

	// check qos
	if !pp.Message.QOS.Successful() {
		return 0, makeError(pp.Type(), "invalid QOS level %d", pp.Message.QOS)
	}

	// check packet id
	if pp.Message.QOS > 0 && !pp.ID.Valid() {
		return total, makeError(pp.Type(), "packet id must be grater than zero")
	}

	// set qos
	flags = (flags & 249) | (byte(pp.Message.QOS) << 1) // 249 = 11111001

	// encode header
	n, err := headerEncode(dst[total:], flags, pp.len(), pp.Len(), PUBLISH)
	total += n
	if err != nil {
		return total, err
	}

	// write topic
	n, err = writeLPString(dst[total:], pp.Message.Topic, pp.Type())
	total += n
	if err != nil {
		return total, err
	}

	// write packet id
	if pp.Message.QOS != 0 {
		binary.BigEndian.PutUint16(dst[total:], uint16(pp.ID))
		total += 2
	}

	// write payload
	copy(dst[total:], pp.Message.Payload)
	total += len(pp.Message.Payload)

	return total, nil
}

// Returns the payload length.
func (pp *Publish) len() int {
	total := 2 + len(pp.Message.Topic) + len(pp.Message.Payload)
	if pp.Message.QOS != 0 {
		total += 2
	}

	return total
}
