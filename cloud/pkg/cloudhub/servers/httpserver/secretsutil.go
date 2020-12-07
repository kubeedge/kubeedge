package httpserver

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/utils"
)

const (
	NamespaceSystem string = "kubeedge"

	TokenSecretName      string = "tokensecret"
	TokenDataName        string = "tokendata"
	CaSecretName         string = "casecret"
	CloudCoreSecretName  string = "cloudcoresecret"
	CaDataName           string = "cadata"
	CaKeyDataName        string = "cakeydata"
	CloudCoreCertName    string = "cloudcoredata"
	CloudCoreKeyDataName string = "cloudcorekeydata"
)

func GetSecret(secretName string, ns string) (*corev1.Secret, error) {
	cli, err := utils.KubeClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	return cli.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
}

// CreateSecret creates a secret
func CreateSecret(secret *corev1.Secret, ns string) error {
	cli, err := utils.KubeClient()
	if err != nil {
		return fmt.Errorf("failed to create KubeClient, error: %s", err)
	}
	if err := CreateNamespaceIfNeeded(cli, ns); err != nil {
		return fmt.Errorf("failed to create Namespace kubeedge, error: %s", err)
	}
	if _, err := cli.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if _, err := cli.CoreV1().Secrets(ns).Update(context.Background(), secret, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("failed to update the secret, namespace: %s, name: %s, err: %v", ns, secret.Name, err)
			}
		} else {
			return fmt.Errorf("failed to create the secret, namespace: %s, name: %s, err: %v", ns, secret.Name, err)
		}
	}
	return nil
}

func CreateTokenSecret(caHashAndToken []byte) error {
	token := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TokenSecretName,
			Namespace: NamespaceSystem,
		},
		Data: map[string][]byte{
			TokenDataName: caHashAndToken,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
	return CreateSecret(token, NamespaceSystem)
}

func CreateCaSecret(certDER, key []byte) error {
	caSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CaSecretName,
			Namespace: NamespaceSystem,
		},
		Data: map[string][]byte{
			CaDataName:    certDER,
			CaKeyDataName: key,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
	return CreateSecret(caSecret, NamespaceSystem)
}

func CreateCloudCoreSecret(certDER, key []byte) error {
	cloudCoreCert := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CloudCoreSecretName,
			Namespace: NamespaceSystem,
		},
		Data: map[string][]byte{
			CloudCoreCertName:    certDER,
			CloudCoreKeyDataName: key,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
	return CreateSecret(cloudCoreCert, NamespaceSystem)
}

func CreateNamespaceIfNeeded(cli *kubernetes.Clientset, ns string) error {
	c := cli.CoreV1()
	if _, err := c.Namespaces().Get(context.Background(), ns, metav1.GetOptions{}); err == nil {
		return nil
	}
	newNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: "",
		},
	}
	_, err := c.Namespaces().Create(context.Background(), newNs, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		err = nil
	}
	return err
}
