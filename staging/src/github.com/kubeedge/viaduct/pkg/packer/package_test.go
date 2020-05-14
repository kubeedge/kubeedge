package packer

import (
	"reflect"
	"testing"
)

// TestNewPackageHeader is function to test NewPackageHeader().
func TestNewPackageHeader(t *testing.T) {
	tests := []struct {
		name        string
		packageType PackageType
		want        *PackageHeader
	}{
		{
			name:        "NewPackageHeaderTest",
			packageType: 00,
			want: &PackageHeader{
				PackageType: 00,
				Version:     makeUpVersion(MajorVersion, MinorVersion, FixVersion),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPackageHeader(tt.packageType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPackageHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSetVersion is function to test SetVersion().
func TestSetVersion(t *testing.T) {
	tests := []struct {
		name    string
		want    *PackageHeader
		version uint32
	}{
		{
			name:    "SetVersionTest",
			version: 00,
			want:    &PackageHeader{Version: 00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{}
			if got := h.SetVersion(tt.version); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PackageHeader.SetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetVersion is function to test GetVersion().
func TestGetVersion(t *testing.T) {
	tests := []struct {
		name    string
		Version uint32
		want    uint32
	}{
		{
			name:    "GetVersionTest",
			Version: 00,
			want:    00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				Version: tt.Version,
			}
			if got := h.GetVersion(); got != tt.want {
				t.Errorf("PackageHeader.GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSetPayloadLen is function to test SetPayloadLen().
func TestSetPayloadLen(t *testing.T) {
	tests := []struct {
		name       string
		PayloadLen uint32
		want       *PackageHeader
	}{
		{
			name:       "SetPayloadLenTest",
			PayloadLen: 00,
			want:       &PackageHeader{PayloadLen: 00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				PayloadLen: tt.PayloadLen,
			}
			if got := h.SetPayloadLen(tt.PayloadLen); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PackageHeader.SetPayloadLen() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetPayloadLen is function to test GetPayloadLen().
func TestGetPayloadLen(t *testing.T) {
	tests := []struct {
		name       string
		PayloadLen uint32
		want       uint32
	}{
		{
			name:       "GetPayloadLenTest",
			PayloadLen: 00,
			want:       00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				PayloadLen: tt.PayloadLen,
			}
			if got := h.GetPayloadLen(); got != tt.want {
				t.Errorf("PackageHeader.GetPayloadLen() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSetPackageType is function to test SetPackageType().
func TestSetPackageType(t *testing.T) {
	tests := []struct {
		name        string
		PackageType PackageType
		want        *PackageHeader
	}{
		{
			name:        "SetPackageTypeTest",
			PackageType: 00,
			want:        &PackageHeader{PackageType: 00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				PackageType: tt.PackageType,
			}
			if got := h.SetPackageType(tt.PackageType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PackageHeader.SetPackageType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetPackageType is function to test GetPackageType().
func TestGetPackageType(t *testing.T) {
	tests := []struct {
		name        string
		PackageType PackageType
		want        PackageType
	}{
		{
			name:        "GetPackageTypeTest",
			PackageType: 00,
			want:        00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				PackageType: tt.PackageType,
			}
			if got := h.GetPackageType(); got != tt.want {
				t.Errorf("PackageHeader.GetPackageType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSetFlags is function to test SetFlags().
func TestSetFlags(t *testing.T) {
	tests := []struct {
		name  string
		Flags uint8
		want  *PackageHeader
	}{
		{
			name:  "SetFlagsTest",
			Flags: 00,
			want:  &PackageHeader{Flags: 00},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				Flags: tt.Flags,
			}
			if got := h.SetFlags(tt.Flags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PackageHeader.SetFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetFlags is function to test GetFlags().
func TestGetFlags(t *testing.T) {
	tests := []struct {
		name  string
		Flags uint8
		want  uint8
	}{
		{
			name:  "GetFlagsTest",
			Flags: 00,
			want:  00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &PackageHeader{
				Flags: tt.Flags,
			}
			if got := h.GetFlags(); got != tt.want {
				t.Errorf("PackageHeader.GetFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}
