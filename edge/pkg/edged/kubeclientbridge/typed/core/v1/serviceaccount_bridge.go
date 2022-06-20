package v1

import (
	"context"

	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// ServiceAccountsBridge implements ServiceAccountInterface
type ServiceAccountsBridge struct {
	fakecorev1.FakeServiceAccounts
	ns         string
	MetaClient client.CoreInterface
}

// CreateToken takes the representation of a tokenRequest and creates it.  Returns the server's representation of the tokenRequest, and an error, if there is any.
func (c *ServiceAccountsBridge) CreateToken(ctx context.Context, serviceAccountName string, tokenRequest *authenticationv1.TokenRequest, opts metav1.CreateOptions) (result *authenticationv1.TokenRequest, err error) {
	return c.MetaClient.ServiceAccountToken().GetServiceAccountToken(c.ns, serviceAccountName, tokenRequest)
}
