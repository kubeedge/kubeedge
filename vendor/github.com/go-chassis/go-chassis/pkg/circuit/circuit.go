package circuit

import (
	"errors"
	"fmt"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/third_party/forked/afex/hystrix-go/hystrix"
	"github.com/go-mesh/openlogging"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	//ReturnNil is build in fallback name
	ReturnNil = "returnnull"
	//ReturnErr is build in fallback name
	ReturnErr = "throwexception"
)

var fallbackFuncMap = make(map[string]Fallback)

//ErrFallbackNotExists happens if fallback implementation does not exist
var ErrFallbackNotExists = errors.New("fallback func does not exist")

//Fallback defines how to return response if remote call fails.
//a implementation should return a closure to handle the error.
//in this closure, if you fallback logic should handle the original error,
//you can return a fallback error to replace the original error
//you can assemble invocation.Response on demand
//in summary the closure defines, "if err happens, how to handle it".
type Fallback func(inv *invocation.Invocation, finish chan *invocation.Response) func(error) error

//Init init functions
func Init() {
	fallbackFuncMap[ReturnErr] = FallbackErr
	fallbackFuncMap[ReturnNil] = FallbackNil
}

//RegisterFallback register custom logic
func RegisterFallback(name string, f Fallback) {
	fallbackFuncMap[name] = f
}

//GetFallback return function
func GetFallback(name string) (Fallback, error) {
	f, ok := fallbackFuncMap[name]
	if !ok {
		return nil, ErrFallbackNotExists
	}
	return f, nil
}

//FallbackNil return empty response
func FallbackNil(inv *invocation.Invocation, finish chan *invocation.Response) func(error) error {
	return func(err error) error {
		// if err is type of hystrix error, return a new response
		if err.Error() == hystrix.ErrForceFallback.Error() || err.Error() == hystrix.ErrCircuitOpen.Error() ||
			err.Error() == hystrix.ErrMaxConcurrency.Error() {
			// isolation happened, so lead to callback
			openlogging.GetLogger().Errorf(fmt.Sprintf("fallback for %s:%s:%s, error [%s]",
				inv.MicroServiceName, inv.SchemaID, inv.OperationID,
				err.Error()))
			resp := &invocation.Response{}
			switch inv.Reply.(type) {
			case *http.Response:
				resp := inv.Reply.(*http.Response)
				resp.StatusCode = http.StatusOK
				//make sure body is empty
				if resp.Body != nil {
					io.Copy(ioutil.Discard, resp.Body)
					resp.Body.Close()
				}
			}
			select {
			case finish <- resp:
			default:
			}
			return nil //no need to return error
		}
		// call back success
		return nil
	}
}

//FallbackErr set err in response
func FallbackErr(inv *invocation.Invocation, finish chan *invocation.Response) func(error) error {
	return func(err error) error {
		// if err is type of hystrix error, return a new response
		resp := &invocation.Response{}
		if err.Error() == hystrix.ErrForceFallback.Error() || err.Error() == hystrix.ErrCircuitOpen.Error() {
			// isolation happened, so lead to callback
			openlogging.GetLogger().Errorf(fmt.Sprintf("fallback for %s:%s:%s, error [%s]",
				inv.MicroServiceName, inv.SchemaID, inv.OperationID,
				err.Error()))
			resp.Err = hystrix.CircuitError{
				Message: fmt.Sprintf("API %s:%s:%s is isolated because of error: %s", inv.MicroServiceName,
					inv.SchemaID, inv.OperationID, err.Error()),
			}
		} else if err.Error() == hystrix.ErrMaxConcurrency.Error() {
			// isolation happened, so lead to callback
			openlogging.GetLogger().Errorf(fmt.Sprintf("fallback for %s:%s:%s, error [%s]",
				inv.MicroServiceName, inv.SchemaID, inv.OperationID,
				err.Error()))
			resp.Err = hystrix.CircuitError{
				Message: fmt.Sprintf("API %s:%s:%s is reject because of error: %s", inv.MicroServiceName,
					inv.SchemaID, inv.OperationID, err.Error()),
			}

		} else {
			//do nothing, just give original error
			return nil
		}
		switch inv.Reply.(type) {
		case *http.Response:
			resp := inv.Reply.(*http.Response)
			resp.StatusCode = http.StatusInternalServerError
			//make sure body is empty
			if resp.Body != nil {
				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()
			}
		}
		select {
		case finish <- resp:
		default:
		}
		return nil //no need to return error

	}
}
