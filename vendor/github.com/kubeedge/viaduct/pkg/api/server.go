package api

import "net/http"

// quic server option
// including the extend options when getting server instance
// we can add the essential option into
type QuicServerOption struct {
	// the max incoming stream
	MaxIncomingStreams int
}

// the filter function before upgrading the http to websocket
type WSFilterFunc func(w http.ResponseWriter, r *http.Request) bool

// websocket server option
// you can add the extend options when getting websocket server instance
type WSServerOption struct {
	// the path that the client dialing
	Path string
	// the necessary processing before upgrading
	Filter WSFilterFunc
}
