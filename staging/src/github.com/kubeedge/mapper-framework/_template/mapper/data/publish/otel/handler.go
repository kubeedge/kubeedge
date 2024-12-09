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

package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric"
	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/driver"
	"github.com/kubeedge/mapper-framework/pkg/common"
)

const meterName = "github.com/kubeedge/Template/data/dbmethod/otel"

func DataHandler(ctx context.Context, twin *common.Twin, client *driver.CustomizedClient, visitorConfig *driver.VisitorConfig, dataModel *common.DataModel) {
	cfg, err := NewConfig(twin.Property.PushMethod.MethodConfig)
	if err != nil {
		klog.Errorf("new config fail: %v", err)
		return
	}

	provider, err := cfg.InitProvider(time.Duration(twin.Property.ReportCycle), dataModel)
	if err != nil {
		klog.Errorf("init provider fail: %v", err)
		return
	}
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = provider.Shutdown(ctx)
		if err != nil {
			klog.Errorf("shutdown provider fail: %v", err)
		}
	}()

	meter := provider.Meter(meterName)

	gauge, err := meter.Float64ObservableGauge(dataModel.PropertyName)
	if err != nil {
		klog.Errorf("create metric fail: %v", err)
		return
	}

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		data, err := client.GetDeviceData(visitorConfig)
		if err != nil {
			return fmt.Errorf("get device data fail: %v", err)
		}

		o.ObserveFloat64(gauge, data.(float64))
		return nil
	}, gauge)
	if err != nil {
		klog.Errorf("register callback fail: %v", err)
	}
}
