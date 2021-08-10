package client

import (
	"encoding/json"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// ServiceAccountsGetter is interface to get client service accounts
type ServiceAccountsGetter interface {
	ServiceAccounts() ServiceAccountsInterface
}

// ServiceAccountsInterface is interface for client service account token
type ServiceAccountsInterface interface {
	GetServiceAccountToken(namespace string, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error)
}

type serviceAccounts struct {
	send SendInterface
}

func newServiceAccounts(s SendInterface) *serviceAccounts {
	return &serviceAccounts{
		send: s,
	}
}

func (c *serviceAccounts) GetServiceAccountToken(namespace string, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	resource := fmt.Sprintf("%s/%s/%s", namespace, constants.ResourceTypeServiceAccount, name)
	tokenMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, constants.OperationTypeGetServiceAccount, tr)
	msg, err := c.send.SendSync(tokenMsg)
	if err != nil {
		klog.Errorf("get service account token failed, err: %v", err)
		return nil, fmt.Errorf("get service account token failed, err: %v", err)
	}

	var content []byte
	switch msg.Content.(type) {
	case []byte:
		content = msg.GetContent().([]byte)
	default:
		content, err = json.Marshal(msg.GetContent())
		if err != nil {
			klog.Errorf("marshal message to serviceaccount token failed, err: %v", err)
			return nil, fmt.Errorf("marshal message to serviceaccount token failed, err: %v", err)
		}
	}

	return handleServiceAccount(content)
}

func handleServiceAccount(content []byte) (*authenticationv1.TokenRequest, error) {
	var serviceAccount authenticationv1.TokenRequest
	err := json.Unmarshal(content, &serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service account failed, err: %v", err)
	}
	return &serviceAccount, nil
}
