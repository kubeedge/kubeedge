package proxy

import (
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/util/httpstream"
	awareproxy "k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream"
)

// UpgradeHandler is a handler for proxy requests that may require an upgrade
type UpgradeHandler struct {
	// Responder is passed errors that occur while setting up proxying.
	Responder awareproxy.ErrorResponder

	Session *cloudstream.Session
}

// NewUpgradeHandler creates a new proxy handler with a default flush interval. Responder is required for returning
// errors to the caller.
func NewUpgradeHandler(s *cloudstream.Session) *UpgradeHandler {
	return &UpgradeHandler{
		Session:   s,
		Responder: &responder{},
	}
}

// ServeHTTP handles the proxy request
func (h *UpgradeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if !httpstream.IsUpgradeRequest(req) {
		klog.V(6).Infof("Request was not an upgrade")
		return
	}

	// Once the connection is hijacked, the ErrorResponder will no longer work, so
	// hijacking should be the last step in the upgrade.
	requestHijacker, ok := w.(http.Hijacker)
	if !ok {
		klog.V(6).Infof("Unable to hijack response writer: %T", w)
		h.Responder.Error(w, req, fmt.Errorf("request connection cannot be hijacked: %T", w))
		return
	}
	requestHijackedConn, _, err := requestHijacker.Hijack()
	if err != nil {
		klog.V(6).Infof("Unable to hijack response: %v", err)
		h.Responder.Error(w, req, fmt.Errorf("error hijacking connection: %v", err))
		return
	}
	defer requestHijackedConn.Close()

}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	klog.Errorf("Error while proxying request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
