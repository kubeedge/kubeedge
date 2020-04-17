package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type EdgeConnector interface {
	Bytes() ([]byte, error)
	Serve(tunnel *websocket.Conn) error
	fmt.Stringer
}

type EdgeLogsConnector struct {
	Url    url.URL     `json:"url"`
	Header http.Header `json:"header"`
}

func (l *EdgeLogsConnector) Bytes() ([]byte, error) {
	return json.Marshal(l)
}

func (l *EdgeLogsConnector) String() string {
	return "EDGE_LOGS_CONNECTOR"
}

func (l *EdgeLogsConnector) Serve(tunnel *websocket.Conn) error {

	return nil
}

func init() {
	var _ EdgeConnector = &EdgeLogsConnector{}
}
