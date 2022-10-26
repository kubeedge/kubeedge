/*
Copyright 2022 The KubeEdge Authors.

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

package monitor

import (
	"context"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	config "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

const (
	metricNamespace = "KubeEdge"

	// CloudHubSubsystem - subsystem name used by CloudHub
	CloudHubSubsystem = "CloudHub"
)

var (
	ConnectedNodes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: CloudHubSubsystem,
			Name:      "connected_nodes",
			Help:      "Number of nodes that connected to the cloudHub instance",
		},
	)
)

var registerOnce sync.Once

// registerMetrics register all metrics.
func registerMetrics() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			ConnectedNodes,
		)
	})
}

func InstallHandlerForPProf(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// ServeMonitor serve monitoring metric.
func ServeMonitor(config config.MonitorServer) {
	registerMetrics()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	if config.EnableProfiling {
		InstallHandlerForPProf(mux)
	}

	s := http.Server{
		Addr:    config.BindAddress,
		Handler: mux,
	}

	go func() {
		ctx := beehivecontext.GetContext()
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			klog.Errorf("Server shutdown failed: %v", err)
		}
	}()

	klog.Infof("starting monitor server on addr: %s", config.BindAddress)
	klog.Exit(s.ListenAndServe())
}
