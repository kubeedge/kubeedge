/*
Copyright 2025 The KubeEdge Authors.

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

// Package interlink provides an HTTP client for the InterLink API server,
// enabling KubeEdge EdgeD to offload pod execution to remote HPC/HTC
// backends (e.g., SLURM, HTCondor) via the InterLink plugin interface.
//
// InterLink API Spec: https://interlink-project.dev/docs/guides/api-reference
package interlink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const (
	// OffloadAnnotationKey is the annotation key used to mark pods for
	// InterLink offloading. Pods with this annotation set to
	// OffloadAnnotationValue will be routed to the InterLink API server
	// instead of the local container runtime.
	OffloadAnnotationKey = "kubeedge.io/offload-to"

	// OffloadAnnotationValue is the expected value of the offload annotation.
	OffloadAnnotationValue = "interlink"

	createPath  = "/create"
	deletePath  = "/delete"
	statusPath  = "/status"
	getLogsPath = "/getLogs"
)

// CreateRequest represents the request body for the InterLink /create endpoint.
type CreateRequest struct {
	// Pod is the Kubernetes Pod spec to be executed on the remote backend.
	Pod *v1.Pod `json:"pod"`
	// ConfigMaps associated with the pod.
	ConfigMaps []v1.ConfigMap `json:"configmaps,omitempty"`
	// Secrets associated with the pod.
	Secrets []v1.Secret `json:"secrets,omitempty"`
}

// DeleteRequest represents the request body for the InterLink /delete endpoint.
type DeleteRequest struct {
	// Pod is the Kubernetes Pod to be deleted from the remote backend.
	Pod *v1.Pod `json:"pod"`
}

// StatusRequest represents the request body for the InterLink /status endpoint.
type StatusRequest struct {
	// Pods is the list of pods to query status for.
	Pods []*v1.Pod `json:"pods"`
}

// PodStatusResponse represents a single pod's status from the InterLink API.
type PodStatusResponse struct {
	// PodName is the name of the pod.
	PodName string `json:"name"`
	// PodNamespace is the namespace of the pod.
	PodNamespace string `json:"namespace"`
	// UID is the pod UID.
	UID string `json:"UID"`
	// Containers holds the status of each container in the pod.
	Containers []ContainerStatusResponse `json:"containers"`
}

// ContainerStatusResponse represents a single container's status from InterLink.
type ContainerStatusResponse struct {
	// Name of the container.
	Name string `json:"name"`
	// State of the container.
	State v1.ContainerState `json:"state"`
	// ExitCode of the container, if terminated.
	ExitCode int32 `json:"exitCode"`
}

// LogRequest represents the request body for the InterLink /getLogs endpoint.
type LogRequest struct {
	// PodName is the name of the pod.
	PodName string `json:"podName"`
	// PodNamespace is the namespace of the pod.
	PodNamespace string `json:"podNamespace"`
	// ContainerName is the name of the container.
	ContainerName string `json:"containerName"`
	// Opts contains the log options (tail lines, follow, etc).
	Opts v1.PodLogOptions `json:"opts"`
}

// Client is an HTTP client for the InterLink API server.
type Client struct {
	// serverURL is the base URL of the InterLink API server (e.g., "http://localhost:3000").
	serverURL string
	// httpClient is the underlying HTTP client with configured timeout.
	httpClient *http.Client
}

// NewClient creates a new InterLink API client.
func NewClient(serverURL string, timeout time.Duration) *Client {
	return &Client{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Create submits a pod to the InterLink API server for execution on a remote backend.
// This translates to a POST /create call which, for example, triggers an sbatch
// command on a SLURM cluster.
func (c *Client) Create(pod *v1.Pod) error {
	req := CreateRequest{
		Pod: pod,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal create request for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	klog.V(4).Infof("InterLink: creating pod %s/%s on remote backend", pod.Namespace, pod.Name)

	resp, err := c.httpClient.Post(c.serverURL+createPath, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("InterLink: failed to create pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("InterLink: create pod %s/%s returned status %d: %s", pod.Namespace, pod.Name, resp.StatusCode, string(respBody))
	}

	klog.V(2).Infof("InterLink: successfully created pod %s/%s on remote backend", pod.Namespace, pod.Name)
	return nil
}

// Delete requests the InterLink API server to cancel/cleanup a remote job.
// For SLURM backends, this triggers scancel.
func (c *Client) Delete(pod *v1.Pod) error {
	req := DeleteRequest{
		Pod: pod,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request for pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}

	klog.V(4).Infof("InterLink: deleting pod %s/%s from remote backend", pod.Namespace, pod.Name)

	resp, err := c.httpClient.Post(c.serverURL+deletePath, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("InterLink: failed to delete pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("InterLink: delete pod %s/%s returned status %d: %s", pod.Namespace, pod.Name, resp.StatusCode, string(respBody))
	}

	klog.V(2).Infof("InterLink: successfully deleted pod %s/%s from remote backend", pod.Namespace, pod.Name)
	return nil
}

// Status queries the InterLink API server for the current status of pods.
// Returns a list of pod status responses that can be mapped to Kubernetes pod phases.
func (c *Client) Status(pods []*v1.Pod) ([]PodStatusResponse, error) {
	req := StatusRequest{
		Pods: pods,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status request: %w", err)
	}

	klog.V(4).Infof("InterLink: querying status for %d pod(s)", len(pods))

	httpReq, err := http.NewRequest(http.MethodGet, c.serverURL+statusPath, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("InterLink: failed to build status request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("InterLink: failed to get status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("InterLink: status returned %d: %s", resp.StatusCode, string(respBody))
	}

	var statuses []PodStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		return nil, fmt.Errorf("InterLink: failed to decode status response: %w", err)
	}

	return statuses, nil
}

// GetLogs fetches container logs from the remote backend via the InterLink API.
func (c *Client) GetLogs(podNamespace, podName, containerName string, opts v1.PodLogOptions) (string, error) {
	req := LogRequest{
		PodName:       podName,
		PodNamespace:  podNamespace,
		ContainerName: containerName,
		Opts:          opts,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal log request: %w", err)
	}

	klog.V(4).Infof("InterLink: fetching logs for %s/%s/%s", podNamespace, podName, containerName)

	httpReq, err := http.NewRequest(http.MethodGet, c.serverURL+getLogsPath, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("InterLink: failed to build logs request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("InterLink: failed to get logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("InterLink: getLogs returned %d: %s", resp.StatusCode, string(respBody))
	}

	logData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("InterLink: failed to read logs response: %w", err)
	}

	return string(logData), nil
}

// IsInterLinkPod checks whether a pod is annotated for InterLink offloading.
func IsInterLinkPod(pod *v1.Pod) bool {
	if pod == nil || pod.Annotations == nil {
		return false
	}
	return pod.Annotations[OffloadAnnotationKey] == OffloadAnnotationValue
}
