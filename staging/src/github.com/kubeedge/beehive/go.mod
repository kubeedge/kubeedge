module github.com/kubeedge/beehive

go 1.16

require (
	github.com/satori/go.uuid v1.2.0
	k8s.io/klog/v2 v2.2.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/apache/servicecomb-kie v0.1.0 => github.com/apache/servicecomb-kie v0.0.0-20190905062319-5ee098c8886f // indirect. TODO: remove this line when servicecomb-kie has a stable release
	github.com/kubeedge/beehive => ../beehive
)
