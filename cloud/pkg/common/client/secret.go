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
package client

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSecret ...
func GetSecret(ctx context.Context, secretName string, ns string) (*corev1.Secret, error) {
	cli := GetKubeClient()
	return cli.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
}

// saveSecret creates a secret when it does not exist, otherwise updates it.
func SaveSecret(ctx context.Context, secret *corev1.Secret, ns string) error {
	cli := GetKubeClient()
	if err := CreateNamespaceIfNeeded(ctx, ns); err != nil {
		return fmt.Errorf("failed to create Namespace kubeedge, error: %v", err)
	}
	if _, err := cli.CoreV1().Secrets(ns).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if _, err := cli.CoreV1().Secrets(ns).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to update the secret, namespace: %s, name: %s, err: %v", ns, secret.Name, err)
			}
		} else {
			return fmt.Errorf("failed to create the secret, namespace: %s, name: %s, err: %v", ns, secret.Name, err)
		}
	}
	return nil
}
