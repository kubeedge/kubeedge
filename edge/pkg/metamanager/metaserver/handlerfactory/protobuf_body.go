/*
Copyright 2026 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlerfactory

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/protobuf"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
)

// client-go's scheme holds every built-in API group, so it can decode any
// built-in kind a client sends as protobuf. Custom resources are never protobuf.
var (
	protobufDecoder = protobuf.NewSerializer(kubescheme.Scheme, kubescheme.Scheme)
	jsonEncoder     = json.NewSerializerWithOptions(json.DefaultMetaFactory, kubescheme.Scheme, kubescheme.Scheme, json.SerializerOptions{})
)

// protobufToJSON lets the stock write handlers, which accept only JSON and YAML,
// take a protobuf body: it decodes the body into its built-in type and passes it
// on as JSON. Requests with any other content type pass through untouched.
func protobufToJSON(s *handlers.RequestScope, newHandler func(*handlers.RequestScope) http.Handler) http.Handler {
	h := newHandler(s)
	scope := wrapScope{RequestScope: s}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil || mediaType != runtime.ContentTypeProtobuf {
			h.ServeHTTP(w, req)
			return
		}

		body, err := limitedReadBody(req, scope.MaxRequestBodyBytes)
		if err != nil {
			scope.err(err, w, req)
			return
		}
		jsonBody, err := protobufBodyToJSON(body)
		if err != nil {
			logProtobufError(req, err)
			scope.err(err, w, req)
			return
		}

		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(jsonBody))
		req.ContentLength = int64(len(jsonBody))
		req.Header.Set("Content-Type", runtime.ContentTypeJSON)
		// The limit was checked on the protobuf body; its JSON form is larger.
		converted := *s
		converted.MaxRequestBodyBytes = 0
		newHandler(&converted).ServeHTTP(w, req)
	})
}

// A client retries a refused or failed write, so each is logged a few times per
// minute. Separate limits keep refusals from hiding a failure of the check.
var (
	refusalWarnings = rate.NewLimiter(rate.Every(20*time.Second), 3)
	failureErrors   = rate.NewLimiter(rate.Every(20*time.Second), 3)
)

// logProtobufError logs a 415 as a warning and a 500 as an error. A 400 is the
// client's fault and is not logged.
func logProtobufError(req *http.Request, err error) {
	limiter, logf, verb := refusalWarnings, klog.Warningf, "refused"
	switch {
	case errors.IsUnsupportedMediaType(err):
	case errors.IsInternalError(err):
		limiter, logf, verb = failureErrors, klog.Errorf, "failed to convert"
	default:
		return
	}
	if !limiter.Allow() {
		logf = klog.V(4).Infof
	}
	logf("MetaServer %s the protobuf body of %s %s from %q: %v", verb, req.Method, req.URL.Path, req.UserAgent(), err)
}

// protobufBodyToJSON returns API status errors: 415 for a kind or a field that
// edgecore's types lack, so the client can send it as JSON instead, and 400 for
// a body that is not valid protobuf.
func protobufBodyToJSON(body []byte) ([]byte, error) {
	obj, gvk, err := protobufDecoder.Decode(body, nil, nil)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			return nil, unsupportedProtobuf(fmt.Sprintf("%s is not known to this edgecore's API types, send it as JSON", kindName(gvk)))
		}
		return nil, errors.NewBadRequest(fmt.Sprintf("failed to decode the protobuf request body: %v", err))
	}

	// Decoding skips fields the types lack, and an update would then remove them in the cloud.
	dropped, err := droppedProtobufFields(body, obj)
	if _, ok := err.(malformedBodyError); ok {
		return nil, errors.NewBadRequest(fmt.Sprintf("failed to read the protobuf request body: %v", err))
	}
	if err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("failed to check the %s protobuf request body: %w", kindName(gvk), err))
	}
	if len(dropped) > 0 {
		return nil, unsupportedProtobuf(fmt.Sprintf("this edgecore's %s type lacks fields set in the protobuf body, so they would be lost: %s; send it as JSON",
			kindName(gvk), strings.Join(dropped, ", ")))
	}

	// The decoder has set apiVersion and kind on obj, which the stock handler needs.
	jsonBody, err := runtime.Encode(jsonEncoder, obj)
	if err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("failed to encode the %v request body as JSON: %w", gvk, err))
	}
	return jsonBody, nil
}

func droppedProtobufFields(body []byte, obj runtime.Object) ([]string, error) {
	envelope := &runtime.Unknown{}
	if _, _, err := protobufDecoder.Decode(body, nil, envelope); err != nil {
		return nil, err
	}
	name, err := builtInMessages.messageOf(obj)
	if err != nil {
		return nil, err
	}
	return builtInMessages.droppedFields(envelope.Raw, name, "", nil)
}

// kindName gives "v1 Node" rather than "/v1, Kind=Node".
func kindName(gvk *schema.GroupVersionKind) string {
	if gvk == nil {
		return "this kind"
	}
	return gvk.GroupVersion().String() + " " + gvk.Kind
}

func unsupportedProtobuf(message string) error {
	return &errors.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    http.StatusUnsupportedMediaType,
		Reason:  metav1.StatusReasonUnsupportedMediaType,
		Message: message,
	}}
}
