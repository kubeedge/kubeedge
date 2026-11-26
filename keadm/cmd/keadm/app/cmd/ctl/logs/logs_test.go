/*
Copyright 2024 The KubeEdge Authors.

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
	"net/http"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewEdgePodLogs(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgePodLogs()

	assert.NotNil(cmd)
	assert.Equal("logs", cmd.Use)
	assert.Equal(edgePodLogsShortDescription, cmd.Short)
	assert.Equal(edgePodLogsShortDescription, cmd.Long)
	assert.NotNil(cmd.RunE)

	assert.Equal("namespace", cmd.Flags().Lookup(common.FlagNameNamespace).Name)
	assert.Equal("container", cmd.Flags().Lookup(common.FlagNameContainer).Name)
	assert.Equal("follow", cmd.Flags().Lookup(common.FlagNameFollow).Name)
	assert.Equal("previous", cmd.Flags().Lookup(common.FlagNamePrevious).Name)
	assert.Equal(common.FlagNameSinceSecond, cmd.Flags().Lookup(common.FlagNameSinceSecond).Name)
	assert.Equal(common.FlagNameSinceTime, cmd.Flags().Lookup(common.FlagNameSinceTime).Name)
	assert.Equal(common.FlagNameTimestamps, cmd.Flags().Lookup(common.FlagNameTimestamps).Name)
	assert.Equal(common.FlagNameTailLines, cmd.Flags().Lookup(common.FlagNameTailLines).Name)
	assert.Equal(common.FlagNameLimitBytes, cmd.Flags().Lookup(common.FlagNameLimitBytes).Name)
	assert.Equal(common.FlagNameInsecureSkipTLSVerifyBackend, cmd.Flags().Lookup(common.FlagNameInsecureSkipTLSVerifyBackend).Name)
}

func TestNewLogsPodOpts(t *testing.T) {
	assert := assert.New(t)

	opts := NewLogsPodOpts()
	assert.NotNil(opts)
	assert.Equal("default", opts.Namespace)
}

func TestAddLogsPodFlags(t *testing.T) {
	assert := assert.New(t)
	opts := NewLogsPodOpts()
	cmd := &cobra.Command{}

	AddLogsPodFlags(cmd, opts)

	assert.Equal("default", cmd.Flags().Lookup(common.FlagNameNamespace).DefValue)
	assert.Equal("false", cmd.Flags().Lookup(common.FlagNameFollow).DefValue)
	assert.Equal("0", cmd.Flags().Lookup(common.FlagNameSinceSecond).DefValue)
	assert.Equal("-1", cmd.Flags().Lookup(common.FlagNameTailLines).DefValue)
	assert.Equal("-1", cmd.Flags().Lookup(common.FlagNameLimitBytes).DefValue)
}

func TestAddQueryParams(t *testing.T) {
	assert := assert.New(t)

	baseURL, err := url.Parse("http://localhost:8080/api")
	assert.NoError(err)

	restReq := rest.NewRequestWithClient(baseURL, "v1", rest.ClientContentConfig{}, http.DefaultClient)

	opts := &PodLogsOptions{
		Container:                    "app-container",
		Follow:                       true,
		Previous:                     true,
		SinceSecond:                  "60",
		SinceTime:                    "2026-10-25T00:00:00Z",
		Timestamps:                   true,
		TailLines:                    "100",
		LimitBytes:                   "2048",
		InsecureSkipTLSVerifyBackend: true,
	}

	req := addQueryParams(restReq, opts)
	query := req.URL().Query()

	assert.Equal("app-container", query.Get("container"))
	assert.Equal("true", query.Get("follow"))
	assert.Equal("true", query.Get("previous"))
	assert.Equal("60", query.Get("sinceSeconds"))
	assert.Equal("2026-10-25T00:00:00Z", query.Get("sinceTime"))
	assert.Equal("true", query.Get("timestamps"))
	assert.Equal("100", query.Get("tailLines"))
	assert.Equal("2048", query.Get("limitBytes"))
	assert.Equal("true", query.Get("insecureSkipTLSVerifyBackend"))
}

func TestAddQueryParams_Defaults(t *testing.T) {
	assert := assert.New(t)

	baseURL, err := url.Parse("http://localhost:8080/api")
	assert.NoError(err)

	restReq := rest.NewRequestWithClient(baseURL, "v1", rest.ClientContentConfig{}, http.DefaultClient)

	opts := &PodLogsOptions{
		SinceSecond: "0",
		TailLines:   "-1",
		LimitBytes:  "-1",
	}

	req := addQueryParams(restReq, opts)
	query := req.URL().Query()

	assert.Empty(query.Get("sinceSeconds"))
	assert.Empty(query.Get("tailLines"))
	assert.Empty(query.Get("limitBytes"))
	assert.Empty(query.Get("follow"))
	assert.Empty(query.Get("previous"))
}
