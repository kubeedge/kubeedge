package handler

import (
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/core/lager"
	"github.com/go-chassis/go-chassis/core/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// TracingProviderHandler tracing provider handler
type TracingProviderHandler struct{}

// Handle is to handle the provider tracing related things
func (t *TracingProviderHandler) Handle(chain *Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	var (
		err         error
		wireContext opentracing.SpanContext
		span        opentracing.Span
	)
	// extract span context
	// header stored in context

	switch err {
	case nil:
	case opentracing.ErrSpanContextNotFound:
		lager.Logger.Debug(err.Error())
	default:
		lager.Logger.Errorf("Extract span failed, err [%s]", err.Error())
	}
	wireContext, err = opentracing.GlobalTracer().Extract(opentracing.TextMap, opentracing.TextMapCarrier(i.Headers()))
	if wireContext == nil {
		span = opentracing.StartSpan(i.OperationID, ext.RPCServerOption(wireContext))
	} else {
		// store span in context
		span = opentracing.StartSpan(i.OperationID, opentracing.ChildOf(wireContext), ext.RPCServerOption(wireContext))
	}
	ext.SpanKindRPCServer.Set(span)
	// To ensure accuracy, spans should finish immediately once server responds.
	// So the best way is that spans finish in the callback func, not after it.
	// But server may respond in the callback func too, that we have to remove
	// span finishing from callback func's inside to outside.
	chain.Next(i, func(r *invocation.Response) (err error) {
		err = cb(r)
		switch i.Protocol {
		case common.ProtocolRest:
			span.SetTag(tracing.HTTPMethod, i.Metadata[common.RestMethod])
			span.SetTag(tracing.HTTPPath, i.OperationID)
			span.SetTag(tracing.HTTPStatusCode, r.Status)
		default:
		}
		span.Finish()
		return
	})
}

// Name returns tracing-provider string
func (t *TracingProviderHandler) Name() string {
	return TracingProvider
}

func newTracingProviderHandler() Handler {
	return &TracingProviderHandler{}
}

// TracingConsumerHandler tracing consumer handler
type TracingConsumerHandler struct{}

// Handle is handle consumer tracing related things
func (t *TracingConsumerHandler) Handle(chain *Chain, i *invocation.Invocation, cb invocation.ResponseCallBack) {
	// the span context is in invocation.Ctx
	// start a new span from context
	var span opentracing.Span
	wireContext, _ := opentracing.GlobalTracer().Extract(opentracing.TextMap, opentracing.TextMapCarrier(i.Headers()))
	if wireContext == nil {
		span = opentracing.StartSpan(i.OperationID)
	} else {
		// store span in context
		span = opentracing.StartSpan(i.OperationID, opentracing.ChildOf(wireContext))
	}
	// set span kind to be client
	ext.SpanKindRPCClient.Set(span)
	// store span in context
	i.Ctx = opentracing.ContextWithSpan(i.Ctx, span)

	// inject span context into carrier

	// header stored in context

	if err := opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.TextMap,
		(opentracing.TextMapCarrier)(i.Headers()),
	); err != nil {
		lager.Logger.Errorf("Inject span failed, err [%s]", err.Error())
	}
	// To ensure accuracy, spans should finish immediately once client send req.
	// So the best way is that spans finish in the callback func, not after it.
	// But client may send req in the callback func too, that we have to remove
	// span finishing from callback func's inside to outside.
	chain.Next(i, func(r *invocation.Response) (err error) {
		switch i.Protocol {
		case common.ProtocolRest:
			span.SetTag(tracing.HTTPMethod, i.Metadata[common.RestMethod])
			span.SetTag(tracing.HTTPPath, i.OperationID)
			span.SetTag(tracing.HTTPStatusCode, r.Status)
			span.SetTag(tracing.HTTPHost, i.Endpoint)
		default:
		}
		span.Finish()
		return cb(r)
	})
}

// Name returns tracing-consumer string
func (t *TracingConsumerHandler) Name() string {
	return TracingConsumer
}

func newTracingConsumerHandler() Handler {
	return &TracingConsumerHandler{}
}
