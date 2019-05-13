package utils

import (
	"encoding/json"
	"net/http"

	"github.com/kubeedge/beehive/pkg/common/log"
)

func DeepCopyHeader(header http.Header) http.Header {
	headerByte, err := json.Marshal(header)
	if err != nil {
		log.LOGGER.Errorf("faile to marshal header, error:%+v", err)
		return nil
	}

	dstHeader := make(http.Header)
	err = json.Unmarshal(headerByte, &dstHeader)
	if err != nil {
		log.LOGGER.Errorf("failed to unmarshal header, error:%+v", err)
		return nil
	}
	return dstHeader
}
