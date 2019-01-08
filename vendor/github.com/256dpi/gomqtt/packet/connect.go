package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// The supported MQTT versions.
const (
	Version311 byte = 4
	Version31  byte = 3
)

var version311Name = []byte("MQTT")
var version31Name = []byte("MQIsdp")

// A Connect packet is sent by a client to the server after a network
// connection has been established.
type Connect struct {
	// The clients client id.
	ClientID string

	// The keep alive value.
	KeepAlive uint16

	// The authentication username.
	Username string

	// The authentication password.
	Password string

	// The clean session flag.
	CleanSession bool

	// The will message.
	Will *Message

	// The MQTT version 3 or 4 (defaults to 4 when 0).
	Version byte
}

// NewConnect creates a new Connect packet.
func NewConnect() *Connect {
	return &Connect{
		CleanSession: true,
		Version:      4,
	}
}

// Type returns the packets type.
func (cp *Connect) Type() Type {
	return CONNECT
}

// String returns a string representation of the packet.
func (cp *Connect) String() string {
	will := "nil"

	if cp.Will != nil {
		will = cp.Will.String()
	}

	return fmt.Sprintf("<Connect ClientID=%q KeepAlive=%d Username=%q "+
		"Password=%q CleanSession=%t Will=%s Version=%d>",
		cp.ClientID,
		cp.KeepAlive,
		cp.Username,
		cp.Password,
		cp.CleanSession,
		will,
		cp.Version,
	)
}

// Len returns the byte length of the encoded packet.
func (cp *Connect) Len() int {
	ml := cp.len()
	return headerLen(ml) + ml
}

