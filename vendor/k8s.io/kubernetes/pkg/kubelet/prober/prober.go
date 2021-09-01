/*
Copyright 2014 The Kubernetes Authors.

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

package prober

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/events"
	"k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	"k8s.io/kubernetes/pkg/probe"
	execprobe "k8s.io/kubernetes/pkg/probe/exec"
	httpprobe "k8s.io/kubernetes/pkg/probe/http"
	tcpprobe "k8s.io/kubernetes/pkg/probe/tcp"
	"k8s.io/utils/exec"

	"k8s.io/klog/v2"
)

const maxProbeRetries = 3

// Prober helps to check the liveness/readiness/startup of a container.
type prober struct {
	exec execprobe.Prober
	// probe types needs different httpprobe instances so they don't
	// share a connection pool which can cause collisions to the
	// same host:port and transient failures. See #49740.
	readinessHTTP httpprobe.Prober
	livenessHTTP  httpprobe.Prober
	startupHTTP   httpprobe.Prober
	tcp           tcpprobe.Prober
	runner        kubecontainer.CommandRunner

	recorder record.EventRecorder
}

// NewProber creates a Prober, it takes a command runner and
// several container info managers.
func newProber(
	runner kubecontainer.CommandRunner,
	recorder record.EventRecorder) *prober {

	const followNonLocalRedirects = false
	return &prober{
		exec:          execprobe.New(),
		readinessHTTP: httpprobe.New(followNonLocalRedirects),
		livenessHTTP:  httpprobe.New(followNonLocalRedirects),
		startupHTTP:   httpprobe.New(followNonLocalRedirects),
		tcp:           tcpprobe.New(),
		runner:        runner,
		recorder:      recorder,
	}
}

// recordContainerEvent should be used by the prober for all container related events.
func (pb *prober) recordContainerEvent(pod *v1.Pod, container *v1.Container, eventType, reason, message string, args ...interface{}) {
	ref, err := kubecontainer.GenerateContainerRef(pod, container)
	if err != nil {
		klog.ErrorS(err, "Can't make a ref to pod and container", "pod", klog.KObj(pod), "containerName", container.Name)
		return
	}
	pb.recorder.Eventf(ref, eventType, reason, message, args...)
}

// probe probes the container.
func (pb *prober) probe(probeType probeType, pod *v1.Pod, status v1.PodStatus, container v1.Container, containerID kubecontainer.ContainerID) (results.Result, error) {
	var probeSpec *v1.Probe
	switch probeType {
	case readiness:
		probeSpec = container.ReadinessProbe
	case liveness:
		probeSpec = container.LivenessProbe
	case startup:
		probeSpec = container.StartupProbe
	default:
		return results.Failure, fmt.Errorf("unknown probe type: %q", probeType)
	}

	if probeSpec == nil {
		klog.InfoS("Probe is nil", "probeType", probeType, "pod", klog.KObj(pod), "podUID", pod.UID, "containerName", container.Name)
		return results.Success, nil
	}

	result, output, err := pb.runProbeWithRetries(probeType, probeSpec, pod, status, container, containerID, maxProbeRetries)
	if err != nil || (result != probe.Success && result != probe.Warning) {
		// Probe failed in one way or another.
		if err != nil {
			klog.V(1).ErrorS(err, "Probe errored", "probeType", probeType, "pod", klog.KObj(pod), "podUID", pod.UID, "containerName", container.Name)
			pb.recordContainerEvent(pod, &container, v1.EventTypeWarning, events.ContainerUnhealthy, "%s probe errored: %v", probeType, err)
		} else { // result != probe.Success
			klog.V(1).InfoS("Probe failed", "probeType", probeType, "pod", klog.KObj(pod), "podUID", pod.UID, "containerName", container.Name, "probeResult", result, "output", output)
			pb.recordContainerEvent(pod, &container, v1.EventTypeWarning, events.ContainerUnhealthy, "%s probe failed: %s", probeType, output)
		}
		return results.Failure, err
	}
	if result == probe.Warning {
		pb.recordContainerEvent(pod, &container, v1.EventTypeWarning, events.ContainerProbeWarning, "%s probe warning: %s", probeType, output)
		klog.V(3).InfoS("Probe succeeded with a warning", "probeType", probeType, "pod", klog.KObj(pod), "podUID", pod.UID, "containerName", container.Name, "output", output)
	} else {
		klog.V(3).InfoS("Probe succeeded", "probeType", probeType, "pod", klog.KObj(pod), "podUID", pod.UID, "containerName", container.Name)
	}
	return results.Success, nil
}

// runProbeWithRetries tries to probe the container in a finite loop, it returns the last result
// if it never succeeds.
func (pb *prober) runProbeWithRetries(probeType probeType, p *v1.Probe, pod *v1.Pod, status v1.PodStatus, container v1.Container, containerID kubecontainer.ContainerID, retries int) (probe.Result, string, error) {
	var err error
	var result probe.Result
	var output string
	for i := 0; i < retries; i++ {
		result, output, err = pb.runProbe(probeType, p, pod, status, container, containerID)
		if err == nil {
			return result, output, nil
		}
	}
	return result, output, err
}

// buildHeaderMap takes a list of HTTPHeader <name, value> string
// pairs and returns a populated string->[]string http.Header map.
func buildHeader(headerList []v1.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}

func (pb *prober) runProbe(probeType probeType, p *v1.Probe, pod *v1.Pod, status v1.PodStatus, container v1.Container, containerID kubecontainer.ContainerID) (probe.Result, string, error) {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if p.Exec != nil {
		klog.V(4).InfoS("Exec-Probe runProbe", "pod", klog.KObj(pod), "containerName", container.Name, "execCommand", p.Exec.Command)
		command := kubecontainer.ExpandContainerCommandOnlyStatic(p.Exec.Command, container.Env)
		return pb.exec.Probe(pb.newExecInContainer(container, containerID, command, timeout))
	}
	if p.HTTPGet != nil {
		scheme := strings.ToLower(string(p.HTTPGet.Scheme))
		host := p.HTTPGet.Host
		if host == "" {
			host = status.PodIP
		}
		port, err := extractPort(p.HTTPGet.Port, container)
		if err != nil {
			return probe.Unknown, "", err
		}
		path := p.HTTPGet.Path
		klog.V(4).InfoS("HTTP-Probe Host", "scheme", scheme, "host", host, "port", port, "path", path)
		url := formatURL(scheme, host, port, path)
		headers := buildHeader(p.HTTPGet.HTTPHeaders)
		klog.V(4).InfoS("HTTP-Probe Headers", "headers", headers)
		switch probeType {
		case liveness:
			return pb.livenessHTTP.Probe(url, headers, timeout)
		case startup:
			return pb.startupHTTP.Probe(url, headers, timeout)
		default:
			return pb.readinessHTTP.Probe(url, headers, timeout)
		}
	}
	if p.TCPSocket != nil {
		port, err := extractPort(p.TCPSocket.Port, container)
		if err != nil {
			return probe.Unknown, "", err
		}
		host := p.TCPSocket.Host
		if host == "" {
			host = status.PodIP
		}
		klog.V(4).InfoS("TCP-Probe Host", "host", host, "port", port, "timeout", timeout)
		return pb.tcp.Probe(host, port, timeout)
	}
	klog.InfoS("Failed to find probe builder for container", "containerName", container.Name)
	return probe.Unknown, "", fmt.Errorf("missing probe handler for %s:%s", format.Pod(pod), container.Name)
}

func extractPort(param intstr.IntOrString, container v1.Container) (int, error) {
	port := -1
	var err error
	switch param.Type {
	case intstr.Int:
		port = param.IntValue()
	case intstr.String:
		if port, err = findPortByName(container, param.StrVal); err != nil {
			// Last ditch effort - maybe it was an int stored as string?
			if port, err = strconv.Atoi(param.StrVal); err != nil {
				return port, err
			}
		}
	default:
		return port, fmt.Errorf("intOrString had no kind: %+v", param)
	}
	if port > 0 && port < 65536 {
		return port, nil
	}
	return port, fmt.Errorf("invalid port number: %v", port)
}

// findPortByName is a helper function to look up a port in a container by name.
func findPortByName(container v1.Container, portName string) (int, error) {
	for _, port := range container.Ports {
		if port.Name == portName {
			return int(port.ContainerPort), nil
		}
	}
	return 0, fmt.Errorf("port %s not found", portName)
}

// formatURL formats a URL from args.  For testability.
func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}

type execInContainer struct {
	// run executes a command in a container. Combined stdout and stderr output is always returned. An
	// error is returned if one occurred.
	run    func() ([]byte, error)
	writer io.Writer
}

func (pb *prober) newExecInContainer(container v1.Container, containerID kubecontainer.ContainerID, cmd []string, timeout time.Duration) exec.Cmd {
	return &execInContainer{run: func() ([]byte, error) {
		return pb.runner.RunInContainer(containerID, cmd, timeout)
	}}
}

func (eic *execInContainer) Run() error {
	return nil
}

func (eic *execInContainer) CombinedOutput() ([]byte, error) {
	return eic.run()
}

func (eic *execInContainer) Output() ([]byte, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (eic *execInContainer) SetDir(dir string) {
	//unimplemented
}

func (eic *execInContainer) SetStdin(in io.Reader) {
	//unimplemented
}

func (eic *execInContainer) SetStdout(out io.Writer) {
	eic.writer = out
}

func (eic *execInContainer) SetStderr(out io.Writer) {
	eic.writer = out
}

func (eic *execInContainer) SetEnv(env []string) {
	//unimplemented
}

func (eic *execInContainer) Stop() {
	//unimplemented
}

func (eic *execInContainer) Start() error {
	data, err := eic.run()
	if eic.writer != nil {
		eic.writer.Write(data)
	}
	return err
}

func (eic *execInContainer) Wait() error {
	return nil
}

func (eic *execInContainer) StdoutPipe() (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (eic *execInContainer) StderrPipe() (io.ReadCloser, error) {
	return nil, fmt.Errorf("unimplemented")
}
