package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	v1 "k8s.io/api/core/v1"
)

func GetSecrets(apiserver, label string) (v1.SecretList, error) {
	var secrets v1.SecretList
	var resp *http.Response
	var err error

	if len(label) > 0 {
		resp, err = SendHTTPRequest(http.MethodGet, apiserver+podLabelSelector+label)
	} else {
		resp, err = SendHTTPRequest(http.MethodGet, apiserver)
	}
	if err != nil {
		Fatalf("Frame HTTP request failed: %v", err)
		return secrets, nil
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Fatalf("HTTP Response reading has failed: %v", err)
		return secrets, nil
	}
	err = json.Unmarshal(contents, &secrets)
	if err != nil {
		Fatalf("Unmarshal HTTP Response has failed: %v", err)
		return secrets, nil
	}
	return secrets, nil
}
