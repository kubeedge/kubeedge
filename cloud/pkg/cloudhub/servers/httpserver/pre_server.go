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
package httpserver

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
	"github.com/kubeedge/kubeedge/pkg/security/token"
)

const (
	TokenSecretName      string = "tokensecret"
	TokenDataName        string = "tokendata"
	CaSecretName         string = "casecret"
	CloudCoreSecretName  string = "cloudcoresecret"
	CaDataName           string = "cadata"
	CaKeyDataName        string = "cakeydata"
	CloudCoreCertName    string = "cloudcoredata"
	CloudCoreKeyDataName string = "cloudcorekeydata"
)

// PrepareAllCerts check whether the certificates exist in the local directory,
// and then check whether certificates exist in the secret, generate if they don't exist
func PrepareAllCerts(ctx context.Context) error {
	if err := createCAToSecret(ctx); err != nil {
		return err
	}
	return createCertsToSecret(ctx)
}

func createCAToSecret(ctx context.Context) error {
	var caDER, keyDER []byte
	// Check whether the ca exists in the local directory
	if hubconfig.Config.Ca == nil && hubconfig.Config.CaKey == nil {
		klog.Info("Ca and CaKey don't exist in local directory, and will read from the secret")

		// Check whether the ca exists in the secret
		caSecret, err := client.GetSecret(ctx, CaSecretName, constants.SystemNamespace)
		if err != nil {
			if !apierror.IsNotFound(err) {
				return fmt.Errorf("get secret: %s error: %v", CaSecretName, err)
			}

			klog.Info("Ca and CaKey don't exist in the secret, and will be created by CloudCore")
			h := certs.GetCAHandler(certs.CAHandlerTypeX509)
			pk, err := h.GenPrivateKey()
			if err != nil {
				return err
			}

			caPem, err := h.NewSelfSigned(pk)
			if err != nil {
				return fmt.Errorf("failed to create Certificate Authority, error: %v", err)
			}
			caDER = caPem.Bytes
			keyDER = pk.DER()
		} else {
			caDER = caSecret.Data[CaDataName]
			keyDER = caSecret.Data[CaKeyDataName]
		}

		hubconfig.Config.UpdateCA(caDER, keyDER)
	} else {
		// HubConfig has been initialized
		caDER = hubconfig.Config.Ca
		keyDER = hubconfig.Config.CaKey
	}

	if err := client.SaveSecret(ctx, createCaSecret(caDER, keyDER), constants.SystemNamespace); err != nil {
		return fmt.Errorf("failed to create ca to secrets, error: %v", err)
	}

	return nil
}

func createCertsToSecret(ctx context.Context) error {
	const year100 = time.Hour * 24 * 364 * 100
	var certDER, keyDER []byte

	// Check whether the CloudCore certificates exist in the local directory
	if hubconfig.Config.Key == nil && hubconfig.Config.Cert == nil {
		klog.Infof("CloudCoreCert and key don't exist in local directory, and will read from the secret")

		// Check whether the CloudCore certificates exist in the secret
		cloudSecret, err := client.GetSecret(ctx, CloudCoreSecretName, constants.SystemNamespace)
		if err != nil {
			if !apierror.IsNotFound(err) {
				return fmt.Errorf("get secret: %s error: %v", CloudCoreSecretName, err)
			}

			klog.Info("CloudCoreCert and key don't exist in the secret, and will be signed by CA")

			ips := make([]net.IP, 0, len(hubconfig.Config.AdvertiseAddress))
			for _, addr := range hubconfig.Config.AdvertiseAddress {
				ips = append(ips, net.ParseIP(addr))
			}
			h := certs.GetHandler(certs.HandlerTypeX509)

			keywrap, err := h.GenPrivateKey()
			if err != nil {
				return fmt.Errorf("failed to generate the private key, err: %v", err)
			}
			key, err := keywrap.Signer()
			if err != nil {
				return fmt.Errorf("failed parse the priavte key, err: %v", err)
			}

			opts := certs.SignCertsOptionsWithCA(certutil.Config{
				CommonName:   constants.ProjectName,
				Organization: []string{constants.ProjectName},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				AltNames: certutil.AltNames{
					DNSNames: hubconfig.Config.DNSNames,
					IPs:      ips,
				},
			}, hubconfig.Config.Ca, hubconfig.Config.CaKey, key.Public(), year100)
			certPEM, err := h.SignCerts(opts)
			if err != nil {
				return fmt.Errorf("failed to sign the certificate, err: %v", err)
			}
			keyDER = keywrap.DER()
			certDER = certPEM.Bytes
		} else {
			certDER = cloudSecret.Data[CloudCoreCertName]
			keyDER = cloudSecret.Data[CloudCoreKeyDataName]
		}

		hubconfig.Config.UpdateCerts(certDER, keyDER)
	} else {
		// HubConfig has been initialized
		certDER = hubconfig.Config.Cert
		keyDER = hubconfig.Config.Key
	}

	if err := client.SaveSecret(ctx, createCloudCoreSecret(certDER, keyDER), constants.SystemNamespace); err != nil {
		return fmt.Errorf("failed to save CloudCore cert and key to secret, error: %v", err)
	}

	return nil
}

// GenerateAndRefreshToken creates a token and save it to secret, then craete a timer to refresh the token.
func GenerateAndRefreshToken(ctx context.Context) error {
	if err := createNewToken(ctx); err != nil {
		return err
	}
	t := time.NewTicker(time.Hour * hubconfig.Config.CloudHub.TokenRefreshDuration)
	go func() {
		for {
			select {
			case <-t.C:
				if err := createNewToken(ctx); err != nil {
					klog.Warningf("failed to refresh the new token, err: %v", err)
					return
				}
				klog.Info("token refreshed successfully")
			case <-ctx.Done():
				break
			}
		}
	}()
	klog.Info("token created successfully")
	return nil
}

func createNewToken(ctx context.Context) error {
	caHashToken, err := token.Create(hubconfig.Config.Ca, hubconfig.Config.CaKey,
		hubconfig.Config.CloudHub.TokenRefreshDuration)
	if err != nil {
		return fmt.Errorf("failed to generate the token for edgecore register, err: %v", err)
	}
	// save caHashAndToken to secret
	if err := client.SaveSecret(ctx, createTokenSecret([]byte(caHashToken)), constants.SystemNamespace); err != nil {
		return fmt.Errorf("failed to create tokenSecret, err: %v", err)
	}
	return nil
}

func createTokenSecret(caHashAndToken []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TokenSecretName,
			Namespace: constants.SystemNamespace,
		},
		Data: map[string][]byte{
			TokenDataName: caHashAndToken,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
}

func createCaSecret(certDER, key []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CaSecretName,
			Namespace: constants.SystemNamespace,
		},
		Data: map[string][]byte{
			CaDataName:    certDER,
			CaKeyDataName: key,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
}

func createCloudCoreSecret(certDER, key []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CloudCoreSecretName,
			Namespace: constants.SystemNamespace,
		},
		Data: map[string][]byte{
			CloudCoreCertName:    certDER,
			CloudCoreKeyDataName: key,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
}
