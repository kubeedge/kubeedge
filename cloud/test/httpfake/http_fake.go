package httpfake

import "net/http"

type ResponseWriter struct {
	HTTPHeader http.Header
	Status     int
	Body       []byte
}

func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		HTTPHeader: make(http.Header),
	}
}

func (f *ResponseWriter) Header() http.Header {
	return f.HTTPHeader
}

func (f *ResponseWriter) Write(b []byte) (int, error) {
	f.Body = b
	return len(b), nil
}

func (f *ResponseWriter) WriteHeader(statusCode int) {
	f.Status = statusCode
}
