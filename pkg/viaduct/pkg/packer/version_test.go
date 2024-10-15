package packer

import "testing"

// TestMakeUpVersion is function to test makeUpVersion().
func TestMakeUpVersion(t *testing.T) {
	tests := []struct {
		name  string
		major uint8
		minor uint8
		fix   uint8
		want  uint32
	}{
		{
			name:  "MakeUpversionTest",
			major: FixVersion,
			minor: MinorVersion,
			fix:   FixVersion,
			want:  16843008,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeUpVersion(tt.major, tt.minor, tt.fix); got != tt.want {
				t.Errorf("makeUpVersion() = %v, majorVersion %v", got, tt.want)
			}
		})
	}
}

// TestBreadDownVersion is function to test breadDownVersion().
func TestBreadDownVersion(t *testing.T) {
	tests := []struct {
		name          string
		version       uint32
		majorVersion  uint8
		middleVersion uint8
		minorVersion  uint8
	}{
		{
			name:          "BreadDownVersionTest",
			version:       00,
			majorVersion:  00,
			middleVersion: 00,
			minorVersion:  00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMajorVersion, gotMiddleVersion, gotMinorVersion := breadDownVersion(tt.version)
			if gotMajorVersion != tt.majorVersion {
				t.Errorf("breadDownVersion() gotMajorVersion = %v, majorVersion %v", gotMajorVersion, tt.majorVersion)
			}
			if gotMiddleVersion != tt.middleVersion {
				t.Errorf("breadDownVersion() gotMiddleVersion = %v, majorVersion %v", gotMiddleVersion, tt.middleVersion)
			}
			if gotMinorVersion != tt.minorVersion {
				t.Errorf("breadDownVersion() gotMinorVersion = %v, majorVersion %v", gotMinorVersion, tt.minorVersion)
			}
		})
	}
}
