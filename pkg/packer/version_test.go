package packer

import "testing"

// Test_makeUpVersion is function to test makeUpVersion().
func Test_makeUpVersion(t *testing.T) {
	tests := []struct {
		name string
		major uint8
		minor uint8
		fix   uint8
		want uint32
	}{
		{
			name:"MakeUpversionTest",
			major:FixVersion,
			minor:MinorVersion,
			fix:FixVersion,
			want:16843008,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeUpVersion(tt.major, tt.minor, tt.fix); got != tt.want {
				t.Errorf("makeUpVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_breadDownVersion is function to test breadDownVersion().
func Test_breadDownVersion(t *testing.T) {
	tests := []struct {
		name  string
		version uint32
		want  uint8
		want1 uint8
		want2 uint8
	}{
		{
			name:"BreadDownVersionTest",
			version:00,
			want:00,
			want1:00,
			want2:00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := breadDownVersion(tt.version)
			if got != tt.want {
				t.Errorf("breadDownVersion() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("breadDownVersion() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("breadDownVersion() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}
