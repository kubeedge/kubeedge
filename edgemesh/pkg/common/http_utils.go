package common

import (
	"io/ioutil"
	"net/http"
	"strings"
)

func httpMethods() (methods []string) {
	methods = []string{"GET", "HEAD", "POST", "OPTIONS", "PUT", "DELETE", "TRACE", "CONNECT"}
	return
}

func IsHTTPRequest(s string) bool {
	methods := httpMethods()
	for _, method := range methods {
		if strings.HasPrefix(s, method) {
			return true
		}
	}
	return false
}

func HTTPResponseToStr(resp *http.Response) string {
	respString := resp.Proto + " " + resp.Status + "\n"
	for key, values := range resp.Header {
		respString += key + ": "
		for _, v := range values {
			respString += v + ", "
		}
		respString = respString[0 : len(respString)-2]
		respString += "\n"
	}
	b, _ := ioutil.ReadAll(resp.Body)
	respString += "\n" + string(b)
	return respString
}
