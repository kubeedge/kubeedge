module github.com/kubeedge/mapper-framework

go 1.21

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.0
	github.com/kubeedge/kubeedge v0.0.0
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	golang.org/x/net v0.23.0 // indirect
	google.golang.org/grpc v1.63.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.110.1
)

require (
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/kubeedge/kubeedge => ../../../../../../kubeedge