// Decode reads from the byte slice argument. It returns the total number of
// bytes decoded, and whether there have been any errors during the process.
func (cp *Connect) Decode(src []byte) (int, error) {
	total := 0

	// decode header
	hl, _, _, err := headerDecode(src[total:], CONNECT)
	total += hl
	if err != nil {
		return total, err
	}

	// read protocol string
	protoName, n, err := readLPBytes(src[total:], false, cp.Type())
	total += n
	if err != nil {
		return total, err
	}

	// check buffer length
	if len(src) < total+1 {
		return total, makeError(cp.Type(), "insufficient buffer size, expected %d, got %d", total+1, len(src))
	}

	// read version
	versionByte := src[total]
	total++

	// check protocol string and version
	if versionByte != Version311 && versionByte != Version31 {
		return total, makeError(cp.Type(), "invalid protocol version (%d)", versionByte)
	}

	// set version
	cp.Version = versionByte

	// check protocol version string
	if !bytes.Equal(protoName, version311Name) && !bytes.Equal(protoName, version31Name) {
		return total, makeError(cp.Type(), "invalid protocol version description (%s)", protoName)
	}

	// check buffer length
	if len(src) < total+1 {
		return total, makeError(cp.Type(), "insufficient buffer size, expected %d, got %d", total+1, len(src))
	}

	// read connect flags
	connectFlags := src[total]
	total++

	// read flags
	usernameFlag := ((connectFlags >> 7) & 0x1) == 1
	passwordFlag := ((connectFlags >> 6) & 0x1) == 1
	willFlag := ((connectFlags >> 2) & 0x1) == 1
	willRetain := ((connectFlags >> 5) & 0x1) == 1
	willQOS := QOS((connectFlags >> 3) & 0x3)
	cp.CleanSession = ((connectFlags >> 1) & 0x1) == 1

	// check reserved bit
	if connectFlags&0x1 != 0 {
		return total, makeError(cp.Type(), "reserved bit 0 is not 0")
	}

	// check will qos
	if !willQOS.Successful() {
		return total, makeError(cp.Type(), "invalid QOS level (%d) for will message", willQOS)
	}

	// check will flags
	if !willFlag && (willRetain || willQOS != 0) {
		return total, makeError(cp.Type(), "if the will flag is set to 0 the will qos and will retain fields must be set to zero")
	}

	// create will if present
	if willFlag {
		cp.Will = &Message{QOS: willQOS, Retain: willRetain}
	}

	// check auth flags
	if !usernameFlag && passwordFlag {
		return total, makeError(cp.Type(), "password flag is set but username flag is not set")
	}

	// check buffer length
	if len(src) < total+2 {
		return total, makeError(cp.Type(), "insufficient buffer size, expected %d, got %d", total+2, len(src))
	}

	// read keep alive
	cp.KeepAlive = binary.BigEndian.Uint16(src[total:])
	total += 2

	// read client id
	cp.ClientID, n, err = readLPString(src[total:], cp.Type())
	total += n
	if err != nil {
		return total, err
	}

	// if the client supplies a zero-byte clientID, the client must also set CleanSession to 1
	if len(cp.ClientID) == 0 && !cp.CleanSession {
		return total, makeError(cp.Type(), "clean session must be 1 if client id is zero length")
	}

	// read will topic and payload
	if cp.Will != nil {
		cp.Will.Topic, n, err = readLPString(src[total:], cp.Type())
		total += n
		if err != nil {
			return total, err
		}

		cp.Will.Payload, n, err = readLPBytes(src[total:], true, cp.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	// read username
	if usernameFlag {
		cp.Username, n, err = readLPString(src[total:], cp.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	// read password
	if passwordFlag {
		cp.Password, n, err = readLPString(src[total:], cp.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// Encode writes the packet bytes into the byte slice from the argument. It
// returns the number of bytes encoded and whether there's any errors along
// the way. If there is an error, the byte slice should be considered invalid.
func (cp *Connect) Encode(dst []byte) (int, error) {
	total := 0

	// encode header
	n, err := headerEncode(dst[total:], 0, cp.len(), cp.Len(), CONNECT)
	total += n
	if err != nil {
		return total, err
	}

	// set default version byte
	if cp.Version == 0 {
		cp.Version = Version311
	}

	// check version byte
	if cp.Version != Version311 && cp.Version != Version31 {
		return total, makeError(cp.Type(), "unsupported protocol version %d", cp.Version)
	}

	// write version string, length has been checked beforehand
	if cp.Version == Version311 {
		n, _ = writeLPBytes(dst[total:], version311Name, cp.Type())
		total += n
	} else if cp.Version == Version31 {
		n, _ = writeLPBytes(dst[total:], version31Name, cp.Type())
		total += n
	}

	// write version value
	dst[total] = cp.Version
	total++

	var connectFlags byte

	// set username flag
	if len(cp.Username) > 0 {
		connectFlags |= 128 // 10000000
	} else {
		connectFlags &= 127 // 01111111
	}

	// set password flag
	if len(cp.Password) > 0 {
		connectFlags |= 64 // 01000000
	} else {
		connectFlags &= 191 // 10111111
	}

	// set will flag
	if cp.Will != nil {
		connectFlags |= 0x4 // 00000100

		// check will topic length
		if len(cp.Will.Topic) == 0 {
			return total, makeError(cp.Type(), "will topic is empty")
		}

		// check will qos
		if !cp.Will.QOS.Successful() {
			return total, makeError(cp.Type(), "invalid will qos level %d", cp.Will.QOS)
		}

		// set will qos flag
		connectFlags = (connectFlags & 231) | (byte(cp.Will.QOS) << 3) // 231 = 11100111

		// set will retain flag
		if cp.Will.Retain {
			connectFlags |= 32 // 00100000
		} else {
			connectFlags &= 223 // 11011111
		}

	} else {
		connectFlags &= 251 // 11111011
	}

	// check client id and clean session
	if len(cp.ClientID) == 0 && !cp.CleanSession {
		return total, makeError(cp.Type(), "clean session must be 1 if client id is zero length")
	}

	// set clean session flag
	if cp.CleanSession {
		connectFlags |= 0x2 // 00000010
	} else {
		connectFlags &= 253 // 11111101
	}

	// write connect flags
	dst[total] = connectFlags
	total++

	// write keep alive
	binary.BigEndian.PutUint16(dst[total:], cp.KeepAlive)
	total += 2

	// write client id
	n, err = writeLPString(dst[total:], cp.ClientID, cp.Type())
	total += n
	if err != nil {
		return total, err
	}

	// write will topic and payload
	if cp.Will != nil {
		n, err = writeLPString(dst[total:], cp.Will.Topic, cp.Type())
		total += n
		if err != nil {
			return total, err
		}

		n, err = writeLPBytes(dst[total:], cp.Will.Payload, cp.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	if len(cp.Username) == 0 && len(cp.Password) > 0 {
		return total, makeError(cp.Type(), "password set without username")
	}

	// write username
	if len(cp.Username) > 0 {
		n, err = writeLPString(dst[total:], cp.Username, cp.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	// write password
	if len(cp.Password) > 0 {
		n, err = writeLPString(dst[total:], cp.Password, cp.Type())
		total += n
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

// Returns the payload length.
func (cp *Connect) len() int {
	total := 0

	if cp.Version == Version31 {
		// 2 bytes protocol name length
		// 6 bytes protocol name
		// 1 byte protocol version
		total += 2 + 6 + 1
	} else {
		// 2 bytes protocol name length
		// 4 bytes protocol name
		// 1 byte protocol version
		total += 2 + 4 + 1
	}

	// 1 byte connect flags
	// 2 bytes keep alive timer
	total += 1 + 2

	// add the clientID length
	total += 2 + len(cp.ClientID)

	// add the will topic and will message length
	if cp.Will != nil {
		total += 2 + len(cp.Will.Topic) + 2 + len(cp.Will.Payload)
	}

	// add the username length
	if len(cp.Username) > 0 {
		total += 2 + len(cp.Username)
	}

	// add the password length
	if len(cp.Password) > 0 {
		total += 2 + len(cp.Password)
	}

	return total
}
