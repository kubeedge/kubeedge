package packer

const (
	// package type definition
	Message     PackageType = 0x01
	Stream      PackageType = 0x02
	UserDefined PackageType = 0x04

	// flags
	FlagCompressed = 0x80

	// the len of magic sequence
	VersionSize      = 4
	PackageTypeSize  = 1
	PackageFlagsSize = 1
	PayloadLenSize   = 4
	HeaderSize       = VersionSize + PackageTypeSize + PackageFlagsSize + PayloadLenSize

	// filed offsets
	VersionOffset     = 0
	PackageTypeOffset = VersionSize
	FlagsOffset       = VersionSize + PackageTypeSize
	PayloadLenOffset  = VersionSize + PackageTypeSize + PackageFlagsSize
)

type PackageType uint8

type PackageHeader struct {
	// major_version: Version << 24
	// minor_version: Version << 16
	// fix_version: Version << 8
	// Version => {major_version} | {minor_version} | {fix_version}
	Version uint32

	// the package type
	// message package: 0x01
	// stream package: 0x02
	// user-defined package: 0x04
	PackageType PackageType

	// flags
	Flags uint8

	// payload length
	// the size of package payload
	PayloadLen uint32
}

// new package
func NewPackageHeader(packageType PackageType) *PackageHeader {
	return &PackageHeader{
		PackageType: packageType,
		Version:     makeUpVersion(MajorVersion, MinorVersion, FixVersion),
	}
}

// set version
func (h *PackageHeader) SetVersion(version uint32) *PackageHeader {
	h.Version = version
	return h
}

// get version
func (h *PackageHeader) GetVersion() uint32 {
	return h.Version
}

// set payload len
func (h *PackageHeader) SetPayloadLen(payloadLen uint32) *PackageHeader {
	h.PayloadLen = payloadLen
	return h
}

// get payload len of package
func (h *PackageHeader) GetPayloadLen() uint32 {
	return h.PayloadLen
}

// set package type
func (h *PackageHeader) SetPackageType(packageType PackageType) *PackageHeader {
	h.PackageType = packageType
	return h
}

// get package type
func (h *PackageHeader) GetPackageType() PackageType {
	return h.PackageType
}

// set package flags
func (h *PackageHeader) SetFlags(flags uint8) *PackageHeader {
	h.Flags = flags
	return h
}

// get package flags
func (h *PackageHeader) GetFlags() uint8 {
	return h.Flags
}
