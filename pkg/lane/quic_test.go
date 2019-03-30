package lane

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/packer"
	"github.com/kubeedge/viaduct/pkg/translator"
	"github.com/lucas-clemente/quic-go"
	"reflect"
	"testing"
	"time"
)

var err1 error

// TestNewQuicLane is function to test NewQuicLane().
func TestNewQuicLane(t *testing.T) {
	initMocks(t)
	tests := []struct {
		name string
		van interface{}
		want *QuicLane
	}{
		{
			name:"NewQuickLaneTest-StreamObject",
			van:mockStream,
			want:&QuicLane{stream:mockStream},
		},
		{
			name:"NewQuickLaneTest-EmptyObject",
			van:"",
			want:nil,
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

// TestQuicLane_ReadMessage is function to test ReadMessage().
func TestQuicLane_ReadMessage(t *testing.T) {
	var msg  = model.Message{Content:"message"}
	bytesMsg, _ :=translator.NewTran().Encode(&msg)
	header := packer.PackageHeader{Version:0011,PayloadLen:(uint32(len(bytesMsg)))}
	headerBuffer := make([]byte, 0)
	header.Pack(&headerBuffer)
	err1=errors.New("Error")
	tests := []struct {
		name    string
		stream        quic.Stream
		msg *model.Message
		wantErr error
	}{
		{
			name:"Test-FailureCase",
			stream:mockStream,
			msg:&model.Message{},
			wantErr:err1,
		},
		{
			name:"Test-SuccessCase",
			stream:mockStream,
			msg:&model.Message{Content:"message"},
			wantErr:nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				stream:        tt.stream,
			}
			if ( tt.name == "Test-FailureCase") {
				mockStream.EXPECT().Read(gomock.Any()).Return(0, err1).Times(1)
			}
			if ( tt.name == "Test-SuccessCase") {
			callFirst:=	mockStream.EXPECT().Read(gomock.Any()).SetArg(0, headerBuffer).Return(packer.HeaderSize,nil).Times(1)
			mockStream.EXPECT().Read(gomock.Any()).SetArg(0, bytesMsg).Return(len(bytesMsg), nil).Times(1).After(callFirst)
			}
			if err := l.ReadMessage(tt.msg); (err != tt.wantErr) {
				t.Errorf("QuicLane.ReadMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestQuicLane_WriteMessage is function to test WriteMessage().
func TestQuicLane_WriteMessage(t *testing.T) {
	tests := []struct {
		name    string
		stream        quic.Stream
		msg *model.Message
		wantErr bool
	}{
		{
			name:"TestWriteMessage",
			stream:mockStream,
			msg:&model.Message{Content:"message"},
			wantErr:false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				stream:        tt.stream,
			}
			mockStream.EXPECT().Write(gomock.Any()).Return(1,nil).AnyTimes()
			if err := l.WriteMessage(tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("QuicLane.WriteMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestQuicLane_SetReadDeadline is function to test SetReadDeadline().
func TestQuicLane_SetReadDeadline(t *testing.T) {
	tests := []struct {
		name    string
		readDeadline  time.Time
		stream        quic.Stream
		t time.Time
		wantErr bool
	}{
		{
			name:"TestSetReadDeadline",
			readDeadline:time.Time{},
			t:time.Time{},
			wantErr:false,
			stream:mockStream,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &QuicLane{
				readDeadline:  tt.readDeadline,
				stream:        tt.stream,
			}
			mockStream.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).Times(1)
			if err := l.SetReadDeadline(tt.t); (err != nil) != tt.wantErr {
				t.Errorf("QuicLane.SetReadDeadline() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestQuicLane_SetWriteDeadline is function to test SetWriteDeadline().
func TestQuicLane_SetWriteDeadline(t *testing.T) {
	tests := []struct {
		name    string
		writeDeadline time.Time
		stream        quic.Stream
		t time.Time
		wantErr bool
	}{
		{
			name:"Test1",
			writeDeadline:time.Time{},
			t:time.Time{},
			wantErr:false,
			stream:mockStream,
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
