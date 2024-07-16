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
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/security/token"
)

// SignCerts creates server's certificate and key
func SignCerts() ([]byte, []byte, error) {
	cfg := &certutil.Config{
		CommonName:   constants.ProjectName,
		Organization: []string{constants.ProjectName},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: certutil.AltNames{
			DNSNames: hubconfig.Config.DNSNames,
			IPs:      getIps(hubconfig.Config.AdvertiseAddress),
		},
	}

	certDER, keyDER, err := NewCloudCoreCertDERandKey(cfg)
	if err != nil {
		return nil, nil, err
	}

	return certDER, keyDER, nil
}

func getIps(advertiseAddress []string) (Ips []net.IP) {
	for _, addr := range advertiseAddress {
		Ips = append(Ips, net.ParseIP(addr))
	}
	return
}

// GenerateAndRefresh creates a token and save it to secret, then craete a timer to refresh the token.
func GenerateAndRefresh(ctx context.Context) error {
	caHashToken, err := token.Create(hubconfig.Config.Ca, hubconfig.Config.CaKey,
		hubconfig.Config.CloudHub.TokenRefreshDuration)
	if err != nil {
		return fmt.Errorf("failed to generate the token for edgecore register, err: %v", err)
	}
	// save caHashAndToken to secret
	err = CreateTokenSecret([]byte(caHashToken))
	if err != nil {
		return fmt.Errorf("failed to create tokenSecret, err: %v", err)
	}

	t := time.NewTicker(time.Hour * hubconfig.Config.CloudHub.TokenRefreshDuration)
	go func() {
		for {
			select {
			case <-t.C:
				caHashToken, err = token.Create(hubconfig.Config.Ca, hubconfig.Config.CaKey,
					hubconfig.Config.CloudHub.TokenRefreshDuration)
				if err != nil {
					klog.Error("failed to refresh the token for edgecore register, err: %v", err)
				}
			case <-ctx.Done():
				break
			}
		}
	}()
	klog.Info("Succeed to creating token")
	return nil
}
