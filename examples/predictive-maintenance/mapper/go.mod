module github.com/kubeedge/kubeedge/examples/predictive-maintenance/mapper

go 1.23

require (
	github.com/kubeedge/api v0.0.0
	github.com/kubeedge/kubeedge v0.0.0
	github.com/stretchr/testify v1.10.0
	google.golang.org/grpc v1.67.1
	k8s.io/klog/v2 v2.140.0
)

replace (
	github.com/kubeedge/api => ../../../staging/src/github.com/kubeedge/api
	github.com/kubeedge/kubeedge => ../../..
)
