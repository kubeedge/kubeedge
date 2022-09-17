package client

import (
	"encoding/json"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
)

// ServiceAccountTokenGetter is interface to get client service account token
type ServiceAccountTokenGetter interface {
	ServiceAccountToken() ServiceAccountTokenInterface
}

// ServiceAccountTokenInterface is interface for client service account token
type ServiceAccountTokenInterface interface {
	GetServiceAccountToken(namespace string, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error)
}

type serviceAccountToken struct {
	send SendInterface
}

func newServiceAccountToken(s SendInterface) *serviceAccountToken {
	return &serviceAccountToken{
		send: s,
	}
}

func (c *serviceAccountToken) GetServiceAccountToken(namespace string, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	resource := fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeServiceAccountToken, name)
	tokenMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, tr)
	msg, err := c.send.SendSync(tokenMsg)
	if err != nil {
		klog.Errorf("get service account token from metaManager failed, err: %v", err)
		return nil, fmt.Errorf("get service account token from metaManager failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		klog.Errorf("parse message to serviceaccount token failed, err: %v", err)
		return nil, fmt.Errorf("marshal message to serviceaccount token failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == metamanager.MetaManagerModuleName {
		return handleServiceAccountTokenFromMetaDB(content)
	}
	return handleServiceAccountTokenFromMetaManager(content)
}

func handleServiceAccountTokenFromMetaDB(content []byte) (*authenticationv1.TokenRequest, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to serviceaccount list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("serviceaccount length from meta db is %d", len(lists))
	}

	var tokenRequest authenticationv1.TokenRequest
	err = json.Unmarshal([]byte(lists[0]), &tokenRequest)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to serviceaccount token from db failed, err: %v", err)
	}
	return &tokenRequest, nil
}

func handleServiceAccountTokenFromMetaManager(content []byte) (*authenticationv1.TokenRequest, error) {
	var serviceAccount authenticationv1.TokenRequest
	err := json.Unmarshal(content, &serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to service account failed, err: %v", err)
	}
	return &serviceAccount, nil
}
