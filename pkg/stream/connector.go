package stream

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type Connector interface {
	Bytes() ([]byte, error)
	Dial() error
}

type LogsConnectorInfo struct {
	Url    url.URL     `json:"url"`
	Header http.Header `json:"header"`
}

func (l *LogsConnectorInfo) Bytes() ([]byte, error) {
	return json.Marshal(l)
}

func (l *LogsConnectorInfo) Dial() error {
	/*
		dialer := websocket.Dialer{
			HandshakeTimeout: time.Second * 2,
		}
		con, _, err := dialer.Dial(l.Url.String(), l.Header)
	*/
	return nil
}

var _ Connector = &LogsConnectorInfo{}
