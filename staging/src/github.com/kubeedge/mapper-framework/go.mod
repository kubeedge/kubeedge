module github.com/kubeedge/mapper-framework

go 1.25.0

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.0
	github.com/kubeedge/api v0.0.0
	github.com/spf13/pflag v1.0.6
	golang.org/x/net v0.56.0 // indirect
	google.golang.org/grpc v1.72.1
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.140.0
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.39.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/kubeedge/api => ../api
