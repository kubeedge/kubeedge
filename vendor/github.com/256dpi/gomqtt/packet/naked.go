package packet

// returns the byte length of a naked packet
func nakedLen() int {
	return headerLen(0)
}

// decodes a naked packet
func nakedDecode(src []byte, t Type) (int, error) {
	// decode header
	hl, _, rl, err := headerDecode(src, t)

	// check remaining length
	if rl != 0 {
		return hl, makeError(t, "expected zero remaining length")
	}

	return hl, err
}

// encodes a naked packet
func nakedEncode(dst []byte, t Type) (int, error) {
	// encode header
	return headerEncode(dst, 0, 0, nakedLen(), t)
}

// A Disconnect packet is sent from the client to the server.
// It indicates that the client is disconnecting cleanly.
type Disconnect struct{}

// NewDisconnect creates a new Disconnect packet.
func NewDisconnect() *Disconnect {
	return &Disconnect{}
}

// Type returns the packets type.
func (dp *Disconnect) Type() Type {
	return DISCONNECT
}

// Len returns the byte length of the encoded packet.
func (dp *Disconnect) Len() int {
	return nakedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (dp *Disconnect) Decode(src []byte) (int, error) {
	return nakedDecode(src, DISCONNECT)
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (dp *Disconnect) Encode(dst []byte) (int, error) {
	return nakedEncode(dst, DISCONNECT)
}

// String returns a string representation of the packet.
func (dp *Disconnect) String() string {
	return "<Disconnect>"
}

// A Pingreq packet is sent from a client to the server.
type Pingreq struct{}

// NewPingreq creates a new Pingreq packet.
func NewPingreq() *Pingreq {
	return &Pingreq{}
}

// Type returns the packets type.
func (pp *Pingreq) Type() Type {
	return PINGREQ
}

// Len returns the byte length of the encoded packet.
func (pp *Pingreq) Len() int {
	return nakedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Pingreq) Decode(src []byte) (int, error) {
	return nakedDecode(src, PINGREQ)
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Pingreq) Encode(dst []byte) (int, error) {
	return nakedEncode(dst, PINGREQ)
}

// String returns a string representation of the packet.
func (pp *Pingreq) String() string {
	return "<Pingreq>"
}

// A Pingresp packet is sent by the server to the client in response to a
// Pingreq. It indicates that the server is alive.
type Pingresp struct{}

var _ Generic = (*Pingresp)(nil)

// NewPingresp creates a new Pingresp packet.
func NewPingresp() *Pingresp {
	return &Pingresp{}
}

// Type returns the packets type.
func (pp *Pingresp) Type() Type {
	return PINGRESP
}

// Len returns the byte length of the encoded packet.
func (pp *Pingresp) Len() int {
	return nakedLen()
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (pp *Pingresp) Decode(src []byte) (int, error) {
	return nakedDecode(src, PINGRESP)
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (pp *Pingresp) Encode(dst []byte) (int, error) {
	return nakedEncode(dst, PINGRESP)
}

// String returns a string representation of the packet.
func (pp *Pingresp) String() string {
	return "<Pingresp>"
}
