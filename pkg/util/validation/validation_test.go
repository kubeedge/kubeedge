/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
