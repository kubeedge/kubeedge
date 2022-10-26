/*
Copyright 2022 The KubeEdge Authors.

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

package metric

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/handler"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	cloudcoreConfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// DefaultHealthResponseBody define the response body of health handler
const DefaultHealthResponseBody = "everything is fine\n"

func metricHandler(w http.ResponseWriter, r *http.Request) {
	_, err := io.WriteString(w, informationFormat())
	if err != nil {
		return
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(DefaultHealthResponseBody))
	if err != nil {
		return
	}
}

// StartMetricServer start a metric server for controller
func StartMetricServer(config *cloudcoreConfig.CommonConfig) {
	address := fmt.Sprintf("%s:%d", config.Host, config.MetricPort)

	router := mux.NewRouter()
	router.HandleFunc(config.MetricPatten, metricHandler)
	router.HandleFunc(config.HealthPatten, healthHandler)

	cert, err := tls.X509KeyPair(hubconfig.Config.Cert, hubconfig.Config.Key)
	if err != nil {
		klog.Fatalf("generate key pair failed with error: %s", err)
		os.Exit(1)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
	}

	s := http.Server{
		Addr:      address,
		Handler:   router,
		TLSConfig: tlsConfig,
	}
	err = s.ListenAndServeTLS("", "")
	// err will always be not nil in real run, we check here for unit test purpose
	if err != nil {
		klog.Fatalf("Start cloudcore metric server failed, error: %s", err.Error())
		os.Exit(1)
	}
}

// MasterMetric is the function register to metric return controller master failure rate
func MasterMetric(messageHandler handler.Handler) (string, string, string, string) {
	value := 1
	kubeClient := client.GetKubeClient()
	_, err := kubeClient.CoreV1().Namespaces().List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		value = 0
	}
	return "cloudcore_edgemaster_connectivity", Gauge, "cloudcore to edge master connectivity, 1: connected, 0: disconnected", fmt.Sprintf("%d", value)
}
