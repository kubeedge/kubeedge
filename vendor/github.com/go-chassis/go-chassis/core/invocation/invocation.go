package invocation

import (
	"context"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/pkg/runtime"
	"github.com/go-chassis/go-chassis/pkg/util/tags"
)

// constant values for consumer and provider
const (
	Consumer = iota
	Provider
)

// Response is invocation response struct
type Response struct {
	Status int
	Result interface{}
	Err    error
}

// ResponseCallBack process invocation response
type ResponseCallBack func(*Response) error

//Invocation is the basic struct that used in go sdk to make client and transport layer transparent .
//developer should implements a client which is able to transfer invocation to there own request
//a protocol server should transfer request to invocation and then back to request
type Invocation struct {
	HandlerIndex       int
	Endpoint           string //service's ip and port, it is decided in load balancing
	Protocol           string
	Port               string //Port is the name of a real service port
	SourceServiceID    string
	SourceMicroService string
	MicroServiceName   string //Target micro service name
	SchemaID           string //correspond struct name
	OperationID        string //correspond struct func name
	Args               interface{}
	URLPathFormat      string
	Reply              interface{}
	Ctx                context.Context        //ctx can save protocol headers
	Metadata           map[string]interface{} //local scope data
	RouteTags          utiltags.Tags          //route tags is decided in router handler
	Strategy           string                 //load balancing strategy
	Filters            []string
}

//Reset reset clear a invocation
func (inv *Invocation) Reset() {
	inv.Endpoint = ""
	inv.Protocol = ""
	inv.SourceServiceID = ""
	inv.SourceMicroService = ""
	inv.MicroServiceName = ""
	inv.SchemaID = ""
	inv.OperationID = ""
	inv.Args = nil
	inv.URLPathFormat = ""
	inv.Reply = nil
	inv.Ctx = nil
	inv.Metadata = nil
	inv.RouteTags = utiltags.Tags{}
	inv.Filters = nil
	inv.Strategy = ""

}

// New create invocation, context can not be nil
// if you don't set ContextHeaderKey, then New will init it
func New(ctx context.Context) *Invocation {
	inv := &Invocation{
		SourceServiceID: runtime.ServiceID,
		Ctx:             ctx,
	}
	if inv.Ctx == nil {
		inv.Ctx = context.TODO()
	}
	if inv.Ctx.Value(common.ContextHeaderKey{}) == nil {
		inv.Ctx = context.WithValue(inv.Ctx, common.ContextHeaderKey{}, map[string]string{})
	}
	return inv
}

//SetMetadata local scope params
func (inv *Invocation) SetMetadata(key string, value interface{}) {
	if inv.Metadata == nil {
		inv.Metadata = make(map[string]interface{})
	}
	inv.Metadata[key] = value
}

//SetHeader set headers, the client and server plugins should use them in protocol headers
//it is convenience but has lower performance than you use Headers[k]=v,
// when you have a batch of kv to set
func (inv *Invocation) SetHeader(k, v string) {
	m := inv.Ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	m[k] = v
}

//Headers return a map that protocol plugin should deliver in transport
func (inv *Invocation) Headers() map[string]string {
	return inv.Ctx.Value(common.ContextHeaderKey{}).(map[string]string)
}
