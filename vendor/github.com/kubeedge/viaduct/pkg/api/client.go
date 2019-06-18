package api

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// quic client option
// extend options when you using quic in client
type QuicClientOption struct {
	// send heads after connection completed using control stream
	// TODO:
	Header http.Header
	// the max incoming stream
	MaxIncomingStreams int
}

// you can do some additional processes after successful dialing
type WSClientCallback func(*websocket.Conn, *http.Response)

// websocket client options
// extend options when you using websocket in client
type WSClientOption struct {
	// extend headers that you want to input
	Header http.Header
	// called after dialing
	Callback WSClientCallback
}
