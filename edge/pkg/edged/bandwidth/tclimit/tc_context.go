/*
Copyright 2024 The KubeEdge Authors.

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

package tclimit

import (
	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/consts"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/bandwidth/tclinux"
)

func deleteNetworkLimit(pod *v1.Pod) {
	tclinux.DeleteNetworkDevice(pod.Name)
}

func podBandwidthLimit(pod *v1.Pod) {
	ingressBandwidthLimit(pod)
	egressBandwidthLimit(pod)
}

func ingressBandwidthLimit(pod *v1.Pod) {
	if !checkIngressEnable(pod.Annotations) {
		return
	}
	doIngressBandwidthLimit(pod)
}

func egressBandwidthLimit(pod *v1.Pod) {
	if !checkEgressEnable(pod.Annotations) {
		return
	}
	doEgressBandwidthLimit(pod)
}

func checkIngressEnable(annotations map[string]string) bool {
	_, ok := annotations[consts.AnnotationIngressBandwidth]
	return ok
}

func checkEgressEnable(annotations map[string]string) bool {
	_, ok := annotations[consts.AnnotationEgressBandwidth]
	return ok
}
