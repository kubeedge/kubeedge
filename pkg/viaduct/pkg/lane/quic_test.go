package lane

import (
	"bytes"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/mocks"
)

func TestNewQuicLane(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := mocks.NewMockStream(ctrl)

	// Test valid quic.Stream
	quicLane := NewQuicLane(mockStream)
	assert.NotNil(t, quicLane)
	assert.Equal(t, mockStream, quicLane.stream)

	// Test bad type of van
	badLane := NewQuicLane("invalid_stream_type")
	assert.Nil(t, badLane)
}

func TestQuicLane_ReadWrite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := mocks.NewMockStream(ctrl)
	quicLane := NewQuicLane(mockStream)

	inputData := []byte("hello quic lane")
	mockStream.EXPECT().Write(inputData).Return(len(inputData), nil)

	n, err := quicLane.Write(inputData)
	assert.NoError(t, err)
	assert.Equal(t, len(inputData), n)

	mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
		return copy(b, inputData), nil
	})

	readBuf := make([]byte, len(inputData))
	readN, readErr := quicLane.Read(readBuf)
	assert.NoError(t, readErr)
	assert.Equal(t, len(inputData), readN)
	assert.Equal(t, inputData, readBuf)
}

func TestQuicLane_Deadlines(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := mocks.NewMockStream(ctrl)
	quicLane := NewQuicLane(mockStream)

	now := time.Now()

	mockStream.EXPECT().SetReadDeadline(now).Return(nil)
	err := quicLane.SetReadDeadline(now)
	assert.NoError(t, err)
	assert.Equal(t, now, quicLane.readDeadline)

	mockStream.EXPECT().SetWriteDeadline(now).Return(nil)
	err = quicLane.SetWriteDeadline(now)
	assert.NoError(t, err)
	assert.Equal(t, now, quicLane.writeDeadline)
}

func TestQuicLane_ReadWriteMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStream := mocks.NewMockStream(ctrl)
	quicLane := NewQuicLane(mockStream)

	msg := model.NewMessage("").BuildHeader("msg_id", "parent_id", 100).FillBody("test body")

	var writeBuffer bytes.Buffer
	mockStream.EXPECT().Write(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
		return writeBuffer.Write(p)
	}).AnyTimes()

	err := quicLane.WriteMessage(msg)
	assert.NoError(t, err)

	encodedBytes := writeBuffer.Bytes()
	mockStream.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
		if len(encodedBytes) == 0 {
			return 0, nil
		}
		n := copy(p, encodedBytes)
		encodedBytes = encodedBytes[n:]
		return n, nil
	}).AnyTimes()

	var readMsg model.Message
	err = quicLane.ReadMessage(&readMsg)
	assert.NoError(t, err)
	assert.Equal(t, msg.GetID(), readMsg.GetID())
}
