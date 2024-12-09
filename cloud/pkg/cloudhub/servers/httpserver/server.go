/*
Copyright 2020 The KubeEdge Authors.

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
package httpserver

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
	certutil "k8s.io/client-go/util/cert"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	certshandler "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver/certificate"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver/node"
	nodetaskhandler "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/servers/httpserver/nodetask"
	"github.com/kubeedge/kubeedge/common/constants"
)

// StartHTTPServer starts the http service
func StartHTTPServer() error {
	serverContainer := restful.NewContainer()
	serverContainer.Add(routes())
	addr := fmt.Sprintf("%s:%d", hubconfig.Config.HTTPS.Address, hubconfig.Config.HTTPS.Port)
	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: hubconfig.Config.Cert}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: hubconfig.Config.Key}),
	)
	if err != nil {
		return fmt.Errorf("failed to create a x509 tls certificate")
	}

	server := &http.Server{
		Addr:    addr,
		Handler: serverContainer,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequestClientCert,
		},
	}
	return server.ListenAndServeTLS("", "")
}

func routes() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/")
	ws.Route(ws.GET(constants.DefaultCertURL).To(certshandler.EdgeCoreClientCert))
	ws.Route(ws.GET(constants.DefaultCAURL).To(certshandler.GetCA))
	ws.Route(ws.GET(constants.DefaultCheckNodeURL).To(node.CheckNode))
	ws.Route(ws.POST(constants.DefaultNodeUpgradeURL).To(nodetaskhandler.UpgradeEdge))
	ws.Route(ws.POST(constants.DefaultTaskStateReportURL).To(nodetaskhandler.ReportStatus))
	return ws
}
