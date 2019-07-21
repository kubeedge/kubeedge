package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chassis/go-chassis/core/config"
	"github.com/go-chassis/go-chassis/core/config/model"
	"github.com/go-chassis/go-chassis/core/fault"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/lager"
)

// constant for fault handler name
const (
	FaultHandlerName = "fault-inject"
)

// FaultHandler handler
type FaultHandler struct{}

// newFaultHandler fault handle gives the object of FaultHandler
func newFaultHandler() Handler {
	return &FaultHandler{}
}

// Name function returns fault-inject string
func (rl *FaultHandler) Name() string {
	return "fault-inject"
}

// Handle is to handle the API
func (rl *FaultHandler) Handle(chain *Chain, inv *invocation.Invocation, cb invocation.ResponseCallBack) {

	faultStruct := GetFaultConfig(inv.Protocol, inv.MicroServiceName, inv.SchemaID, inv.OperationID)
	faultConfig := model.FaultProtocolStruct{}
	faultConfig.Fault = make(map[string]model.Fault)
	faultConfig.Fault[inv.Protocol] = faultStruct

	faultInject, ok := fault.Injectors[inv.Protocol]
	r := &invocation.Response{}
	if !ok {
		lager.Logger.Warnf("fault injection doesn't support for protocol ", errors.New(inv.Protocol))
		r.Err = nil
		cb(r)
		return
	}

	faultValue := faultConfig.Fault[inv.Protocol]
	err := faultInject(faultValue, inv)
	if err != nil {
		if strings.Contains(err.Error(), "injecting abort") {
			switch inv.Reply.(type) {
			case *http.Response:
				resp := inv.Reply.(*http.Response)
				resp.StatusCode = faultConfig.Fault[inv.Protocol].Abort.HTTPStatus
			}
			r.Status = faultConfig.Fault[inv.Protocol].Abort.HTTPStatus
		} else {
			switch inv.Reply.(type) {
			case *http.Response:
				resp := inv.Reply.(*http.Response)
				resp.StatusCode = http.StatusBadRequest
			}
			r.Status = http.StatusBadRequest
		}

		r.Err = fault.FaultError{Message: err.Error()}
		cb(r)
		return
	}

	chain.Next(inv, func(r *invocation.Response) error {
		return cb(r)
	})
}

// GetFaultConfig get faultconfig
func GetFaultConfig(protocol, microServiceName, schemaID, operationID string) model.Fault {

	faultStruct := model.Fault{}
	faultStruct.Abort.Percent = config.GetAbortPercent(protocol, microServiceName, schemaID, operationID)
	faultStruct.Abort.HTTPStatus = config.GetAbortStatus(protocol, microServiceName, schemaID, operationID)
	faultStruct.Delay.Percent = config.GetDelayPercent(protocol, microServiceName, schemaID, operationID)
	faultStruct.Delay.FixedDelay = config.GetFixedDelay(protocol, microServiceName, schemaID, operationID)

	return faultStruct
}
