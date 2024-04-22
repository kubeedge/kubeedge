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

package restful

import (
	"net/url"

	corev1 "k8s.io/api/core/v1"
)

type PodRequest struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	PodName       string
}

func (podRequest *PodRequest) GetPod() (*corev1.Pod, error) {
	request := Request{
		Method: "GET",
		Path: "/" + CoreAPIPrefix + "/" + CoreAPIGroupVersion.Version +
			"/namespaces/" + podRequest.Namespace + "/pods/" + podRequest.PodName,
	}

	pod, err := request.ResponseToPod()
	if err != nil {
		return nil, err
	}
	return pod, nil
}

func (podRequest *PodRequest) GetPods() (*corev1.PodList, error) {
	var request Request
	if podRequest.AllNamespaces {
		request = Request{
			Method: "GET",
			Path:   "/" + CoreAPIPrefix + "/" + CoreAPIGroupVersion.Version + "/pods",
		}
	} else {
		request = Request{
			Method: "GET",
			Path: "/" + CoreAPIPrefix + "/" + CoreAPIGroupVersion.Version +
				"/namespaces/" + podRequest.Namespace + "/pods",
		}
	}

	if podRequest.LabelSelector != "" {
		values := url.Values{}
		values.Set("labelSelector", podRequest.LabelSelector)
		queryParams := values.Encode()
		request.Path += "?" + queryParams
	}

	podList, err := request.ResponseToPodList()
	if err != nil {
		return nil, err
	}
	return podList, nil
}
