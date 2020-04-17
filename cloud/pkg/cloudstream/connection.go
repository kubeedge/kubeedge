package cloudstream

import (
	"fmt"
	"io"

	"github.com/emicklei/go-restful"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

type ApiServerConnection interface {
	fmt.Stringer
	io.Closer
	SendConnector() error
	WriteToTunnel(m *stream.Message) error
	io.Writer
	Serve() error
	SetID(id uint64)
}

type LogsConnection struct {
	ID      uint64 // 唯一的ID表示，用来生成message 用
	r       *restful.Request
	flush   io.Writer
	session *Session
}

func (l *LogsConnection) SetID(id uint64) {
	l.ID = id
}

func (l *LogsConnection) String() string {
	return "APIServer_LogsConnection"
}

func (l *LogsConnection) Close() error {
	return nil
}

func (l *LogsConnection) SendConnector() error {
	connector := &stream.EdgeLogsConnector{
		Url:    *l.r.Request.URL,
		Header: l.r.Request.Header,
	}
	data, err := connector.Bytes()
	if err != nil {
		return err
	}
	m := stream.NewMessage(l.ID, stream.MessageTypeLogsConnect, data)
	return l.WriteToTunnel(m)
}

func (l *LogsConnection) WriteToTunnel(m *stream.Message) error {
	return l.session.WriteMessageToTunnel(m)
}

func (l *LogsConnection) Write(data []byte) (int, error) {
	return l.flush.Write(data)
}

func (l *LogsConnection) Serve() error {
	// first send connect info
	return l.SendConnector()
}

var _ ApiServerConnection = &LogsConnection{}
