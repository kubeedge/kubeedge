package checker

import (
	"net/http"
	"strings"
	"time"

	"k8s.io/klog"

	"go.uber.org/atomic"

	"github.com/kubeedge/kubeedge/edge/pkg/edgeproxy/util"
)

type Checker interface {
	Check() bool
}

func NewHealthzChecker(url string) Checker {
	hc := &healthzChecker{
		url:      url,
		isOk:     atomic.NewBool(false),
		interval: time.Duration(5) * time.Second,
		client: &http.Client{
			Transport: util.GetInsecureTransport(),
			Timeout:   time.Duration(3) * time.Second,
		},
	}
	go hc.loop()
	return hc
}

type healthzChecker struct {
	url      string
	isOk     *atomic.Bool
	interval time.Duration
	client   *http.Client
}

func (h *healthzChecker) Check() bool {
	return h.isOk.Load()
}

func (h *healthzChecker) loop() {
	healthzURL := strings.Join([]string{h.url, "healthz"}, "/")
	intervalTicker := time.NewTicker(h.interval)
	isHealthy := false
	retryTimes := 3
	for range intervalTicker.C {
		for i := 0; i < retryTimes; i++ {
			resp, err := h.client.Get(healthzURL)
			if err != nil {
				isHealthy = false
				klog.Warningf("check %s failed", healthzURL)
				continue
			}
			if resp.StatusCode == http.StatusOK {
				isHealthy = true
				break
			}
		}
		klog.Infof("healthzChecker check %s result %v", healthzURL, isHealthy)
		h.isOk.Store(isHealthy)
	}
}
