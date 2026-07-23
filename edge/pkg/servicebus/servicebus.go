package servicebus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	servicebusConfig "github.com/kubeedge/kubeedge/edge/pkg/servicebus/config"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus/util"
	"github.com/kubeedge/kubeedge/pkg/features"
)

var (
	inited int32
	c      = make(chan struct{})
	// configuredTLSOpts holds the TLS options supplied at registration time.
	// It is set once by Register and then read by both Start() and
	// processMessage() so that initial and dynamic/delayed server startup
	// always use the same TLS configuration.
	configuredTLSOpts TLSOptions
)

const (
	sourceType  = "router_servicebus"
	maxBodySize = 5 * 1e6
)

// TLSOptions carries the ServiceBus-specific TLS certificate material.
// It is passed explicitly through Register so that the server() function
// never has to read global command-line options.
//
//   - If TLSEnabled is false, the server starts plain HTTP (backward-compatible
//     default).
//   - If TLSEnabled is true and the cert or key path is empty or invalid,
//     server() logs an error, resets the inited flag, and returns without
//     starting.  The server is NOT started in plain HTTP — a missing or bad
//     TLS configuration must never silently downgrade to plaintext when the
//     operator explicitly requested TLS.
//     Note: Register has no return value; TLS validation happens
//     asynchronously inside server() after the module starts.
//   - ClientAuth is intentionally omitted: local applications that talk to
//     ServiceBus are not provisioned with client certificates, so this
//     implementation provides server-only TLS.  A follow-up can add
//     configurable mTLS once a client-certificate provisioning workflow exists.
//
// NOTE: The certificate supplied here must have ExtKeyUsageServerAuth and
// IP/DNS SANs matching the ServiceBus listen address (e.g. 127.0.0.1).
// EdgeCore enforces ExtKeyUsageServerAuth at startup: a ClientAuth-only
// certificate (such as the EdgeHub client certificate) is rejected with an
// error before the server starts. The same check is applied during certificate
// rotation: if the GetCertificate callback detects that a rotated certificate
// lacks ExtKeyUsageServerAuth, it returns an error so the handshake fails
// rather than silently serving a misconfigured cert. SAN matching is enforced
// by HTTPS clients during the TLS handshake. The EdgeHub client certificate
// CANNOT be reused because it carries ExtKeyUsageClientAuth and no ServiceBus
// SANs.
type TLSOptions struct {
	// TLSEnabled controls whether the ServiceBus HTTP server starts with TLS.
	// Default: false (plain HTTP, backward compatible).
	TLSEnabled bool

	// CertFile is the path to the PEM-encoded server certificate.
	// Required when TLSEnabled is true.
	CertFile string

	// KeyFile is the path to the PEM-encoded private key.
	// Required when TLSEnabled is true.
	KeyFile string
}

// servicebus struct
type servicebus struct {
	enable bool
	// default 127.0.0.1
	server  string
	port    int
	timeout int
	sbs     *dbclient.ServiceBusService
	tlsOpts TLSOptions
}

type serverRequest struct {
	Method    string      `json:"method"`
	TargetURL string      `json:"targetURL"`
	Payload   interface{} `json:"payload"`
}

type serverResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Body string `json:"body"`
}

var htc = new(http.Client)
var uc = new(util.URLClient)

func newServicebus(enable bool, server string, port, timeout int, tlsOpts TLSOptions) *servicebus {
	return &servicebus{
		enable:  enable,
		server:  server,
		port:    port,
		timeout: timeout,
		sbs:     dbclient.NewServiceBusService(),
		tlsOpts: tlsOpts,
	}
}

// Register registers the servicebus module.  tlsOpts controls whether the
// embedded HTTP server uses TLS.  Pass a zero-value TLSOptions{} for plain
// HTTP (backward-compatible default).
func Register(s *v1alpha2.ServiceBus, tlsOpts TLSOptions) {
	configuredTLSOpts = tlsOpts
	servicebusConfig.InitConfigure(s)
	core.Register(newServicebus(s.Enable, s.Server, s.Port, s.Timeout, tlsOpts))
}

