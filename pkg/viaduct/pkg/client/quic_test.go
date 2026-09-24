package client

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/viaduct/mocks"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func TestNewQuicClient(t *testing.T) {
	opts := Options{
		Addr:             "127.0.0.1:9090",
		HandshakeTimeout: 5 * time.Second,
		ConnUse:          api.UseTypeShare,
	}
	exOpts := api.QuicClientOption{
		Header: make(http.Header),
	}

	cli := NewQuicClient(opts, exOpts)
	assert.NotNil(t, cli)
	assert.Equal(t, opts, cli.options)
	assert.Equal(t, exOpts, cli.exOpts)

	// Test invalid extend options type panic
	assert.Panics(t, func() {
		NewQuicClient(opts, "invalid_ex_opts")
	})
}

func TestQuicClient_GetControlLane(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	opts := Options{Addr: "127.0.0.1:9090"}
	exOpts := api.QuicClientOption{Header: make(http.Header)}
	cli := NewQuicClient(opts, exOpts)

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)

	mockSess.EXPECT().OpenStreamSync(gomock.Any()).Return(mockStream, nil)

	err := cli.getControlLane(mockSess)
	assert.NoError(t, err)
	assert.NotNil(t, cli.ctrlLane)

	// Second call should return existing lane immediately without calling OpenStreamSync again
	err = cli.getControlLane(mockSess)
	assert.NoError(t, err)
}
