module github.com/kubeedge/mapper-framework

go 1.22.0

toolchain go1.23.2

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.0
	github.com/kubeedge/api v0.0.0
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	golang.org/x/net v0.24.0 // indirect
	google.golang.org/grpc v1.63.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.120.1
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/kubeedge/api => ../api
