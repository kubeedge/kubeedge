/*
Copyright 2019 The KubeEdge Authors.

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

package options

import (
	cliflag "k8s.io/component-base/cli/flag"
)

// AdmissionOptions admission-controller server config.
type AdmissionOptions struct {
	Master                    string
	Kubeconfig                string
	CertFile                  string
	KeyFile                   string
	CaCertFile                string
	Port                      int32
	PrintVersion              bool
	AdmissionServiceName      string
	AdmissionServiceNamespace string
	SchedulerName             string
}

// NewAdmissionOptions create new config
func NewAdmissionOptions() *AdmissionOptions {
	return &AdmissionOptions{}
}

// Flags add flags for admission webhook
func (o *AdmissionOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("admission")
	fs.StringVar(&o.Master, "master", o.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	fs.StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")
	fs.StringVar(&o.CertFile, "tls-cert-file", o.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+"after server cert).")
	fs.StringVar(&o.KeyFile, "tls-private-key-file", o.KeyFile, "File containing the default x509 private key matching --tls-cert-file.")
	fs.StringVar(&o.CaCertFile, "ca-cert-file", o.CaCertFile, "File containing the x509 Certificate for HTTPS.")
	fs.Int32Var(&o.Port, "port", 443, "the port used by admission-controller-server.")
	fs.StringVar(&o.AdmissionServiceNamespace, "webhook-namespace", "kubeedge", "The namespace of this webhook")
	fs.StringVar(&o.AdmissionServiceName, "webhook-service-name", "kubeedge-admission-service", "The name of this admission service")
	return
}
