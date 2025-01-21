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

package client

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type PodRequest struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	PodName       string
}

func (podRequest *PodRequest) GetPod(ctx context.Context) (*corev1.Pod, error) {
	kubeClient, err := KubeClient()
	if err != nil {
		return nil, err
	}
	pod, err := kubeClient.CoreV1().Pods(podRequest.Namespace).Get(ctx, podRequest.PodName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	pod.APIVersion = common.PodAPIVersion
	pod.Kind = common.PodKind
	return pod, nil
}

func (podRequest *PodRequest) GetPods(ctx context.Context) (*corev1.PodList, error) {
	kubeClient, err := KubeClient()
	if err != nil {
		return nil, err
	}
	if podRequest.AllNamespaces {
		podList, err := kubeClient.CoreV1().Pods(metaV1.NamespaceAll).List(ctx, metaV1.ListOptions{
			LabelSelector: podRequest.LabelSelector,
		})
		if err != nil {
			return nil, err
		}
		return podList, nil
	}

	podList, err := kubeClient.CoreV1().Pods(podRequest.Namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: podRequest.LabelSelector,
	})
	if err != nil {
		return nil, err
	}
	return podList, nil
}
