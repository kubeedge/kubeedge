package checker

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/atomic"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/config"
	"github.com/kubeedge/kubeedge/edge/pkg/lwproxy/util"
)

type Checker interface {
	Check() bool
}

func NewHealthzChecker(url string) Checker {
	hc := &healthzChecker{
		url:      url,
		isOk:     atomic.NewBool(false),
		interval: time.Duration(config.Config.HealthzCheckInterval) * time.Second,
		client: &http.Client{
			Transport: util.GetInsecureTransport(),
			Timeout:   time.Duration(config.Config.HealthzCheckTimeout) * time.Second,
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
	for range intervalTicker.C {
		for i := 0; i < config.Config.HealthzCheckRetryTimes; i++ {
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
