module github.com/kubeedge/Template

go 1.22.0

toolchain go1.23.2

require (
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.7.1
	github.com/influxdata/influxdb-client-go/v2 v2.13.0
	github.com/kubeedge/api v0.0.0
	github.com/kubeedge/mapper-framework v0.0.0
	github.com/sailorvii/goav v0.1.4
	github.com/taosdata/driver-go/v3 v3.5.1
	go.opentelemetry.io/otel v1.23.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.23.0
	go.opentelemetry.io/otel/metric v1.23.0
	go.opentelemetry.io/otel/sdk v1.23.0
	go.opentelemetry.io/otel/sdk/metric v1.23.0
	k8s.io/klog/v2 v2.120.1
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/avast/retry-go v3.0.0+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oapi-codegen/runtime v1.0.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	go.opentelemetry.io/otel/trace v1.23.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241007155032-5fefd90f89a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241007155032-5fefd90f89a9 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace (
	github.com/kubeedge/api => ../api
	github.com/kubeedge/mapper-framework => ../mapper-framework
)
