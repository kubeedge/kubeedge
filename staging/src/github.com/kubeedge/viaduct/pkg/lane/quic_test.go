package lane

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lucas-clemente/quic-go"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/packer"
	"github.com/kubeedge/viaduct/pkg/translator"
)

var errorReturn error

// TestNewQuicLane is function to test NewQuicLane().
func TestNewQuicLane(t *testing.T) {
	initMocks(t)
	tests := []struct {
		name string
		van  interface{}
		want *QuicLane
	}{
		{
			name: "NewQuickLaneTest-StreamObject",
			van:  mockStream,
			want: &QuicLane{stream: mockStream},
		},
		{
			name: "NewQuickLaneTest-EmptyObject",
			van:  "",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewQuicLane(tt.van); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewQuicLane() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestQuicLaneReadMessage is function to test ReadMessage().
func TestQuicLaneReadMessage(t *testing.T) {
	var msg = model.Message{Content: "message"}
	bytesMsg, _ := translator.NewTran().Encode(&msg)
	header := packer.PackageHeader{Version: 0011, PayloadLen: (uint32(len(bytesMsg)))}
	headerBuffer := make([]byte, 0)
	header.Pack(&headerBuffer)
	errorReturn = errors.New("Error")
	tests := []struct {
		name         string
		stream       quic.Stream
		msg          *model.Message
		failureTimes int
		successTimes int
		wantErr      error
	}{
		{
			name:         "Test-FailureCase",
			stream:       mockStream,
			msg:          &model.Message{},
			successTimes: 0,
			failureTimes: 1,
			wantErr:      errorReturn,
		},
		{
			name:         "Test-SuccessCase",
			stream:       mockStream,
			msg:          &model.Message{Content: "message"},
			wantErr:      nil,
			successTimes: 1,
			failureTimes: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				stream: tt.stream,
			}
			mockStream.EXPECT().Read(gomock.Any()).Return(0, errorReturn).Times(tt.failureTimes)
			callFirst := mockStream.EXPECT().Read(gomock.Any()).SetArg(0, headerBuffer).Return(packer.HeaderSize, nil).Times(tt.successTimes)
			mockStream.EXPECT().Read(gomock.Any()).SetArg(0, bytesMsg).Return(len(bytesMsg), nil).Times(tt.successTimes).After(callFirst)
			if err := l.ReadMessage(tt.msg); err != tt.wantErr {
				t.Errorf("QuicLane.ReadMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestQuicLaneWriteMessage is function to test WriteMessage().
func TestQuicLaneWriteMessage(t *testing.T) {
	tests := []struct {
		name    string
		stream  quic.Stream
		msg     *model.Message
		wantErr bool
	}{
		{
			name:    "WriteMessageTest",
			stream:  mockStream,
			msg:     &model.Message{Content: "message"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				stream: tt.stream,
			}
			mockStream.EXPECT().Write(gomock.Any()).Return(1, nil).AnyTimes()
			if err := l.WriteMessage(tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("QuicLane.WriteMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestQuicLaneSetReadDeadline is function to test SetReadDeadline().
func TestQuicLaneSetReadDeadline(t *testing.T) {
	tests := []struct {
		name         string
		readDeadline time.Time
		stream       quic.Stream
		t            time.Time
		wantErr      bool
	}{
		{
			name:         "SetReadDeadlineTest",
			readDeadline: time.Time{},
			t:            time.Time{},
			wantErr:      false,
			stream:       mockStream,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				readDeadline: tt.readDeadline,
				stream:       tt.stream,
			}
			mockStream.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).Times(1)
			if err := l.SetReadDeadline(tt.t); (err != nil) != tt.wantErr {
				t.Errorf("QuicLane.SetReadDeadline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestQuicLaneSetWriteDeadline is function to test SetWriteDeadline().
func TestQuicLaneSetWriteDeadline(t *testing.T) {
	tests := []struct {
		name          string
		writeDeadline time.Time
		stream        quic.Stream
		t             time.Time
		wantErr       bool
	}{
		{
			name:          "SetWriteDeadLineTest",
			writeDeadline: time.Time{},
			t:             time.Time{},
			wantErr:       false,
			stream:        mockStream,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				writeDeadline: tt.writeDeadline,
				stream:        tt.stream,
			}
			mockStream.EXPECT().SetWriteDeadline(gomock.Any()).Return(nil).Times(1)
			if err := l.SetWriteDeadline(tt.t); (err != nil) != tt.wantErr {
				t.Errorf("QuicLane.SetWriteDeadline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
