package packet

import (
	"encoding/binary"
)

const maxRemainingLength = 268435455 // 256 MB

func headerLen(rl int) int {
	// packet type and flag byte
	total := 1

	if rl <= 127 {
		total++
	} else if rl <= 16383 {
		total += 2
	} else if rl <= 2097151 {
		total += 3
	} else {
		total += 4
	}

	return total
}

func headerEncode(dst []byte, flags byte, rl int, tl int, t Type) (int, error) {
	total := 0

	// check buffer length
	if len(dst) < tl {
		return total, makeError(t, "insufficient buffer size, expected %d, got %d", tl, len(dst))
	}

	// check remaining length
	if rl > maxRemainingLength || rl < 0 {
		return total, makeError(t, "remaining length (%d) out of bound (max %d, min 0)", rl, maxRemainingLength)
	}

	// check header length
	hl := headerLen(rl)
	if len(dst) < hl {
		return total, makeError(t, "insufficient buffer size, expected %d, got %d", hl, len(dst))
	}

	// write type and flags
	typeAndFlags := byte(t)<<4 | (t.defaultFlags() & 0xf)
	typeAndFlags |= flags
	dst[total] = typeAndFlags
	total++

	// write remaining length
	n := binary.PutUvarint(dst[total:], uint64(rl))
	total += n

	return total, nil
}

func headerDecode(src []byte, t Type) (int, byte, int, error) {
	total := 0

	// check buffer size
	if len(src) < 2 {
		return total, 0, 0, makeError(t, "insufficient buffer size, expected %d, got %d", 2, len(src))
	}

	// read type and flags
	typeAndFlags := src[total : total+1]
	decodedType := Type(typeAndFlags[0] >> 4)
	flags := typeAndFlags[0] & 0x0f
	total++

	// check against static type
	if decodedType != t {
		return total, 0, 0, makeError(t, "invalid type %d", decodedType)
	}

	// check flags except for publish packets
	if t != PUBLISH && flags != t.defaultFlags() {
		return total, 0, 0, makeError(t, "invalid flags, expected %d, got %d", t.defaultFlags(), flags)
	}

	// read remaining length
	_rl, m := binary.Uvarint(src[total:])
	rl := int(_rl)
	total += m

	// check resulting remaining length
	if m <= 0 {
		return total, 0, 0, makeError(t, "error reading remaining length")
	}

	// check remaining buffer
	if rl > len(src[total:]) {
		return total, 0, 0, makeError(t, "remaining length (%d) is greater than remaining buffer (%d)", rl, len(src[total:]))
	}

	return total, flags, rl, nil
}
