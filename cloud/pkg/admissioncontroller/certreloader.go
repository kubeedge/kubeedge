/*
Copyright 2026 The KubeEdge Authors.

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

package admissioncontroller

import (
	"bytes"
	"crypto/tls"
	"os"
	"sync"

	"k8s.io/klog/v2"
)

// certReloader serves the webhook certificate from the files it was loaded from and
// reloads it whenever they change. The certificate is mounted from a secret that can
// be rotated while the process runs, and without reloading the webhook keeps serving
// the certificate read at startup, which stops matching the CA bundle published in
// the webhook configurations once the CA is rotated too.
type certReloader struct {
	certFile string
	keyFile  string

	// mutex guards the cached key pair and the PEM it was loaded from.
	mutex   sync.Mutex
	certPEM []byte
	keyPEM  []byte
	cert    *tls.Certificate
}

// newCertReloader returns a reloader serving the key pair held by certFile and keyFile.
func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	reloader := &certReloader{certFile: certFile, keyFile: keyFile}
	if err := reloader.load(); err != nil {
		return nil, err
	}
	return reloader, nil
}

// load caches the key pair on disk and does nothing when it has not changed since the
// last successful load. The cached key pair is left alone when the files cannot be
// read or do not parse, which also covers reading the certificate and the key while
// they are rotated and picking up halves of two different key pairs.
func (r *certReloader) load() error {
	certPEM, err := os.ReadFile(r.certFile)
	if err != nil {
		return err
	}
	keyPEM, err := os.ReadFile(r.keyFile)
	if err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if bytes.Equal(certPEM, r.certPEM) && bytes.Equal(keyPEM, r.keyPEM) {
		return nil
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return err
	}
	r.certPEM, r.keyPEM, r.cert = certPEM, keyPEM, &cert
	klog.Infof("Loaded the certificate from %s", r.certFile)
	return nil
}

// GetCertificate is the tls.Config callback returning the certificate to serve.
func (r *certReloader) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if err := r.load(); err != nil {
		klog.Errorf("failed to reload the certificate, serving the previously loaded one: %v", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.cert, nil
}
