package handlerfactory

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversionscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/fakers"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/scope"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

type Factory struct {
	storage           *storage.REST
	scope             *handlers.RequestScope
	MinRequestTimeout time.Duration
	handlers          map[string]http.Handler
	lock              sync.RWMutex
}

func NewFactory() *Factory {
	s, err := storage.NewREST()
	utilruntime.Must(err)
	f := Factory{
		storage:           s,
		scope:             scope.NewRequestScope(),
		MinRequestTimeout: 1800 * time.Second,
		handlers:          make(map[string]http.Handler),
		lock:              sync.RWMutex{},
	}
	return &f
}

func (f *Factory) Get() http.Handler {
	if h, ok := f.getHandler("get"); ok {
		return h
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	h := handlers.GetResource(f.storage, f.scope)
	f.handlers["get"] = h
	return h
}

func (f *Factory) List() http.Handler {
	if h, ok := f.getHandler("list"); ok {
		return h
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	h := handlers.ListResource(f.storage, f.storage, f.scope, false, f.MinRequestTimeout)
	f.handlers["list"] = h
	return h
}

func (f *Factory) Create(req *request.RequestInfo) http.Handler {
	s := scope.NewRequestScope()
	s.Kind = schema.GroupVersionKind{
		Group:   req.APIGroup,
		Version: req.APIVersion,
		Kind:    util.UnsafeResourceToKind(req.Resource),
	}
	h := handlers.CreateResource(f.storage, s, fakers.NewAlwaysAdmit())
	return h
}

func (f *Factory) Delete() http.Handler {
	if h, ok := f.getHandler("delete"); ok {
		return h
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	h := handlers.DeleteResource(f.storage, false, f.scope, fakers.NewAlwaysAdmit())
	f.handlers["delete"] = h
	return h
}

func (f *Factory) getHandler(key string) (http.Handler, bool) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	h, ok := f.handlers[key]
	return h, ok
}

func (f *Factory) Update(req *request.RequestInfo) http.Handler {
	s := scope.NewRequestScope()
	s.Kind = schema.GroupVersionKind{
		Group:   req.APIGroup,
		Version: req.APIVersion,
		Kind:    util.UnsafeResourceToKind(req.Resource),
	}
	h := handlers.UpdateResource(f.storage, s, fakers.NewAlwaysAdmit())
	return h
}

func (f *Factory) Patch(reqInfo *request.RequestInfo) http.Handler {
	scope := wrapScope{RequestScope: scope.NewRequestScope()}
	scope.Kind = schema.GroupVersionKind{
		Group:   reqInfo.APIGroup,
		Version: reqInfo.APIVersion,
		Kind:    util.UnsafeResourceToKind(reqInfo.Resource),
	}

	h := func(w http.ResponseWriter, req *http.Request) {
		// Do this first, otherwise name extraction can fail for unrecognized content types
		// TODO: handle this in negotiation
		contentType := req.Header.Get("Content-Type")
		// Remove "; charset=" if included in header.
		if idx := strings.Index(contentType, ";"); idx > 0 {
			contentType = contentType[:idx]
		}
		patchType := types.PatchType(contentType)

		// TODO: we either want to remove timeout or document it (if we
		// document, move timeout out of this function and declare it in
		// api_installer)
		timeout := parseTimeout(req.URL.Query().Get("timeout"))

		namespace, name, err := scope.Namer.Name(req)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()
		ctx = request.WithNamespace(ctx, namespace)

		patchBytes, err := limitedReadBody(req, scope.MaxRequestBodyBytes)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		options := &metav1.PatchOptions{}
		if err := metainternalversionscheme.ParameterCodec.DecodeParameters(req.URL.Query(), scope.MetaGroupVersion, options); err != nil {
			err = errors.NewBadRequest(err.Error())
			scope.err(err, w, req)
			return
		}
		if errs := validation.ValidatePatchOptions(options, patchType); len(errs) > 0 {
			err := errors.NewInvalid(schema.GroupKind{Group: metav1.GroupName, Kind: "PatchOptions"}, "", errs)
			scope.err(err, w, req)
			return
		}
		options.TypeMeta.SetGroupVersionKind(metav1.SchemeGroupVersion.WithKind("PatchOptions"))

		reqInfo, _ := request.RequestInfoFrom(req.Context())
		pi := application.PatchInfo{
			Name:         name,
			PatchType:    patchType,
			Data:         patchBytes,
			Options:      *options,
			Subresources: []string{reqInfo.Subresource},
		}

		retObj, err := f.storage.Patch(ctx, pi)
		if err != nil {
			scope.err(err, w, req)
			return
		}

		responsewriters.WriteObjectNegotiated(scope.Serializer, scope, scope.Kind.GroupVersion(), w, req, 200, retObj)
	}
	return http.HandlerFunc(h)
}

type wrapScope struct {
	*handlers.RequestScope
}

func (scope *wrapScope) err(err error, w http.ResponseWriter, req *http.Request) {
	responsewriters.ErrorNegotiated(err, scope.Serializer, scope.Kind.GroupVersion(), w, req)
}

func parseTimeout(str string) time.Duration {
	if str != "" {
		timeout, err := time.ParseDuration(str)
		if err == nil {
			return timeout
		}
		klog.Errorf("Failed to parse %q: %v", str, err)
	}
	// 34 chose as a number close to 30 that is likely to be unique enough to jump out at me the next time I see a timeout.  Everyone chooses 30.
	return 34 * time.Second
}

func limitedReadBody(req *http.Request, limit int64) ([]byte, error) {
	defer req.Body.Close()
	if limit <= 0 {
		return io.ReadAll(req.Body)
	}
	lr := &io.LimitedReader{
		R: req.Body,
		N: limit + 1,
	}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N <= 0 {
		return nil, errors.NewRequestEntityTooLargeError(fmt.Sprintf("limit is %d", limit))
	}
	return data, nil
}
