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

func NewHealthzChecker(url string) *healthzChecker {
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
	healthzUrl := strings.Join([]string{h.url, "healthz"}, "/")
	intervalTicker := time.NewTicker(h.interval)
	isHealthy := false
	for range intervalTicker.C {
		for i := 0; i < 3; i++ {
			resp, err := h.client.Get(healthzUrl)
			if err != nil {
				isHealthy = false
				klog.Warningf("check %s failed", healthzUrl)
				continue
			}
			if resp.StatusCode == http.StatusOK {
				isHealthy = true
				break
			}
		}
		klog.Infof("healthzChecker check %s result %v", healthzUrl, isHealthy)
		h.isOk.Store(isHealthy)
	}

}
