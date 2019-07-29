package packet

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// A Subscription is a single subscription in a Subscribe packet.
type Subscription struct {
	// The topic to subscribe.
	Topic string

	// The requested maximum QOS level.
	QOS QOS
}

func (s *Subscription) String() string {
	return fmt.Sprintf("%q=>%d", s.Topic, s.QOS)
}

// A Subscribe packet is sent from the client to the server to create one or
// more Subscriptions. The server will forward application messages that match
// these subscriptions using PublishPackets.
type Subscribe struct {
	// The subscriptions.
	Subscriptions []Subscription

	// The packet identifier.
	ID ID
}

// NewSubscribe creates a new Subscribe packet.
func NewSubscribe() *Subscribe {
	return &Subscribe{}
}

// Type returns the packets type.
func (sp *Subscribe) Type() Type {
	return SUBSCRIBE
}

// String returns a string representation of the packet.
func (sp *Subscribe) String() string {
	var subscriptions []string

	for _, t := range sp.Subscriptions {
		subscriptions = append(subscriptions, t.String())
	}

	return fmt.Sprintf("<Subscribe ID=%d Subscriptions=[%s]>",
		sp.ID, strings.Join(subscriptions, ", "))
}

// Len returns the byte length of the encoded packet.
func (sp *Subscribe) Len() int {
	ml := sp.len()
	return headerLen(ml) + ml
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (sp *Subscribe) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, _, rl, err := headerDecode(src[total:], SUBSCRIBE)
	total += hl
	if err != nil {
		return total, err
	}

	// check buffer length
	if len(src) < total+2 {
		return total, makeError(sp.Type(), "insufficient buffer size, expected %d, got %d", total+2, len(src))
	}

	// read packet id
	sp.ID = ID(binary.BigEndian.Uint16(src[total:]))
	total += 2

	// check packet id
	if !sp.ID.Valid() {
		return total, makeError(sp.Type(), "packet id must be grater than zero")
	}

	// reset subscriptions
	sp.Subscriptions = sp.Subscriptions[:0]

	// calculate number of subscriptions
	sl := int(rl) - 2

	for sl > 0 {
		// read topic
		t, n, err := readLPString(src[total:], sp.Type())
		total += n
		if err != nil {
			return total, err
		}

		// check buffer length
		if len(src) < total+1 {
			return total, makeError(sp.Type(), "insufficient buffer size, expected %d, got %d", total+1, len(src))
		}

		// read qos
		qos := QOS(src[total])
		if !qos.Successful() {
			return total, makeError(sp.Type(), "invalid QOS level (%d)", qos)
		}

		// read qos and add subscription
		sp.Subscriptions = append(sp.Subscriptions, Subscription{t, qos})
		total++

		// decrement counter
		sl = sl - n - 1
	}

	// check for empty subscription list
	if len(sp.Subscriptions) == 0 {
		return total, makeError(sp.Type(), "empty subscription list")
	}

	return total, nil
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (sp *Subscribe) Encode(dst []byte) (int, error) {
	total := 0

	// check packet id
	if !sp.ID.Valid() {
		return total, makeError(sp.Type(), "packet id must be grater than zero")
	}

	// encode header
	n, err := headerEncode(dst[total:], 0, sp.len(), sp.Len(), SUBSCRIBE)
	total += n
	if err != nil {
		return total, err
	}

	// write packet id
	binary.BigEndian.PutUint16(dst[total:], uint16(sp.ID))
	total += 2

	for _, t := range sp.Subscriptions {
		// write topic
		n, err := writeLPString(dst[total:], t.Topic, sp.Type())
		total += n
		if err != nil {
			return total, err
		}

		// check qos
		if !t.QOS.Successful() {
			return total, makeError(sp.Type(), "invalid QOS level (%d)", t.QOS)
		}

		// write qos
		dst[total] = byte(t.QOS)

		total++
	}

	return total, nil
}

// Returns the payload length.
func (sp *Subscribe) len() int {
	// packet ID
	total := 2

	for _, t := range sp.Subscriptions {
		total += 2 + len(t.Topic) + 1
	}

	return total
}
