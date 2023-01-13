module github.com/kubeedge/beehive

go 1.17

require (
	github.com/google/uuid v1.2.0
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/yaml v1.2.0
)

require (
	github.com/go-logr/logr v0.4.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace github.com/kubeedge/beehive => ../beehive
