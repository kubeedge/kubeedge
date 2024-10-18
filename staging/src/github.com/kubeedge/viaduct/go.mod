module github.com/kubeedge/viaduct

go 1.21

require (
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/websocket v1.5.0
	github.com/kubeedge/beehive v0.0.0
	github.com/lucas-clemente/quic-go v0.10.1
	k8s.io/klog/v2 v2.110.1
)

require (
	github.com/bifurcation/mint v0.0.0-20180715133206-93c51c6ce115 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/lucas-clemente/aes12 v0.0.0-20171027163421-cd47fb39b79f // indirect
	github.com/lucas-clemente/quic-go-certificates v0.0.0-20160823095156-d2f86524cced // indirect
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/onsi/gomega v1.29.0 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/kubeedge/beehive => ../beehive
