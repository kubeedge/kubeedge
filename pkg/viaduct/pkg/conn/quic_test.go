package conn

import (
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/viaduct/mocks"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/lane"
)

type dummyAddr string

func (d dummyAddr) Network() string { return "quic" }
func (d dummyAddr) String() string  { return string(d) }

func TestNewQuicConn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)
	quicConn := NewQuicConn(&ConnectionOptions{
		Base:      mockSess,
		CtrlLane:  lane.NewLane(api.ProtocolTypeQuic, mockStream),
		State:     &ConnectionState{State: api.StatConnected},
		ConnUse:   api.UseTypeShare,
		AutoRoute: true,
	})

	assert.NotNil(t, quicConn)
	assert.Equal(t, api.StatConnected, quicConn.ConnectionState().State)
}

func TestQuicConn_LocalRemoteAddr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)
	quicConn := NewQuicConn(&ConnectionOptions{
		Base:     mockSess,
		CtrlLane: lane.NewLane(api.ProtocolTypeQuic, mockStream),
		State:    &ConnectionState{State: api.StatConnected},
	})

	lAddr := dummyAddr("127.0.0.1:8080")
	rAddr := dummyAddr("127.0.0.1:9090")

	mockSess.EXPECT().LocalAddr().Return(lAddr)
	mockSess.EXPECT().RemoteAddr().Return(rAddr)

	assert.Equal(t, net.Addr(lAddr), quicConn.LocalAddr())
	assert.Equal(t, net.Addr(rAddr), quicConn.RemoteAddr())
}

func TestQuicConn_SetDeadlines(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := mocks.NewMockStream(ctrl)
	quicConn := NewQuicConn(&ConnectionOptions{
		Base:     mocks.NewMockSession(ctrl),
		CtrlLane: lane.NewLane(api.ProtocolTypeQuic, mockStream),
		State:    &ConnectionState{State: api.StatConnected},
	})

	now := time.Now()
	err := quicConn.SetReadDeadline(now)
	assert.NoError(t, err)
	assert.Equal(t, now, quicConn.readDeadline)

	err = quicConn.SetWriteDeadline(now)
	assert.NoError(t, err)
	assert.Equal(t, now, quicConn.writeDeadline)
}

func TestQuicConn_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSess := mocks.NewMockSession(ctrl)
	mockStream := mocks.NewMockStream(ctrl)
	quicConn := NewQuicConn(&ConnectionOptions{
		Base:     mockSess,
		CtrlLane: lane.NewLane(api.ProtocolTypeQuic, mockStream),
		State:    &ConnectionState{State: api.StatConnected},
	})

	mockSess.EXPECT().CloseWithError(gomock.Any(), gomock.Any()).Return(nil)

	err := quicConn.Close()
	assert.NoError(t, err)
	assert.Equal(t, api.StatDisconnected, quicConn.ConnectionState().State)
}
