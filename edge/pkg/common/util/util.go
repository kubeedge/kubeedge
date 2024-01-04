package util

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

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

const ISO8601UTC = "2006-01-02T15:04:05Z"

func ReportUpgradeResult(config *v1alpha2.EdgeCoreConfig, taskType, taskID string, event fsm.Event) error {
	resp := &v1alpha1.TaskStatus{
		NodeName: config.Modules.Edged.HostnameOverride,
		Event:    event.Type,
		Action:   event.Action,
		Time:     time.Now().Format(ISO8601UTC),
		Reason:   event.ErrorMsg,
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
