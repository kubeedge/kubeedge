package constants

import "time"

const (
	Interval = 5 * time.Second
	Timeout  = 10 * time.Minute

	E2ELabelKey   = "kubeedge"
	E2ELabelValue = "e2e-test"

	NodeName = "edge-node"
)

var (
	// KubeEdgeE2ELabel labels resources created during e2e testing
	KubeEdgeE2ELabel = map[string]string{
		E2ELabelKey: E2ELabelValue,
	}
)
