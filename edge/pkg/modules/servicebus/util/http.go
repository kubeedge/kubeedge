package util

import (
	"net/http"
)

// HTTPRequest is used structure used to unmarshal message content from clous
type HTTPRequest struct {
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

// HTTPResponse is HTTP request's response structure used to send response to cloud
type HTTPResponse struct {
	Header     http.Header `json:"header"`
	StatusCode int         `json:"status_code"`
	Body       []byte      `json:"body"`
}
