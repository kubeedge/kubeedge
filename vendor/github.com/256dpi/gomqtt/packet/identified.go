package packet

import (
	"encoding/binary"
	"fmt"
)

// returns the byte length of an identified packet
func identifiedLen() int {
	return headerLen(2) + 2
}

// decodes an identified packet
func identifiedDecode(src []byte, t Type) (int, ID, error) {
	total := 0

	// decode header
	hl, _, rl, err := headerDecode(src, t)
	total += hl
	if err != nil {
		return total, 0, err
	}

	// check remaining length
	if rl != 2 {
		return total, 0, makeError(t, "expected remaining length to be 2")
	}

	// read packet id
	packetID := ID(binary.BigEndian.Uint16(src[total:]))
	total += 2

	// check packet id
	if !packetID.Valid() {
		return total, 0, makeError(t, "packet id must be grater than zero")
	}

	return total, packetID, nil
}

// encodes an identified packet
func identifiedEncode(dst []byte, id ID, t Type) (int, error) {
	total := 0

	// check packet id
	if !id.Valid() {
		return total, makeError(t, "packet id must be grater than zero")
	}

	// encode header
	n, err := headerEncode(dst[total:], 0, 2, identifiedLen(), t)
	total += n
	if err != nil {
		return total, err
	}

	// write packet id
	binary.BigEndian.PutUint16(dst[total:], uint16(id))
	total += 2

	return total, nil
}

// A Puback packet is the response to a Publish packet with QOS level 1.
type Puback struct {
	// The packet identifier.
	ID ID
}

// NewPuback creates a new Puback packet.
func NewPuback() *Puback {
	return &Puback{}
}

// Type returns the packets type.
func (pp *Puback) Type() Type {
	return PUBACK
}

// Len returns the byte length of the encoded packet.
func (pp *Puback) Len() int {
	return identifiedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Puback) Decode(src []byte) (int, error) {
	n, pid, err := identifiedDecode(src, PUBACK)
	pp.ID = pid
	return n, err
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Puback) Encode(dst []byte) (int, error) {
	return identifiedEncode(dst, pp.ID, PUBACK)
}

// String returns a string representation of the packet.
func (pp *Puback) String() string {
	return fmt.Sprintf("<Puback ID=%d>", pp.ID)
}

// A Pubcomp packet is the response to a Pubrel. It is the fourth and
// final packet of the QOS 2 protocol exchange.
type Pubcomp struct {
	// The packet identifier.
	ID ID
}

var _ Generic = (*Pubcomp)(nil)

// NewPubcomp creates a new Pubcomp packet.
func NewPubcomp() *Pubcomp {
	return &Pubcomp{}
}

// Type returns the packets type.
func (pp *Pubcomp) Type() Type {
	return PUBCOMP
}

// Len returns the byte length of the encoded packet.
func (pp *Pubcomp) Len() int {
	return identifiedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Pubcomp) Decode(src []byte) (int, error) {
	n, pid, err := identifiedDecode(src, PUBCOMP)
	pp.ID = pid
	return n, err
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Pubcomp) Encode(dst []byte) (int, error) {
	return identifiedEncode(dst, pp.ID, PUBCOMP)
}

// String returns a string representation of the packet.
func (pp *Pubcomp) String() string {
	return fmt.Sprintf("<Pubcomp ID=%d>", pp.ID)
}

// A Pubrec packet is the response to a Publish packet with QOS 2. It is the
// second packet of the QOS 2 protocol exchange.
type Pubrec struct {
	// Shared packet identifier.
	ID ID
}

// NewPubrec creates a new Pubrec packet.
func NewPubrec() *Pubrec {
	return &Pubrec{}
}

// Type returns the packets type.
func (pp *Pubrec) Type() Type {
	return PUBREC
}

// Len returns the byte length of the encoded packet.
func (pp *Pubrec) Len() int {
	return identifiedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Pubrec) Decode(src []byte) (int, error) {
	n, pid, err := identifiedDecode(src, PUBREC)
	pp.ID = pid
	return n, err
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Pubrec) Encode(dst []byte) (int, error) {
	return identifiedEncode(dst, pp.ID, PUBREC)
}

// String returns a string representation of the packet.
func (pp *Pubrec) String() string {
	return fmt.Sprintf("<Pubrec ID=%d>", pp.ID)
}

// A Pubrel packet is the response to a Pubrec packet. It is the third packet of
// the QOS 2 protocol exchange.
type Pubrel struct {
	// Shared packet identifier.
	ID ID
}

var _ Generic = (*Pubrel)(nil)

// NewPubrel creates a new Pubrel packet.
func NewPubrel() *Pubrel {
	return &Pubrel{}
}

// Type returns the packets type.
func (pp *Pubrel) Type() Type {
	return PUBREL
}

// Len returns the byte length of the encoded packet.
func (pp *Pubrel) Len() int {
	return identifiedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Pubrel) Decode(src []byte) (int, error) {
	n, pid, err := identifiedDecode(src, PUBREL)
	pp.ID = pid
	return n, err
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Pubrel) Encode(dst []byte) (int, error) {
	return identifiedEncode(dst, pp.ID, PUBREL)
}

// String returns a string representation of the packet.
func (pp *Pubrel) String() string {
	return fmt.Sprintf("<Pubrel ID=%d>", pp.ID)
}

// An Unsuback packet is sent by the server to the client to confirm receipt of
// an Unsubscribe packet.
type Unsuback struct {
	// Shared packet identifier.
	ID ID
}

// NewUnsuback creates a new Unsuback packet.
func NewUnsuback() *Unsuback {
	return &Unsuback{}
}

// Type returns the packets type.
func (up *Unsuback) Type() Type {
	return UNSUBACK
}

// Len returns the byte length of the encoded packet.
func (up *Unsuback) Len() int {
	return identifiedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (up *Unsuback) Decode(src []byte) (int, error) {
	n, pid, err := identifiedDecode(src, UNSUBACK)
	up.ID = pid
	return n, err
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (up *Unsuback) Encode(dst []byte) (int, error) {
	return identifiedEncode(dst, up.ID, UNSUBACK)
}

// String returns a string representation of the packet.
func (up *Unsuback) String() string {
	return fmt.Sprintf("<Unsuback ID=%d>", up.ID)
}
