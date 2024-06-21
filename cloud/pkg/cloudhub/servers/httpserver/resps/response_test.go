package resps

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	cases := []struct {
		name    string
		code    int
		message string
		err     error
	}{
		{
			name:    "default error code",
			message: "test message",
		},
		{
			name:    "specified error code",
			code:    http.StatusBadRequest,
			message: "bad request",
		},
		{
			name: "specified error",
			err:  errors.New("new error"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var wantCode int
			if c.code == 0 {
				wantCode = http.StatusInternalServerError
			} else {
				wantCode = c.code
			}
			w := new(FakeResponseWriter)
			var wantMsg string
			if c.err != nil {
				wantMsg = c.err.Error()
				Error(w, c.code, c.err)
			} else {
				wantMsg = c.message
				ErrorMessage(w, c.code, c.message)
			}
			if w.code != wantCode {
				t.Fatalf("want status code is %d, actual is %d", wantCode, w.code)
			}
			if msg := string(w.payload); msg != wantMsg {
				t.Fatalf("want error message is %s, actual is %s", wantMsg, msg)
			}
		})
	}
}

func TestOK(t *testing.T) {
	payload := []byte("test data")
	w := new(FakeResponseWriter)
	OK(w, payload)
	if w.code != http.StatusOK {
		t.Fatalf("want status code is %d, actual is %d", http.StatusOK, w.code)
	}
	if !reflect.DeepEqual(w.payload, payload) {
		t.Fatalf("want error message is %s, actual is %s", string(payload), string(w.payload))
	}
}
