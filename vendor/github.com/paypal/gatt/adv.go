package gatt

import (
	"errors"
	"log"
)

// MaxEIRPacketLength is the maximum allowed AdvertisingPacket
// and ScanResponsePacket length.
const MaxEIRPacketLength = 31

// ErrEIRPacketTooLong is the error returned when an AdvertisingPacket
// or ScanResponsePacket is too long.
var ErrEIRPacketTooLong = errors.New("max packet length is 31")

// Advertising data field types
const (
	typeFlags             = 0x01 // Flags
	typeSomeUUID16        = 0x02 // Incomplete List of 16-bit Service Class UUIDs
	typeAllUUID16         = 0x03 // Complete List of 16-bit Service Class UUIDs
	typeSomeUUID32        = 0x04 // Incomplete List of 32-bit Service Class UUIDs
	typeAllUUID32         = 0x05 // Complete List of 32-bit Service Class UUIDs
	typeSomeUUID128       = 0x06 // Incomplete List of 128-bit Service Class UUIDs
	typeAllUUID128        = 0x07 // Complete List of 128-bit Service Class UUIDs
	typeShortName         = 0x08 // Shortened Local Name
	typeCompleteName      = 0x09 // Complete Local Name
	typeTxPower           = 0x0A // Tx Power Level
	typeClassOfDevice     = 0x0D // Class of Device
	typeSimplePairingC192 = 0x0E // Simple Pairing Hash C-192
	typeSimplePairingR192 = 0x0F // Simple Pairing Randomizer R-192
	typeSecManagerTK      = 0x10 // Security Manager TK Value
	typeSecManagerOOB     = 0x11 // Security Manager Out of Band Flags
	typeSlaveConnInt      = 0x12 // Slave Connection Interval Range
	typeServiceSol16      = 0x14 // List of 16-bit Service Solicitation UUIDs
	typeServiceSol128     = 0x15 // List of 128-bit Service Solicitation UUIDs
	typeServiceData16     = 0x16 // Service Data - 16-bit UUID
	typePubTargetAddr     = 0x17 // Public Target Address
	typeRandTargetAddr    = 0x18 // Random Target Address
	typeAppearance        = 0x19 // Appearance
	typeAdvInterval       = 0x1A // Advertising Interval
	typeLEDeviceAddr      = 0x1B // LE Bluetooth Device Address
	typeLERole            = 0x1C // LE Role
	typeServiceSol32      = 0x1F // List of 32-bit Service Solicitation UUIDs
	typeServiceData32     = 0x20 // Service Data - 32-bit UUID
	typeServiceData128    = 0x21 // Service Data - 128-bit UUID
	typeLESecConfirm      = 0x22 // LE Secure Connections Confirmation Value
	typeLESecRandom       = 0x23 // LE Secure Connections Random Value
	typeManufacturerData  = 0xFF // Manufacturer Specific Data
)

// Advertising type flags
const (
	flagLimitedDiscoverable = 0x01 // LE Limited Discoverable Mode
	flagGeneralDiscoverable = 0x02 // LE General Discoverable Mode
	flagLEOnly              = 0x04 // BR/EDR Not Supported. Bit 37 of LMP Feature Mask Definitions (Page 0)
	flagBothController      = 0x08 // Simultaneous LE and BR/EDR to Same Device Capable (Controller).
	flagBothHost            = 0x10 // Simultaneous LE and BR/EDR to Same Device Capable (Host).
)

// FIXME: check the unmarshalling of this data structure.
type ServiceData struct {
	UUID UUID
	Data []byte
}

// This is borrowed from core bluetooth.
// Embedded/Linux folks might be interested in more details.
type Advertisement struct {
	LocalName        string
	ManufacturerData []byte
	ServiceData      []ServiceData
	Services         []UUID
	OverflowService  []UUID
	TxPowerLevel     int
	Connectable      bool
	SolicitedService []UUID
}

