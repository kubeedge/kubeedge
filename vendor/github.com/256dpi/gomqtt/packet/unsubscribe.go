package packet

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// An Unsubscribe packet is sent by the client to the server.
type Unsubscribe struct {
	// The topics to unsubscribe from.
	Topics []string

	// The packet identifier.
	ID ID
}

// NewUnsubscribe creates a new Unsubscribe packet.
func NewUnsubscribe() *Unsubscribe {
	return &Unsubscribe{}
}

// Type returns the packets type.
func (up *Unsubscribe) Type() Type {
	return UNSUBSCRIBE
}

// String returns a string representation of the packet.
func (up *Unsubscribe) String() string {
	var topics []string

	for _, t := range up.Topics {
		topics = append(topics, fmt.Sprintf("%q", t))
	}

	return fmt.Sprintf("<Unsubscribe Topics=[%s]>",
		strings.Join(topics, ", "))
}

// Len returns the byte length of the encoded packet.
func (up *Unsubscribe) Len() int {
	ml := up.len()
	return headerLen(ml) + ml
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (up *Unsubscribe) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, _, rl, err := headerDecode(src[total:], UNSUBSCRIBE)
	total += hl
	if err != nil {
		return total, err
	}

	// check buffer length
	if len(src) < total+2 {
		return total, makeError(up.Type(), "insufficient buffer size, expected %d, got %d", total+2, len(src))
	}

	// read packet id
	up.ID = ID(binary.BigEndian.Uint16(src[total:]))
	total += 2

	// check packet id
	if !up.ID.Valid() {
		return total, makeError(up.Type(), "packet id must be grater than zero")
	}

	// prepare counter
	tl := int(rl) - 2

	// reset topics
	up.Topics = up.Topics[:0]

	for tl > 0 {
		// read topic
		t, n, err := readLPString(src[total:], up.Type())
		total += n
		if err != nil {
			return total, err
		}

		// append to list
		up.Topics = append(up.Topics, t)

		// decrement counter
		tl = tl - n - 1
	}

	// check for empty list
	if len(up.Topics) == 0 {
		return total, makeError(up.Type(), "empty topic list")
	}

	return total, nil
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (up *Unsubscribe) Encode(dst []byte) (int, error) {
	total := 0

	// check packet id
	if !up.ID.Valid() {
		return total, makeError(up.Type(), "packet id must be grater than zero")
	}

	// encode header
	n, err := headerEncode(dst[total:], 0, up.len(), up.Len(), UNSUBSCRIBE)
	total += n
	if err != nil {
		return total, err
	}

	// write packet id
	binary.BigEndian.PutUint16(dst[total:], uint16(up.ID))
	total += 2

	for _, t := range up.Topics {
		// write topic
		n, err := writeLPString(dst[total:], t, up.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// Returns the payload length.
func (up *Unsubscribe) len() int {
	// packet ID
	total := 2

	for _, t := range up.Topics {
		total += 2 + len(t)
	}

	return total
}
