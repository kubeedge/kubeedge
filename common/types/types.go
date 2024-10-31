package types

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/api/apis/operations/v1alpha1"
)

// EventPatchInfo is used for sending patch event information from edge to cloud
type EventPatchInfo struct {
	Event *v1.Event `json:"event"`
	Data  string    `json:"patchData"`
}

// PodStatusRequest is Message.Content which comes from edge
type PodStatusRequest struct {
	UID    types.UID
	Name   string
	Status v1.PodStatus
}

// ExtendResource is the extended resource detail that comes from edge
type ExtendResource struct {
	Name     string            `json:"name,omitempty"`
	Type     string            `json:"type,omitempty"`
	Capacity resource.Quantity `json:"capacity,omitempty"`
}

// NodeStatusRequest is Message.Content which comes from edge
type NodeStatusRequest struct {
	UID             types.UID
	Status          v1.NodeStatus
	ExtendResources map[v1.ResourceName][]ExtendResource
}

// NodeUpgradeJobRequest is upgrade msg coming from cloud to edge
type NodeUpgradeJobRequest struct {
	UpgradeID           string
	HistoryID           string
	Version             string
	UpgradeTool         string
	Image               string
	ImageDigest         string
	RequireConfirmation bool
}

// NodeUpgradeJobResponse is used to report status msg to cloudhub https service
type NodeUpgradeJobResponse struct {
	UpgradeID   string
	HistoryID   string
	NodeName    string
	FromVersion string
	ToVersion   string
	Status      string
	Reason      string
}

// NodePreCheckRequest is pre-check msg coming from cloud to edge
type NodePreCheckRequest struct {
	CheckItem []string
}

type NodeTaskRequest struct {
	TaskID string
	Type   string
	State  string
	Item   interface{}
}

type NodeTaskResponse struct {
	// NodeName is the name of edge node.
	NodeName string
	// State represents for the upgrade state phase of the edge node.
	// There are several possible state values: "", Upgrading, BackingUp, RollingBack and Checking.
	State api.State
	// Event represents for the event of the ImagePrePullJob.
	// There are three possible event values: Init, Check, Pull.
	Event string
	// Action represents for the action of the ImagePrePullJob.
	// There are three possible action values: Success, Failure, TimeOut.
	Action api.Action
	// Reason represents for the reason of the ImagePrePullJob.
	Reason string
	// Time represents for the running time of the ImagePrePullJob.
	Time string

	ExternalMessage string
}

// ObjectResp is the object that api-server response
type ObjectResp struct {
	Object metaV1.Object
	Err    error
}

// ImagePrePullJobRequest is image prepull msg from cloud to edge
type ImagePrePullJobRequest struct {
	Images     []string
	NodeName   string
	Secret     string
	RetryTimes int32
	CheckItems []string
}

// ImagePrePullJobResponse is used to report status msg to cloudhub https service from each node
type ImagePrePullJobResponse struct {
	NodeName    string
	State       string
	Reason      string
	ImageStatus []v1alpha1.ImageStatus
}

type RestartResponse struct {
	ErrMessages []string `json:"errMessages,omitempty"`
	LogMessages []string `json:"LogMessages,omitempty"`
}

type NodeUpgradeConfirmResponse struct {
	ErrMessages []string `json:"errMessages,omitempty"`
	LogMessages []string `json:"logMessages,omitempty"`
}

type LogsResponse struct {
	ErrMessages []string `json:"errMessages,omitempty"`
	LogMessages []string `json:"logMessages,omitempty"`
}

type ExecResponse struct {
	ErrMessages    []string `json:"errMessages,omitempty"`
	RunOutMessages []string `json:"runOutMessages,omitempty"`
	RunErrMessages []string `json:"runErrMessages,omitempty"`
}
