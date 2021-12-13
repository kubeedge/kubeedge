module github.com/kubeedge/viaduct

go 1.16

require (
	github.com/bifurcation/mint v0.0.0-20180715133206-93c51c6ce115 // indirect
	github.com/cheekybits/genny v0.0.0-20170328200008-9127e812e1e9 // indirect
	github.com/golang/mock v0.0.0-20190508161146-9fa652df1129
	github.com/golang/protobuf v1.3.1
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/golang-lru v0.0.0-20180201235237-0fb14efe8c47 // indirect
	github.com/kubeedge/beehive v0.0.0
	github.com/lucas-clemente/aes12 v0.0.0-20171027163421-cd47fb39b79f // indirect
	github.com/lucas-clemente/quic-go v0.10.1
	github.com/lucas-clemente/quic-go-certificates v0.0.0-20160823095156-d2f86524cced // indirect
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	golang.org/x/sys v0.0.0-20190322080309-f49334f85ddc // indirect
	k8s.io/klog/v2 v2.2.0
)

replace (
	github.com/kubeedge/beehive => ../beehive
	github.com/kubeedge/viaduct => ../viaduct
)
