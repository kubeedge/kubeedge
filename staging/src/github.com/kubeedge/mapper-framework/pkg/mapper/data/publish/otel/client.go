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
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/kubeedge/mapper-framework/pkg/common"
)

type Config struct {
	EndpointURL string `json:"endpointURL,omitempty"`
}

func NewConfig(clientConfig json.RawMessage) (*Config, error) {
	var cfg Config
	err := json.Unmarshal(clientConfig, &cfg)
	if err != nil {
		return nil, err
	}
	if cfg.EndpointURL == "" {
		return nil, errors.New("endpointURL is required")
	}
	return &cfg, nil
}

func (cfg *Config) InitProvider(reportCycle time.Duration, dataModel *common.DataModel) (*metric.MeterProvider, error) {
	exp, err := otlpmetrichttp.New(context.Background(), WithEndpointURL(cfg.EndpointURL)...)
	if err != nil {
		return nil, err
	}

	if reportCycle == 0 {
		reportCycle = common.DefaultReportCycle
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			attribute.String("device.id", dataModel.Namespace+"/"+dataModel.DeviceName),
			//semconv.DeviceID(dataModel.Namespace+"/"+dataModel.DeviceName), // go.opentelemetry.io/otel/semconv/v1.17.0+
		))
	if err != nil {
		return nil, err
	}

	reader := metric.NewPeriodicReader(exp, metric.WithInterval(reportCycle))
	return metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	), nil
}

func WithEndpointURL(v string) []otlpmetrichttp.Option {
	var opts []otlpmetrichttp.Option
	u, err := url.Parse(v)
	if err != nil {
		return nil
	}

	opts = append(opts,
		otlpmetrichttp.WithEndpoint(u.Host),
		otlpmetrichttp.WithURLPath(u.Path),
	)
	if u.Scheme != "https" {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	return opts
}