// This is only used in Linux port.
func (a *Advertisement) unmarshall(b []byte) error {

	// Utility function for creating a list of uuids.
	uuidList := func(u []UUID, d []byte, w int) []UUID {
		for len(d) > 0 {
			u = append(u, UUID{d[:w]})
			d = d[w:]
		}
		return u
	}

	for len(b) > 0 {
		if len(b) < 2 {
			return errors.New("invalid advertise data")
		}
		l, t := b[0], b[1]
		if len(b) < int(1+l) {
			return errors.New("invalid advertise data")
		}
		d := b[2 : 1+l]
		switch t {
		case typeFlags:
			// TODO: should we do anything about the discoverability here?
		case typeSomeUUID16:
			a.Services = uuidList(a.Services, d, 2)
		case typeAllUUID16:
			a.Services = uuidList(a.Services, d, 2)
		case typeSomeUUID32:
			a.Services = uuidList(a.Services, d, 4)
		case typeAllUUID32:
			a.Services = uuidList(a.Services, d, 4)
		case typeSomeUUID128:
			a.Services = uuidList(a.Services, d, 16)
		case typeAllUUID128:
			a.Services = uuidList(a.Services, d, 16)
		case typeShortName:
			a.LocalName = string(d)
		case typeCompleteName:
			a.LocalName = string(d)
		case typeTxPower:
			a.TxPowerLevel = int(d[0])
		case typeServiceSol16:
			a.SolicitedService = uuidList(a.SolicitedService, d, 2)
		case typeServiceSol128:
			a.SolicitedService = uuidList(a.SolicitedService, d, 16)
		case typeServiceSol32:
			a.SolicitedService = uuidList(a.SolicitedService, d, 4)
		case typeManufacturerData:
			a.ManufacturerData = make([]byte, len(d))
			copy(a.ManufacturerData, d)
		// case typeServiceData16,
		// case typeServiceData32,
		// case typeServiceData128:
		default:
			log.Printf("DATA: [ % X ]", d)
		}
		b = b[1+l:]
	}
	return nil
}

// AdvPacket is an utility to help crafting advertisment or scan response data.
type AdvPacket struct {
	b []byte
}

// Bytes returns an 31-byte array, which contains up to 31 bytes of the packet.
func (a *AdvPacket) Bytes() [31]byte {
	b := [31]byte{}
	copy(b[:], a.b)
	return b
}

// Len returns the length of the packets with a maximum of 31.
func (a *AdvPacket) Len() int {
	if len(a.b) > 31 {
		return 31
	}
	return len(a.b)
}

// AppendField appends a BLE advertising packet field.
// TODO: refuse to append field if it'd make the packet too long.
func (a *AdvPacket) AppendField(typ byte, b []byte) *AdvPacket {
	// A field consists of len, typ, b.
	// Len is 1 byte for typ plus len(b).
	if len(a.b)+2+len(b) > MaxEIRPacketLength {
		b = b[:MaxEIRPacketLength-len(a.b)-2]
	}
	a.b = append(a.b, byte(len(b)+1))
	a.b = append(a.b, typ)
	a.b = append(a.b, b...)
	return a
}

// AppendFlags appends a flag field to the packet.
func (a *AdvPacket) AppendFlags(f byte) *AdvPacket {
	return a.AppendField(typeFlags, []byte{f})
}

// AppendFlags appends a name field to the packet.
// If the name fits in the space, it will be append as a complete name field, otherwise a short name field.
func (a *AdvPacket) AppendName(n string) *AdvPacket {
	typ := byte(typeCompleteName)
	if len(a.b)+2+len(n) > MaxEIRPacketLength {
		typ = byte(typeShortName)
	}
	return a.AppendField(typ, []byte(n))
}

// AppendManufacturerData appends a manufacturer data field to the packet.
func (a *AdvPacket) AppendManufacturerData(id uint16, b []byte) *AdvPacket {
	d := append([]byte{uint8(id), uint8(id >> 8)}, b...)
	return a.AppendField(typeManufacturerData, d)
}

// AppendUUIDFit appends a BLE advertised service UUID
// packet field if it fits in the packet, and reports whether the UUID fit.
func (a *AdvPacket) AppendUUIDFit(uu []UUID) bool {
	// Iterate all UUIDs to see if they fit in the packet or not.
	fit, l := true, len(a.b)
	for _, u := range uu {
		if u.Equal(attrGAPUUID) || u.Equal(attrGATTUUID) {
			continue
		}
		l += 2 + u.Len()
		if l > MaxEIRPacketLength {
			fit = false
			break
		}
	}

	// Append the UUIDs until they no longer fit.
	for _, u := range uu {
		if u.Equal(attrGAPUUID) || u.Equal(attrGATTUUID) {
			continue
		}
		if len(a.b)+2+u.Len() > MaxEIRPacketLength {
			break
		}
		switch l = u.Len(); {
		case l == 2 && fit:
			a.AppendField(typeAllUUID16, u.b)
		case l == 16 && fit:
			a.AppendField(typeAllUUID128, u.b)
		case l == 2 && !fit:
			a.AppendField(typeSomeUUID16, u.b)
		case l == 16 && !fit:
			a.AppendField(typeSomeUUID128, u.b)
		}
	}
	return fit
}
