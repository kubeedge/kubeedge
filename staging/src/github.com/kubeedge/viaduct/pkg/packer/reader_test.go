package packer

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/mocks"
	"github.com/kubeedge/viaduct/pkg/translator"
)

// mockStream is mock of interface Stream.
var mockReader *mocks.MockReader

// initMocks is function to initialize mocks.
func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockReader = mocks.NewMockReader(mockCtrl)
}

// TestNewReader is function to test NewReader().
func TestNewReader(t *testing.T) {
	var reader io.Reader
	tests := []struct {
		name string
		r    io.Reader
		want *Reader
	}{
		{
			name: "NewReaderTest",
			r:    reader,
			want: &Reader{reader: reader},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewReader(tt.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReader() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRead is function to test Read().
func TestRead(t *testing.T) {
	initMocks(t)
	var ioreader io.Reader
	var msg = model.Message{Content: "message"}
	bytesMsg, _ := translator.NewTran().Encode(&msg)
	header := PackageHeader{Version: 0011, PayloadLen: (uint32(len(bytesMsg)))}
	headerBuffer := make([]byte, 0)
	header.Pack(&headerBuffer)
	errorReturn := errors.New("Error")
	tests := []struct {
		name    string
		reader  io.Reader
		times   int
		want    []byte
		wantErr bool
	}{
		{
			name:    "TestRead-Failure Case",
			reader:  ioreader,
			times:   0,
			wantErr: true,
		},
		{
			name:    "TestRead-Success Case",
			reader:  mockReader,
			times:   1,
			wantErr: false,
			want:    bytesMsg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reader{
				reader: tt.reader,
			}
			callFirst := mockReader.EXPECT().Read(gomock.Any()).SetArg(0, headerBuffer).Return(HeaderSize, nil).Times(tt.times)
			mockReader.EXPECT().Read(gomock.Any()).SetArg(0, bytesMsg).Return(len(bytesMsg), nil).Times(tt.times).After(callFirst)
			got, err := r.Read()
			if (err != nil) != tt.wantErr {
				t.Errorf("Reader.Read() error = %v, wantErr %v", errorReturn, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reader.Read() = %v, want %v", got, tt.want)
			}
		})
	}
}
