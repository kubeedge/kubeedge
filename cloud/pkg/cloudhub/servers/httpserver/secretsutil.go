package httpserver

import (
	"fmt"
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/utils"
)

const (
	NamespaceSystem string = "kubeedge"

	TokenSecretName      string = "tokenSecret"
	TokenDataName        string = "tokenData"
	CaSecretName         string = "caSecret"
	CloudCoreSecretName  string = "cloudCoreSecret"
	CaDataName           string = "caData"
	CaKeyDataName        string = "caKeyData"
	CloudCoreDataName    string = "cloudCoreData"
	CloudCoreKeyDataName string = "cloudCoreKeyData"
)

func GetSecret(secretName string, ns string) (*v1.Secret, error) {
	cli, err := utils.KubeClient()
	if err != nil {
		fmt.Printf("%v", err)
	}
	return cli.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
}

// CreateSecret creates a secret
func CreateSecret(secret *v1.Secret, ns string) error {
	cli, err := utils.KubeClient()
	if err != nil {
		fmt.Printf("%v", err)
	}
	if _, err := cli.CoreV1().Secrets(ns).Create(secret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "unable to create secret")
		}
	}
	return nil
}

func CreateTokenSecret(caHashAndToken []byte) {
	token := &v1.Secret{
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
	CreateSecret(token, NamespaceSystem)
}

func CreateCaSecret(certDER, key []byte) {
	caSecret := &v1.Secret{
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
	CreateSecret(caSecret, NamespaceSystem)
}

func CreateCloudCoreSecret(certDER, key []byte) {
	cloudCoreCert := &v1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      CloudCoreSecretName,
			Namespace: NamespaceSystem,
		},
		Data: map[string][]byte{
			CloudCoreDataName:    certDER,
			CloudCoreKeyDataName: key,
		},
		StringData: map[string]string{},
		Type:       "Opaque",
	}
	CreateSecret(cloudCoreCert, NamespaceSystem)
}
