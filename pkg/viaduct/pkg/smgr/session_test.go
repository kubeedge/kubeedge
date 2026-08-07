package smgr

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/viaduct/mocks"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

func TestOpenStreamSyncSetsAndClearsWriteDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)

	mockSess.EXPECT().OpenStreamSync().Return(mockStream, nil)
	mockStream.EXPECT().SetWriteDeadline(gomock.Not(time.Time{})).Return(nil)
	mockStream.EXPECT().Write([]byte(api.UseTypeMessage)).Return(len(api.UseTypeMessage), nil)
	mockStream.EXPECT().SetWriteDeadline(time.Time{}).Return(nil)

	s := &Session{Sess: mockSess, HandshakeTimeout: 2 * time.Second}
	stream, err := s.OpenStreamSync(api.UseTypeMessage)
	assert.NoError(t, err)
	assert.Equal(t, api.UseTypeMessage, stream.UseType)
}

func TestAcceptStreamSetsAndClearsReadDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)

	mockSess.EXPECT().AcceptStream().Return(mockStream, nil)
	mockStream.EXPECT().SetReadDeadline(gomock.Not(time.Time{})).Return(nil)
	mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
		copy(p, api.UseTypeMessage)
		return len(api.UseTypeMessage), nil
	})
	mockStream.EXPECT().SetReadDeadline(time.Time{}).Return(nil)

	s := &Session{Sess: mockSess}
	stream, err := s.AcceptStream()
	assert.NoError(t, err)
	assert.Equal(t, api.UseTypeMessage, stream.UseType)
}

func TestHandshakeTimeoutDefault(t *testing.T) {
	s := &Session{}
	assert.Equal(t, DefaultStreamHandshakeTimeout, s.handshakeTimeout())

	s.HandshakeTimeout = 10 * time.Second
	assert.Equal(t, 10*time.Second, s.handshakeTimeout())
}
