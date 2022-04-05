/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/util/constants.go
- pick some constants used by imageoverrider
*/

package overridemanager

const (
	// DeploymentKind indicates the target resource is a deployment
	DeploymentKind = "Deployment"
	// JobKind indicates the target resource is a job
	JobKind = "Job"
	// PodKind indicates the target resource is a pod
	PodKind = "Pod"
	// ReplicaSetKind indicates the target resource is a replicaset
	ReplicaSetKind = "ReplicaSet"
	// StatefulSetKind indicates the target resource is a statefulset
	StatefulSetKind = "StatefulSet"
	// DaemonSetKind indicates the target resource is a daemonset
	DaemonSetKind = "DaemonSet"
)
