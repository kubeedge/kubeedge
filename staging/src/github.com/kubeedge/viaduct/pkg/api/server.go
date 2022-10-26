package api

import (
	"net/http"
)

// QuicFilterFunc is the filter function before process connection
type QuicFilterFunc func(connection interface{}, header *http.Header) error

// QuicServerOption is quic server option
// including the extend options when getting server instance
// we can add the essential option into
type QuicServerOption struct {
	// the max incoming stream
	MaxIncomingStreams int64
	// the necessary processing before connected
	Filter QuicFilterFunc
}

// WSFilterFunc is the filter function before upgrading the http to websocket
type WSFilterFunc func(w http.ResponseWriter, r *http.Request) error

// WSServerOption is websocket server option
// you can add the extend options when getting websocket server instance
type WSServerOption struct {
	// the path that the client dialing
	Path string
	// the necessary processing before upgrading
	Filter WSFilterFunc
}
