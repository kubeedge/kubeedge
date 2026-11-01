/*
Copyright 2024 The KubeEdge Authors.

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

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/config"
	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/driver"
	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/dmi"
	"github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper/pkg/inference"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	cfg, err := config.Load()
	if err != nil {
		klog.Fatalf("config load failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sensor := driver.NewVirtualSensor(cfg.SensorConfig)
	detector := inference.NewDetector()

	client, err := dmi.NewClient(cfg.DMIConfig)
	if err != nil {
		klog.Fatalf("dmi client failed: %v", err)
	}

	if err := client.Register(ctx); err != nil {
		klog.Fatalf("mapper register failed: %v", err)
	}

	go runLoop(ctx, sensor, detector, client, cfg.ReportIntervalSec)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	cancel()
	time.Sleep(500 * time.Millisecond)
}

func runLoop(ctx context.Context, sensor *driver.VirtualSensor, detector *inference.Detector, client *dmi.Client, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r := sensor.Read()
			result := detector.Analyze(r)
			r.IsAnomaly = result.IsAnomaly
			if err := client.ReportStatus(ctx, r); err != nil {
				klog.Warningf("report failed (entering autonomous mode): %v", err)
			}
		}
	}
}
