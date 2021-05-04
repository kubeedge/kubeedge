package validation

import "testing"

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
			Name:   "unvalid have port",
			IP:     "1.1.1.1:1234",
			Expect: false,
		},
		{
			Name:   "unvalid ip1",
			IP:     "1.1.1.",
			Expect: false,
		},
		{
			Name:   "unvalid unit socket",
			IP:     "unix:///var/run/docker.sock",
			Expect: false,
		},
		{
			Name:   "unvalid http",
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
