/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metaserver

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ApplicationStatus string

const (
	// set by agent
	PreApplying ApplicationStatus = "PreApplying" // application is waiting to be sent to cloud
	InApplying  ApplicationStatus = "InApplying"  // application is sending to cloud

	// set by center
	InProcessing ApplicationStatus = "InProcessing" // application is in processing by cloud
	Approved     ApplicationStatus = "Approved"     // application is approved by cloud
	Rejected     ApplicationStatus = "Rejected"     // application is rejected by cloud

	// both
	Failed    ApplicationStatus = "Failed"    // failed to get application resp from cloud
	Completed ApplicationStatus = "Completed" // application is completed and waiting to be recycled
)

type ApplicationVerb string

const (
	Get          ApplicationVerb = "get"
	List         ApplicationVerb = "list"
	Watch        ApplicationVerb = "watch"
	Create       ApplicationVerb = "create"
	Delete       ApplicationVerb = "delete"
	Update       ApplicationVerb = "update"
	UpdateStatus ApplicationVerb = "updatestatus"
	Patch        ApplicationVerb = "patch"
)

type PatchInfo struct {
	Name         string
	PatchType    types.PatchType
	Data         []byte
	Options      metav1.PatchOptions
	Subresources []string
}

// used to set Message.Route
const (
	MetaServerSource    = "metaserver"
	ApplicationResource = "Application"
	ApplicationResp     = "applicationResponse"
	Ignore              = "ignore"
)

const WatchAppSync = "watchapp/sync"
