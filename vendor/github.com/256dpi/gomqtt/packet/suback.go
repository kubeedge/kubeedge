package packet

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// A Suback packet is sent by the server to the client to confirm receipt and
// processing of a Subscribe packet. The Suback packet contains a list of return
// codes, that specify the maximum QOS levels that have been granted.
type Suback struct {
	// The granted QOS levels for the requested subscriptions.
	ReturnCodes []QOS

	// The packet identifier.
	ID ID
}

// NewSuback creates a new Suback packet.
func NewSuback() *Suback {
	return &Suback{}
}

// Type returns the packets type.
func (sp *Suback) Type() Type {
	return SUBACK
}

// String returns a string representation of the packet.
func (sp *Suback) String() string {
	var codes []string

	for _, c := range sp.ReturnCodes {
		codes = append(codes, fmt.Sprintf("%d", c))
	}

	return fmt.Sprintf("<Suback ID=%d ReturnCodes=[%s]>",
		sp.ID, strings.Join(codes, ", "))
}

// Len returns the byte length of the encoded packet.
func (sp *Suback) Len() int {
	ml := sp.len()
	return headerLen(ml) + ml
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (sp *Suback) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, _, rl, err := headerDecode(src[total:], SUBACK)
	total += hl
	if err != nil {
		return total, err
	}

	// check buffer length
	if len(src) < total+2 {
		return total, makeError(sp.Type(), "insufficient buffer size, expected %d, got %d", total+2, len(src))
	}

	// check remaining length
	if rl <= 2 {
		return total, makeError(sp.Type(), "expected remaining length to be greater than 2, got %d", rl)
	}

	// read packet id
	sp.ID = ID(binary.BigEndian.Uint16(src[total:]))
	total += 2

	// check packet id
	if !sp.ID.Valid() {
		return total, makeError(sp.Type(), "packet id must be grater than zero")
	}

	// calculate number of return codes
	rcl := int(rl) - 2

	// read return codes
	sp.ReturnCodes = make([]QOS, rcl)
	for i, rc := range src[total : total+rcl] {
		sp.ReturnCodes[i] = QOS(rc)
	}
	total += len(sp.ReturnCodes)

	// validate return codes
	for i, code := range sp.ReturnCodes {
		if !code.Successful() && code != QOSFailure {
			return total, makeError(sp.Type(), "invalid return code %d for topic %d", code, i)
		}
	}

	return total, nil
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (sp *Suback) Encode(dst []byte) (int, error) {
	total := 0

	// check return codes
	for i, code := range sp.ReturnCodes {
		if !code.Successful() && code != QOSFailure {
			return total, makeError(sp.Type(), "invalid return code %d for topic %d", code, i)
		}
	}

	// check packet id
	if !sp.ID.Valid() {
		return total, makeError(sp.Type(), "packet id must be grater than zero")
	}

	// encode header
	n, err := headerEncode(dst[total:], 0, sp.len(), sp.Len(), SUBACK)
	total += n
	if err != nil {
		return total, err
	}

	// write packet id
	binary.BigEndian.PutUint16(dst[total:], uint16(sp.ID))
	total += 2

	// write return codes
	for i, rc := range sp.ReturnCodes {
		dst[total+i] = byte(rc)
	}
	total += len(sp.ReturnCodes)

	return total, nil
}

// Returns the payload length.
func (sp *Suback) len() int {
	return 2 + len(sp.ReturnCodes)
}
