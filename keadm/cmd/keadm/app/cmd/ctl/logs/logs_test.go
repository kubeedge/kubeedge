/*
Copyright 2026 The KubeEdge Authors.

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

package logs

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestParseSinceSeconds(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: "", want: 0},
		{in: "0", want: 0},
		{in: "300", want: 300},
		{in: "5s", want: 5},
		{in: "2m", want: 120},
		{in: "3h", want: 10800},
		{in: "1m30s", want: 90},
		{in: "1500ms", want: 2},
		{in: "-5", wantErr: true},
		{in: "-5s", wantErr: true},
		{in: "0s", wantErr: true},
		{in: "abc", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseSinceSeconds(tc.in)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAddQueryParamsSinceSeconds(t *testing.T) {
	newReq := func() *rest.Request {
		return rest.NewRequestWithClient(&url.URL{Scheme: "https", Host: "127.0.0.1:10550"}, "", rest.ClientContentConfig{}, nil)
	}

	for in, want := range map[string]string{"5s": "5", "2m": "120", "300": "300", "0": ""} {
		o := NewLogsPodOpts()
		o.SinceSecond = in
		o.TailLines = "-1"
		o.LimitBytes = "-1"
		got := addQueryParams(newReq(), o).URL().Query().Get("sinceSeconds")
		assert.Equal(t, want, got, "--since-second=%s", in)
	}
}
