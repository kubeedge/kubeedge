package validation

import (
	"reflect"
	"testing"
)

func TestIsValidIP(t *testing.T) {
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
			if get != c.Expect {
				t.Errorf("Input %s Expect %v while get %v", c.IP, c.Expect, v)
			}
		})
	}
}

func TestIsValidPortNum(t *testing.T) {
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
			if !reflect.DeepEqual(v, c.Expect) {
				t.Errorf("Input %d Expect %v while get %v", c.Port, c.Expect, v)
			}
		})
	}
}

func TestInclusiveRangeError(t *testing.T) {
	result := InclusiveRangeError(1, 65535)
	expect := "must be between 1 and 65535, inclusive"
	if !reflect.DeepEqual(result, expect) {
		t.Errorf("Expected %v while get %v", expect, result)
	}
}
