package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidIP(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Name   string
		IP     string
		Expect bool
	}{
		{
			Name:   "valid ip",
			IP:     "1.1.1.1",
			Expect: true,
		},
		{
			Name:   "invalid have port",
			IP:     "1.1.1.1:1234",
			Expect: false,
		},
		{
			Name:   "invalid ip1",
			IP:     "1.1.1.",
			Expect: false,
		},
		{
			Name:   "invalid unit socket",
			IP:     "unix:///var/run/docker.sock",
			Expect: false,
		},
		{
			Name:   "invalid http",
			IP:     "http://127.0.0.1",
			Expect: false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := IsValidIP(c.IP)
			get := len(v) == 0
			assert.Equal(c.Expect, get)
		})
	}
}

func TestIsValidPortNum(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Name   string
		Port   int
		Expect []string
	}{
		{
			Name:   "invalid port",
			Port:   0,
			Expect: []string{"must be between 1 and 65535, inclusive"},
		},
		{
			Name:   "valid port",
			Port:   1,
			Expect: nil,
		},
		{
			Name:   "valid port",
			Port:   65535,
			Expect: nil,
		},
		{
			Name:   "invalid port",
			Port:   65536,
			Expect: []string{"must be between 1 and 65535, inclusive"},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			v := IsValidPortNum(c.Port)
			assert.Equal(c.Expect, v)
		})
	}
}

func TestInclusiveRangeError(t *testing.T) {
	assert := assert.New(t)

	result := InclusiveRangeError(1, 65535)
	expect := "must be between 1 and 65535, inclusive"
	assert.Equal(expect, result)
}

func TestValidateImageRepo(t *testing.T) {
	cases := []struct {
		imageRepo string
		want      bool
	}{
		{
			imageRepo: "installation-package",
			want:      false,
		},
		{
			imageRepo: "kubeedge/installation-package",
			want:      true,
		},
		{
			imageRepo: "kubeedge/installation-package;bash",
			want:      false,
		},
		{
			imageRepo: "_kubeedge/installation-package",
			want:      false,
		},
		{
			imageRepo: "aaa.bbb.ccc/kubeedge/installation-package",
			want:      true,
		},
		{
			imageRepo: "registry.example.com:5000/kubeedge/installation-package",
			want:      true,
		},
		{
			imageRepo: "kubeedge/installation-package:v1.23.1",
			want:      true,
		},
		{
			imageRepo: "kubeedge/installation-package@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			want:      true,
		},
		{
			imageRepo: "kubeedge/installation-package;touch /tmp/pwned",
			want:      false,
		},
		{
			imageRepo: "kubeedge/installation-package$(touch /tmp/pwned)",
			want:      false,
		},
		{
			imageRepo: "kubeedge/installation-package`touch /tmp/pwned`",
			want:      false,
		},
		{
			imageRepo: "kubeedge/installation-package\n touch /tmp/pwned",
			want:      false,
		},
	}
	for _, c := range cases {
		t.Run(c.imageRepo, func(t *testing.T) {
			assert.Equal(t, c.want, ValidateImageRepo(c.imageRepo))
		})
	}
}

func TestValidateVersion(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{version: "v1.0.0", want: true},
		{version: "V1.0.0", want: false},
		{version: "1.0.0", want: false},
		{version: "v1.0", want: false},
		{version: "v1.0.0;bash", want: false},
		{version: "v1.0.0-rc1", want: true},
		{version: "v1.0.0-rc1.1", want: true},
	}
	for _, c := range cases {
		t.Run(c.version, func(t *testing.T) {
			assert.Equal(t, c.want, ValidateVersion(c.version))
		})
	}
}
