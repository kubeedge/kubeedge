module github.com/kubeedge/beehive

go 1.14

require (
	github.com/satori/go.uuid v1.2.0
	gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 // indirect
	k8s.io/klog v1.0.0
)

replace (
	github.com/apache/servicecomb-kie v0.1.0 => github.com/apache/servicecomb-kie v0.0.0-20190905062319-5ee098c8886f // indirect. TODO: remove this line when servicecomb-kie has a stable release
	github.com/kubeedge/beehive => ../beehive
)
