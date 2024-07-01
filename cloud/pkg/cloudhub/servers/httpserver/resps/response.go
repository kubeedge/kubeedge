package resps

import (
	"net/http"

	"k8s.io/klog/v2"
)

func Error(w http.ResponseWriter, code int, err error) {
	ErrorMessage(w, code, err.Error())
}

func ErrorMessage(w http.ResponseWriter, code int, msg string) {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		klog.Errorf("failed to write a error messge to the response, err: %v", err)
	}
}

func OK(w http.ResponseWriter, body []byte) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		klog.Errorf("failed to write a payload to the response, err: %v", err)
	}
}
