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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	fakerest "k8s.io/client-go/rest/fake"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type fakeCoreV1Client struct {
	corev1.CoreV1Interface
	restClient rest.Interface
}

func (c *fakeCoreV1Client) RESTClient() rest.Interface {
	return c.restClient
}

type fakeClientSet struct {
	kubernetes.Interface
	coreV1 corev1.CoreV1Interface
}

func (c *fakeClientSet) CoreV1() corev1.CoreV1Interface {
	return c.coreV1
}

func newTestFakeClientSet(restClient rest.Interface) kubernetes.Interface {
	return &fakeClientSet{
		coreV1: &fakeCoreV1Client{
			restClient: restClient,
		},
	}
}

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

	err := cmd.RunE(cmd, []string{})
	assert.Error(err)
	assert.Equal("no pod specified for logs", err.Error())
}

func TestNewLogsPodOpts(t *testing.T) {
	assert := assert.New(t)

	opts := NewLogsPodOpts()
	assert.NotNil(opts)
	assert.Equal("default", opts.Namespace)
}

func TestGetPodLogs_NoConfig(t *testing.T) {
	assert := assert.New(t)
	opts := NewLogsPodOpts()
	err := opts.getPodLogs([]string{"test-pod"})
	assert.Error(err)
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

func TestLogsPod_Follow(t *testing.T) {
	assert := assert.New(t)

	var requestCount int32
	fakeHTTPClient := fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&requestCount, 1)
		assert.Equal(http.MethodGet, req.Method)
		assert.Equal("true", req.URL.Query().Get("follow"))
		assert.Contains(req.URL.Path, "pods/test-pod/log")

		// Simulate chunked/text log stream from metaserver
		pipeReader, pipeWriter := io.Pipe()
		go func() {
			_, _ = pipeWriter.Write([]byte("streaming log line 1\nstreaming log line 2\n"))
			_ = pipeWriter.Close()
		}()

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       pipeReader,
		}, nil
	})

	fakeRESTClient := &fakerest.RESTClient{
		Client:               fakeHTTPClient,
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
	}

	clientSet := newTestFakeClientSet(fakeRESTClient)
	opts := &PodLogsOptions{
		Namespace: "default",
		Follow:    true,
	}

	// Capture stdout
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(err)
	os.Stdout = w

	resp, err := logsPod(context.Background(), clientSet, "test-pod", opts)

	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	assert.NoError(err)
	assert.NotNil(resp)
	assert.Empty(resp.LogMessages)
	assert.Empty(resp.ErrMessages)
	assert.Equal("streaming log line 1\nstreaming log line 2\n", string(out))

	// Verify only a single request was sent (no duplicate request)
	assert.Equal(int32(1), atomic.LoadInt32(&requestCount), "expected exactly 1 HTTP request, but got duplicate request")
}

func TestLogsPod_Follow_ContextCancel(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled before call

	fakeHTTPClient := fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	})

	fakeRESTClient := &fakerest.RESTClient{
		Client:               fakeHTTPClient,
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
	}

	clientSet := newTestFakeClientSet(fakeRESTClient)
	opts := &PodLogsOptions{
		Namespace: "default",
		Follow:    true,
	}

	resp, err := logsPod(ctx, clientSet, "test-pod", opts)
	assert.Error(err)
	assert.Nil(resp)
}

func TestLogsPod_Follow_StreamError(t *testing.T) {
	assert := assert.New(t)

	fakeHTTPClient := fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("connection refused")
	})

	fakeRESTClient := &fakerest.RESTClient{
		Client:               fakeHTTPClient,
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
	}

	clientSet := newTestFakeClientSet(fakeRESTClient)
	opts := &PodLogsOptions{
		Namespace: "default",
		Follow:    true,
	}

	resp, err := logsPod(context.Background(), clientSet, "test-pod", opts)
	assert.Error(err)
	assert.Nil(resp)
	assert.Contains(err.Error(), "connection refused")
}

func TestLogsPod_NoFollow_Success(t *testing.T) {
	assert := assert.New(t)

	expectedResponse := types.LogsResponse{
		LogMessages: []string{"first log line", "second log line"},
		ErrMessages: []string{},
	}
	rawJSON, err := json.Marshal(expectedResponse)
	assert.NoError(err)

	var requestCount int32
	fakeHTTPClient := fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&requestCount, 1)
		assert.Equal(http.MethodGet, req.Method)
		assert.Empty(req.URL.Query().Get("follow"))

		pipeReader, pipeWriter := io.Pipe()
		go func() {
			_, _ = pipeWriter.Write(rawJSON)
			_ = pipeWriter.Close()
		}()

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       pipeReader,
		}, nil
	})

	fakeRESTClient := &fakerest.RESTClient{
		Client:               fakeHTTPClient,
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
	}

	clientSet := newTestFakeClientSet(fakeRESTClient)
	opts := &PodLogsOptions{
		Namespace: "default",
		Follow:    false,
	}

	resp, err := logsPod(context.Background(), clientSet, "test-pod", opts)
	assert.NoError(err)
	assert.NotNil(resp)
	assert.Equal(expectedResponse.LogMessages, resp.LogMessages)
	assert.Equal(int32(1), atomic.LoadInt32(&requestCount))
}

func TestLogsPod_NoFollow_Non200Status(t *testing.T) {
	assert := assert.New(t)

	fakeHTTPClient := fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		pipeReader, pipeWriter := io.Pipe()
		go func() {
			_, _ = pipeWriter.Write([]byte("not found"))
			_ = pipeWriter.Close()
		}()

		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       pipeReader,
		}, nil
	})

	fakeRESTClient := &fakerest.RESTClient{
		Client:               fakeHTTPClient,
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
	}

	clientSet := newTestFakeClientSet(fakeRESTClient)
	opts := &PodLogsOptions{
		Namespace: "default",
		Follow:    false,
	}

	resp, err := logsPod(context.Background(), clientSet, "test-pod", opts)
	assert.Error(err)
	assert.Nil(resp)
}

func TestLogsPod_NoFollow_InvalidJSON(t *testing.T) {
	assert := assert.New(t)

	fakeHTTPClient := fakerest.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
		pipeReader, pipeWriter := io.Pipe()
		go func() {
			_, _ = pipeWriter.Write([]byte("invalid json output"))
			_ = pipeWriter.Close()
		}()

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       pipeReader,
		}, nil
	})

	fakeRESTClient := &fakerest.RESTClient{
		Client:               fakeHTTPClient,
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
	}

	clientSet := newTestFakeClientSet(fakeRESTClient)
	opts := &PodLogsOptions{
		Namespace: "default",
		Follow:    false,
	}

	resp, err := logsPod(context.Background(), clientSet, "test-pod", opts)
	assert.Error(err)
	assert.Nil(resp)
}
