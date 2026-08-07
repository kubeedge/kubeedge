package server

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/viaduct/mocks"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func TestNewQuicServer(t *testing.T) {
	opts := Options{
		Addr:             "127.0.0.1:9090",
		HandshakeTimeout: 5 * time.Second,
	}
	exOpts := api.QuicServerOption{
		MaxIncomingStreams: 10,
	}

	srv := NewQuicServer(opts, exOpts)
	assert.NotNil(t, srv)
	assert.Equal(t, opts, srv.options)
	assert.Equal(t, exOpts, srv.exOpts)

	// Test invalid extend options type panic
	assert.Panics(t, func() {
		NewQuicServer(opts, "invalid_ex_opts")
	})
}

func TestQuicServer_AcceptControlStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	opts := Options{Addr: "127.0.0.1:9090"}
	exOpts := api.QuicServerOption{MaxIncomingStreams: 10}
	srv := NewQuicServer(opts, exOpts)

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)

	mockSess.EXPECT().AcceptStream(gomock.Any()).Return(mockStream, nil)

	st := srv.acceptControlStream(mockSess)
	assert.NotNil(t, st)
	assert.Equal(t, mockStream, st)
}

func TestQuicServer_Close(t *testing.T) {
	opts := Options{Addr: "127.0.0.1:9090"}
	exOpts := api.QuicServerOption{MaxIncomingStreams: 10}
	srv := NewQuicServer(opts, exOpts)

	// Close when listener is nil
	err := srv.Close()
	assert.NoError(t, err)
}
