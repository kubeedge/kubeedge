package httpserver

import (
	"time"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

// BaseResponse the base response struct of all response
type BaseResponse struct {
	APIVersion string `json:"apiVersion"`
	StatusCode int    `json:"statusCode"`
	TimeStamp  string `json:"timeStamp"`
}

// NewBaseResponse get BaseResponse by statusCode
func NewBaseResponse(statusCode int) *BaseResponse {
	return &BaseResponse{
		APIVersion: APIVersion,
		StatusCode: statusCode,
		TimeStamp:  time.Now().Format(time.RFC3339),
	}
}

type PingResponse struct {
	*BaseResponse
	Message string
}

type DeviceWriteResponse struct {
	*BaseResponse
	Message string
}

type DeviceReadResponse struct {
	*BaseResponse
	Data *common.DataModel
}

type DeviceMethodReadResponse struct {
	*BaseResponse
	Data *common.DataMethod
}

type MetaGetModelResponse struct {
	*BaseResponse
	*common.DeviceModel
}

// DataBaseResponse just for test
type DataBaseResponse struct {
	// TODO DataBase API need to add
	*BaseResponse
	Data []common.DataModel
}
