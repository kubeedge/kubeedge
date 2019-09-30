package utils

import (
	"encoding/json"
	"net/http"

	"k8s.io/klog"
)

func DeepCopyHeader(header http.Header) http.Header {
	headerByte, err := json.Marshal(header)
	if err != nil {
		klog.Errorf("faile to marshal header, error:%+v", err)
		return nil
	}

	dstHeader := make(http.Header)
	err = json.Unmarshal(headerByte, &dstHeader)
	if err != nil {
		klog.Errorf("failed to unmarshal header, error:%+v", err)
		return nil
	}
	return dstHeader
}
