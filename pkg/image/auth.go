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

package image

import (
	"github.com/distribution/reference"
	corev1 "k8s.io/api/core/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/credentialprovider"
	credentialprovidersecrets "k8s.io/kubernetes/pkg/credentialprovider/secrets"
)

// PullSecretAuth looks up the registry credentials for images in an image pull secret,
// the way the kubelet does for the imagePullSecrets of a pod.
type PullSecretAuth struct {
	keyring credentialprovider.DockerKeyring
}

// NewPullSecretAuth reads the registry credentials of a kubernetes.io/dockerconfigjson or
// kubernetes.io/dockercfg secret. A secret of any other type has no credentials.
func NewPullSecretAuth(secret *corev1.Secret) (*PullSecretAuth, error) {
	keyring, err := credentialprovidersecrets.MakeDockerKeyring(
		[]corev1.Secret{*secret}, &credentialprovider.BasicDockerKeyring{})
	if err != nil {
		return nil, err
	}
	return &PullSecretAuth{keyring: keyring}, nil
}

// ForImage returns the credentials for the registry of image, or nil when the secret has
// none for it, in which case the image is pulled without credentials.
func (a *PullSecretAuth) ForImage(image string) *runtimeapi.AuthConfig {
	if a == nil {
		return nil
	}
	repo := image
	if named, err := reference.ParseNormalizedNamed(image); err == nil {
		repo = named.Name()
	}
	creds, ok := a.keyring.Lookup(repo)
	if !ok || len(creds) == 0 {
		return nil
	}
	return &runtimeapi.AuthConfig{
		Username:      creds[0].Username,
		Password:      creds[0].Password,
		Auth:          creds[0].Auth,
		ServerAddress: creds[0].ServerAddress,
		IdentityToken: creds[0].IdentityToken,
		RegistryToken: creds[0].RegistryToken,
	}
}