func (*servicebus) Name() string {
	return modules.ServiceBusModuleName
}

func (*servicebus) Group() string {
	return modules.BusGroup
}

func (sb *servicebus) Enable() bool {
	return sb.enable
}

func (sb *servicebus) RestartPolicy() *core.ModuleRestartPolicy {
	if !features.DefaultFeatureGate.Enabled(features.ModuleRestart) {
		return nil
	}
	return &core.ModuleRestartPolicy{
		RestartType:            core.RestartTypeOnFailure,
		IntervalTimeGrowthRate: 2.0,
	}
}

func (sb *servicebus) Start() {
	// no need to call TopicInit now, we have fixed topic
	htc.Timeout = time.Second * 10
	uc.Client = htc
	if !sb.sbs.IsTableEmpty() {
		if atomic.CompareAndSwapInt32(&inited, 0, 1) {
			go server(c, sb.tlsOpts)
		}
	}
	//Get message from channel
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("servicebus stop")
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.ServiceBusModuleName)
		if err != nil {
			klog.Warningf("servicebus receive msg error %v", err)
			continue
		}

		// build new message with required field & send message to servicebus
		klog.V(4).Info("servicebus receive msg")
		go processMessage(&msg)
	}
}

func processMessage(msg *beehiveModel.Message) {
	source := msg.GetSource()
	if source != sourceType {
		return
	}
	resource := msg.GetResource()
	dbc := dbclient.NewServiceBusService()
	switch msg.GetOperation() {
	case message.OperationStart:
		if err := dbc.InsertUrls(resource); err != nil {
			// TODO: handle err
			klog.Error(err)
		}
		if atomic.CompareAndSwapInt32(&inited, 0, 1) {
			// Use configuredTLSOpts so dynamic startup honours the same
			// TLS configuration as initial startup.
			go server(c, configuredTLSOpts)
		}
	case message.OperationStop:
		if err := dbc.DeleteUrlsByKey(resource); err != nil {
			// TODO: handle err
			klog.Error(err)
		}

		if dbc.IsTableEmpty() {
			c <- struct{}{}
		}
	default:
		r := strings.Split(resource, ":")
		if len(r) != 2 {
			m := "the format of resource " + resource + " is incorrect"
			klog.Warning(m)
			code := http.StatusBadRequest
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}
		content, err := msg.GetContentData()
		if err != nil {
			klog.Errorf("marshall message content failed %v", err)
			m := "error to marshal request msg content"
			code := http.StatusBadRequest
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		var httpRequest commonType.HTTPRequest
		if err := json.Unmarshal(content, &httpRequest); err != nil {
			m := "error to parse http request"
			code := http.StatusBadRequest
			klog.Errorf(m, err)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		//send message with resource to the edge part
		operation := httpRequest.Method
		targetURL := "http://127.0.0.1:" + r[0] + r[1]
		resp, err := uc.HTTPDo(operation, targetURL, httpRequest.Header, httpRequest.Body)
		if err != nil {
			m := "error to call service"
			code := http.StatusNotFound
			klog.Errorf(m, err)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}
		defer resp.Body.Close()
		resBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
		if err != nil {
			if err.Error() == "http: request body too large" {
				err = fmt.Errorf("response body too large")
			}
			m := "error to receive response, err: " + err.Error()
			code := http.StatusInternalServerError
			klog.Errorf(m, err)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		response := commonType.HTTPResponse{Header: resp.Header, StatusCode: resp.StatusCode, Body: resBody}
		responseMsg := beehiveModel.NewMessage(msg.GetID()).SetRoute(modules.ServiceBusModuleName, modules.UserGroup).
			SetResourceOperation("", beehiveModel.UploadOperation).FillBody(response)
		beehiveContext.SendToGroup(modules.HubGroup, *responseMsg)
	}
}

// buildTLSConfig constructs a *tls.Config for server-only TLS from the given
// certificate and key files.
//
// Design decisions:
//
//   - Returns (nil, nil) only when opts.TLSEnabled is false — the caller is
//     responsible for checking TLSEnabled before calling this function.
//
//   - Returns a non-nil error when opts.TLSEnabled is true but the cert or
//     key path is empty, the key pair cannot be loaded, or the certificate
//     does not have ExtKeyUsageServerAuth.  The caller MUST treat this as a
//     fatal configuration error and NOT fall back to plain HTTP.  Silently
//     downgrading an explicitly enabled TLS endpoint removes transport
//     security without notifying the operator.
//
//   - ExtKeyUsageServerAuth is enforced at startup: a ClientAuth-only
//     certificate (such as the EdgeHub client certificate) is rejected
//     immediately with a clear error.  This prevents a misconfigured
//     certificate from silently allowing connections that clients would reject
//     at handshake time.
//
//   - GetCertificate is used instead of a static Certificates slice so that
//     certificate rotation takes effect on the next TLS handshake without an
//     EdgeCore restart.
//
//   - This function provides server-only TLS (ClientAuth: NoClientCert).
//     Local applications that talk to ServiceBus are not provisioned with
//     client certificates.  mTLS is intentionally out of scope until a
//     client-certificate provisioning workflow is defined.
//
//   - The certificate must have ExtKeyUsageServerAuth and IP/DNS SANs
//     matching the ServiceBus listen address (e.g. 127.0.0.1).
//     ExtKeyUsageServerAuth is validated here at startup; SAN matching
//     is enforced by HTTPS clients during the TLS handshake (not by EdgeCore).
func buildTLSConfig(opts TLSOptions) (*tls.Config, error) {
	if !opts.TLSEnabled {
		return nil, nil
	}
	if opts.CertFile == "" || opts.KeyFile == "" {
		return nil, fmt.Errorf("[servicebus] TLS is enabled but CertFile or KeyFile is empty")
	}

	// Validate the key pair is loadable at startup so we fail fast with a
	// clear error instead of crashing silently on the first TLS handshake.
	if _, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile); err != nil {
		return nil, fmt.Errorf("[servicebus] failed to load TLS key pair: %w", err)
	}

	// Enforce ExtKeyUsageServerAuth at startup.  A ClientAuth-only certificate
	// (e.g. the EdgeHub certificate) is rejected here with a clear message
	// rather than silently accepted and then rejected by every connecting
	// client during the TLS handshake.
	if err := validateServerAuthEKU(opts.CertFile); err != nil {
		return nil, err
	}

	certFile := opts.CertFile
	keyFile := opts.KeyFile

	// Use GetCertificate so the cert is re-read from disk on every new TLS
	// handshake, enabling transparent certificate rotation.
	// EKU is re-validated on every reload so that a rotated certificate
	// lacking ExtKeyUsageServerAuth is rejected at handshake time rather than
	// silently served, bypassing the startup check.
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Server-only TLS: local applications are not expected to present
		// client certs.  Set explicitly so the policy is visible and auditable.
		ClientAuth: tls.NoClientCert,
		GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
			c, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return nil, fmt.Errorf("[servicebus] certificate rotation: failed to reload key pair: %w", err)
			}
			// Re-validate EKU after rotation so a newly deployed certificate
			// that lacks ServerAuth is rejected rather than served.
			if err := validateServerAuthEKU(certFile); err != nil {
				return nil, fmt.Errorf("[servicebus] certificate rotation: %w", err)
			}
			return &c, nil
		},
	}

	return tlsCfg, nil
}

