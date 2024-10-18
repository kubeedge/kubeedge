package packer

const (
	// pakcage version
	MajorVersion = 1
	MinorVersion = 1
	FixVersion   = 1
)

// make up version
func makeUpVersion(major, minor, fix uint8) uint32 {
	return uint32(major)<<24 | uint32(minor)<<16 | uint32(fix)<<8
}

// break down version
func breadDownVersion(version uint32) (uint8, uint8, uint8) {
	return uint8(version >> 24), uint8(version >> 16), uint8(version >> 8)
}
