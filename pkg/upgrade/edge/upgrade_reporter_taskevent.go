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

package edge

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	fsmv1alpha1 "github.com/kubeedge/api/apis/fsm/v1alpha1"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

// Deprecated: New node jobs no longer use the event field.
// For compatibility with historical versions, It will be removed in v1.23.

const (
	// TaskTypeUpgrade used to select controller in the cloud
	TaskTypeUpgrade = "upgrade"
)

type TaskEventReporter struct {
	JobName   string
	EventType string
	Config    *edgeconfig.EdgeCoreConfig
}

func NewTaskEventReporter(jobName, eventType string,
	config *edgeconfig.EdgeCoreConfig,
) Reporter {
	return &TaskEventReporter{
		JobName:   jobName,
		EventType: eventType,
		Config:    config,
	}
}

func (r *TaskEventReporter) Report(err error) error {
	event := fsm.Event{Type: r.EventType}
	if err != nil {
		event.Action = fsmv1alpha1.ActionFailure
		event.Msg = err.Error()
	} else {
		event.Action = fsmv1alpha1.ActionSuccess
	}
	return ReportTaskResult(r.Config, TaskTypeUpgrade, r.JobName, event)
}

func ReportTaskResult(config *v1alpha2.EdgeCoreConfig, taskType, taskID string, event fsm.Event) error {
	resp := &commontypes.NodeTaskResponse{
		NodeName: config.Modules.Edged.HostnameOverride,
		Event:    event.Type,
		Action:   event.Action,
		Time:     time.Now().UTC().Format(time.RFC3339),
		Reason:   event.Msg,
	}
	edgeHub := config.Modules.EdgeHub
	var caCrt []byte
	caCertPath := edgeHub.TLSCAFile
	caCrt, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read ca: %v", err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(caCrt)

	certFile := edgeHub.TLSCertFile
	keyFile := edgeHub.TLSPrivateKeyFile
	cliCrt, err := tls.LoadX509KeyPair(certFile, keyFile)

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// use TLS configuration
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: false,
			Certificates:       []tls.Certificate{cliCrt},
		},
	}

	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}

	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal failed: %v", err)
	}
	url := edgeHub.HTTPServer + fmt.Sprintf("/task/%s/name/%s/node/%s/status", taskType, taskID, config.Modules.Edged.HostnameOverride)
	result, err := client.Post(url, "application/json", bytes.NewReader(respData))

	if err != nil {
		return fmt.Errorf("post http request failed: %v", err)
	}
	klog.Error("report result ", result)
	defer result.Body.Close()

	return nil
}