// validateServerAuthEKU reads the PEM certificate at certFile and returns an
// error if it does not contain ExtKeyUsageServerAuth.
// This prevents operators from accidentally configuring a ClientAuth-only
// certificate (such as the EdgeHub client certificate) as a ServiceBus
// server identity.
func validateServerAuthEKU(certFile string) error {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("[servicebus] failed to read certificate file %q: %w", certFile, err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("[servicebus] certificate file %q does not contain a valid PEM block", certFile)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("[servicebus] failed to parse certificate %q: %w", certFile, err)
	}
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			return nil
		}
	}
	return fmt.Errorf("[servicebus] certificate %q does not have ExtKeyUsageServerAuth; "+
		"a dedicated server certificate is required (the EdgeHub client certificate cannot be reused)",
		certFile)
}

func server(stopChan <-chan struct{}, tlsOpts TLSOptions) {
	var (
		timeout time.Duration
		err     error
	)
	if timeout, err = time.ParseDuration(fmt.Sprintf("%vs", servicebusConfig.Config.Timeout)); err != nil {
		klog.Errorf("can't format timeout and the default value will be set")
		timeout, _ = time.ParseDuration("10s")
	}

	h := buildBasicHandler(timeout)
	s := http.Server{
		Addr:    fmt.Sprintf("%s:%d", servicebusConfig.Config.Server, servicebusConfig.Config.Port),
		Handler: h,
	}

	if tlsOpts.TLSEnabled {
		// TLS was explicitly requested.  A configuration error must be fatal:
		// do NOT fall back to plain HTTP when the operator enabled TLS.
		tlsCfg, err := buildTLSConfig(tlsOpts)
		if err != nil {
			klog.Errorf("[servicebus] TLS configuration failed, not starting server: %v", err)
			// Reset inited so a later valid startup attempt is not blocked.
			atomic.StoreInt32(&inited, 0)
			return
		}
		s.TLSConfig = tlsCfg
	}

	go func() {
		<-stopChan
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			klog.Errorf("Server shutdown failed: %s", err)
		}
		atomic.StoreInt32(&inited, 0)
	}()

	if s.TLSConfig != nil {
		klog.Infof("[servicebus] starting HTTPS server at %v", s.Addr)
		// cert and key are already loaded via GetCertificate; pass empty strings.
		utilruntime.HandleError(s.ListenAndServeTLS("", ""))
	} else {
		klog.Infof("[servicebus] starting HTTP server at %v (TLS disabled)", s.Addr)
		utilruntime.HandleError(s.ListenAndServe())
	}
}

