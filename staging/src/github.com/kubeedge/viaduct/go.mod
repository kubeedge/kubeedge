module github.com/kubeedge/viaduct

go 1.16

require (
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.4.2
	github.com/kubeedge/beehive v0.0.0
	github.com/lucas-clemente/quic-go v0.24.0
	k8s.io/klog/v2 v2.9.0
)

replace (
	github.com/kubeedge/beehive => ../beehive
	github.com/kubeedge/viaduct => ../viaduct
)
