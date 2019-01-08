package packet

import "fmt"

// The ConnackCode represents the return code in a Connack packet.
type ConnackCode uint8

// All available ConnackCodes.
const (
	ConnectionAccepted ConnackCode = iota
	InvalidProtocolVersion
	IdentifierRejected
	ServerUnavailable
	BadUsernameOrPassword
	NotAuthorized
)

// Valid checks if the ConnackCode is valid.
func (cc ConnackCode) Valid() bool {
	return cc <= 5
}

// String returns the corresponding error string for the ConnackCode.
func (cc ConnackCode) String() string {
	switch cc {
	case ConnectionAccepted:
		return "connection accepted"
	case InvalidProtocolVersion:
		return "connection refused: unacceptable protocol version"
	case IdentifierRejected:
		return "connection refused: identifier rejected"
	case ServerUnavailable:
		return "connection refused: server unavailable"
	case BadUsernameOrPassword:
		return "connection refused: bad user name or password"
	case NotAuthorized:
		return "connection refused: not authorized"
	}

	return "invalid connack code"
}

// A Connack packet is sent by the server in response to a Connect packet
// received from a client.
type Connack struct {
	// The SessionPresent flag enables a client to establish whether the
	// client and server have a consistent view about whether there is already
	// stored session state.
	SessionPresent bool

	// If a well formed Connect packet is received by the server, but the server
	// is unable to process it for some reason, then the server should attempt
	// to send a Connack containing a non-zero ReturnCode.
	ReturnCode ConnackCode
}

// NewConnack creates a new Connack packet.
func NewConnack() *Connack {
	return &Connack{}
}

// Type returns the packets type.
func (cp *Connack) Type() Type {
	return CONNACK
}

// String returns a string representation of the packet.
func (cp *Connack) String() string {
	return fmt.Sprintf("<Connack SessionPresent=%t ReturnCode=%d>",
		cp.SessionPresent, cp.ReturnCode)
}

// Len returns the byte length of the encoded packet.
func (cp *Connack) Len() int {
	return headerLen(2) + 2
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (cp *Connack) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, _, rl, err := headerDecode(src, CONNACK)
	total += hl
	if err != nil {
		return total, err
	}

	// check remaining length
	if rl != 2 {
		return total, makeError(cp.Type(), "expected remaining length to be 2")
	}

	// read connack flags
	connackFlags := src[total]
	cp.SessionPresent = connackFlags&0x1 == 1
	total++

	// check flags
	if connackFlags&254 != 0 {
		return 0, makeError(cp.Type(), "bits 7-1 in acknowledge flags are not 0")
	}

	// read return code
	cp.ReturnCode = ConnackCode(src[total])
	total++

	// check return code
	if !cp.ReturnCode.Valid() {
		return 0, makeError(cp.Type(), "invalid return code (%d)", cp.ReturnCode)
	}

	return total, nil
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (cp *Connack) Encode(dst []byte) (int, error) {
	total := 0

	// encode header
	n, err := headerEncode(dst[total:], 0, 2, cp.Len(), CONNACK)
	total += n
	if err != nil {
		return total, err
	}

	// set session present flag
	if cp.SessionPresent {
		dst[total] = 1 // 00000001
	} else {
		dst[total] = 0 // 00000000
	}
	total++

	// check return code
	if !cp.ReturnCode.Valid() {
		return total, makeError(cp.Type(), "invalid return code (%d)", cp.ReturnCode)
	}

	// set return code
	dst[total] = byte(cp.ReturnCode)
	total++

	return total, nil
}