func buildBasicHandler(timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sReq := &serverRequest{}
		sResp := &serverResponse{}
		req.Body = http.MaxBytesReader(w, req.Body, maxBodySize)
		byteData, err := io.ReadAll(req.Body)
		if err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = "can't read data from body of the http's request"
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		if err = json.Unmarshal(byteData, sReq); err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = "invalid params"
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		if targetURL, _ := dbclient.NewServiceBusService().GetUrlsByKey(sReq.TargetURL); targetURL == nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = fmt.Sprintf("url %s is not allowed and please make a rule for this url in the cloud", sReq.TargetURL)
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		msg := beehiveModel.NewMessage("").BuildRouter(modules.ServiceBusModuleName, modules.UserGroup,
			sReq.TargetURL, beehiveModel.UploadOperation).FillBody(byteData)
		responseMessage, err := beehiveContext.SendSync(modules.EdgeHubModuleName, *msg, timeout)
		if err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = err.Error()
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		resp, err := responseMessage.GetContentData()
		if err != nil {
			sResp.Code = http.StatusInternalServerError
			sResp.Msg = err.Error()
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}

		sResp.Code = http.StatusOK
		sResp.Msg = "receive response from cloud successfully"
		sResp.Body = string(resp)
		if _, err := w.Write(marshalResult(sResp)); err != nil {
			// TODO: handle err
			klog.Error(err)
		}
	})
}

func buildErrorResponse(parentID string, content string, statusCode int) (beehiveModel.Message, error) {
	h := http.Header{}
	h.Add("Server", "kubeedge-edgecore")
	c := commonType.HTTPResponse{Header: h, StatusCode: statusCode, Body: []byte(content)}
	message := beehiveModel.NewMessage(parentID).
		SetRoute(modules.ServiceBusModuleName, modules.UserGroup).FillBody(c)
	return *message, nil
}

func marshalResult(sResp *serverResponse) (resp []byte) {
	resp, _ = json.Marshal(sResp)
	return
}
