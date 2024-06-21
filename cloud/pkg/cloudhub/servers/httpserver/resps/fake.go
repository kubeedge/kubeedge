package resps

import "net/http"

type FakeResponseWriter struct {
	payload []byte
	code    int
}

func (f FakeResponseWriter) Header() http.Header {
	return make(map[string][]string)
}

func (f *FakeResponseWriter) Write(payload []byte) (int, error) {
	f.payload = payload
	return -1, nil
}

func (f *FakeResponseWriter) WriteHeader(code int) {
	f.code = code
}
