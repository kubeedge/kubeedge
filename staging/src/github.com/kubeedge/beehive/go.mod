module github.com/kubeedge/beehive

go 1.12

require (
	github.com/go-chassis/go-archaius v0.20.0
	github.com/go-chassis/go-chassis v1.7.1 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190115171406-56726106282f // indirect
	github.com/prometheus/procfs v0.0.0-20190117184657-bf6a532e95b1 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/smartystreets/assertions v0.0.0-20190116191733-b6c0e53d7304 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	golang.org/x/sys v0.0.0-20190318195719-6c81ef8f67ca // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/klog v1.0.0
)

replace (
	github.com/apache/servicecomb-kie v0.1.0 => github.com/apache/servicecomb-kie v0.0.0-20190905142319-5ee098c8886f // indirect. TODO: remove this line when servicecomb-kie has a stable release
	github.com/kubeedge/beehive => ../beehive
)
