package httpserver

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"golang.org/x/time/rate"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

type IPRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    sync.RWMutex
	qps   rate.Limit
	burst int
}

func NewIPRateLimiter(qps rate.Limit, burst int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips:   make(map[string]*rate.Limiter),
		qps:   qps,
		burst: burst,
	}

	// Simple cleanup routine to prevent memory leaks from old IPs
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			i.mu.Lock()
			i.ips = make(map[string]*rate.Limiter)
			i.mu.Unlock()
		}
	}()

	return i
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		defer i.mu.Unlock()
		limiter, exists = i.ips[ip]
		if !exists {
			limiter = rate.NewLimiter(i.qps, i.burst)
			i.ips[ip] = limiter
		}
	}

	return limiter
}

var ipRateLimiter *IPRateLimiter
var initOnce sync.Once

func rateLimitFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	initOnce.Do(func() {
		qps := 100
		burst := 200
		if hubconfig.Config.HTTPS != nil && hubconfig.Config.HTTPS.RateLimit != nil {
			qps = int(hubconfig.Config.HTTPS.RateLimit.QPS)
			burst = int(hubconfig.Config.HTTPS.RateLimit.Burst)
		}
		ipRateLimiter = NewIPRateLimiter(rate.Limit(qps), burst)
	})

	ip := req.Request.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	limiter := ipRateLimiter.GetLimiter(ip)
	if !limiter.Allow() {
		klog.Warningf("Rate limit exceeded for IP: %s", ip)
		_ = resp.WriteErrorString(http.StatusTooManyRequests, "Too Many Requests")
		return
	}

	chain.ProcessFilter(req, resp)
}
