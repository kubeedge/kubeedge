package client

import (
	"encoding/json"
	"fmt"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	policyv1alpha1 "github.com/kubeedge/api/apis/policy/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

// ServiceAccountTokenGetter is interface to get client service account token
type ServiceAccountTokenGetter interface {
	ServiceAccountToken() ServiceAccountTokenInterface
}

// ServiceAccountTokenInterface is interface for client service account token
type ServiceAccountTokenInterface interface {
	GetServiceAccountToken(namespace string, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error)
	DeleteServiceAccountToken(podUID types.UID)
}

type serviceAccountToken struct {
	send SendInterface
}

const maxTTL = 24 * time.Hour

func newServiceAccountToken(s SendInterface) *serviceAccountToken {
	return &serviceAccountToken{
		send: s,
	}
}

func (c *serviceAccountToken) DeleteServiceAccountToken(podUID types.UID) {
	svcAccounts, err := dao.QueryAllMeta("type", model.ResourceTypeServiceAccountToken)
	if err != nil {
		klog.Errorf("query meta failed: %v", err)
		return
	}
	for _, sa := range *svcAccounts {
		var tr authenticationv1.TokenRequest
		err = json.Unmarshal([]byte(sa.Value), &tr)
		if err != nil || tr.Spec.BoundObjectRef == nil {
			klog.Errorf("unmarshal resource %s token request failed: %v", sa.Key, err)
			continue
		}
		if podUID == tr.Spec.BoundObjectRef.UID {
			err := dao.DeleteMetaByKey(sa.Key)
			if err != nil {
				klog.Errorf("delete meta %s failed: %v", sa.Key, err)
				return
			}
		}
	}
}

// requiresRefresh returns true if the token is older than 80% of its total
// ttl, or if the token is older than 24 hours.
func requiresRefresh(tr *authenticationv1.TokenRequest) bool {
	if tr.Spec.ExpirationSeconds == nil {
		cpy := tr.DeepCopy()
		cpy.Status.Token = ""
		klog.Errorf("expiration seconds was nil for tr: %#v", cpy)
		return false
	}
	now := time.Now()
	exp := tr.Status.ExpirationTimestamp.Time
	iat := exp.Add(-1 * time.Duration(*tr.Spec.ExpirationSeconds) * time.Second)

	if now.After(iat.Add(maxTTL)) {
		return true
	}
	// Require a refresh if within 20% of the TTL from the expiration time.
	if now.After(exp.Add(-1 * time.Duration((*tr.Spec.ExpirationSeconds*20)/100) * time.Second)) {
		return true
	}
	return false
}

// KeyFunc keys should be nonconfidential and safe to log
func KeyFunc(name, namespace string, tr *authenticationv1.TokenRequest) string {
	var exp int64
	if tr.Spec.ExpirationSeconds != nil {
		exp = *tr.Spec.ExpirationSeconds
	}

	var ref authenticationv1.BoundObjectReference
	if tr.Spec.BoundObjectRef != nil {
		ref = *tr.Spec.BoundObjectRef
	}

	return fmt.Sprintf("%q/%q/%#v/%#v/%#v", name, namespace, tr.Spec.Audiences, exp, ref)
}

func getTokenLocally(name, namespace string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	resKey := KeyFunc(name, namespace, tr)
	metas, err := dao.QueryMeta("key", resKey)
	if err != nil {
		klog.Errorf("query meta %s failed: %v", resKey, err)
		return nil, err
	}
	if len(*metas) != 1 {
		klog.Errorf("query meta %s length error", resKey)
		return nil, fmt.Errorf("query meta %s length error", resKey)
	}
	var tokenRequest authenticationv1.TokenRequest
	err = json.Unmarshal([]byte((*metas)[0]), &tokenRequest)
	if err != nil {
		klog.Errorf("unmarshal resource %s token request failed: %v", resKey, err)
		return nil, err
	}
	if requiresRefresh(&tokenRequest) {
		err := dao.DeleteMetaByKey(resKey)
		if err != nil {
			klog.Errorf("delete meta %s failed: %v", resKey, err)
			return nil, err
		}
		klog.Errorf("resource %s token expired", resKey)
		return nil, fmt.Errorf("resource %s token expired", resKey)
	}
	return &tokenRequest, nil
}

func getTokenRemotely(resource string, tr *authenticationv1.TokenRequest, c *serviceAccountToken) (*authenticationv1.TokenRequest, error) {
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

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == modules.MetaManagerModuleName {
		return handleServiceAccountTokenFromMetaDB(content)
	}
	return handleServiceAccountTokenFromMetaManager(content)
}

func (c *serviceAccountToken) GetServiceAccountToken(namespace string, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	tokenReq, err := getTokenLocally(name, namespace, tr)
	if err != nil {
		resource := fmt.Sprintf("%s/%s/%s", namespace, model.ResourceTypeServiceAccountToken, name)
		return getTokenRemotely(resource, tr, c)
	}
	return tokenReq, nil
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

// ServiceAccountGetter is interface to get Client service account
type ServiceAccountsGetter interface {
	ServiceAccounts(namespace string) ServiceAccountInterface
}

// ServiceAccountInterface is interface for Client service account token
type ServiceAccountInterface interface {
	Get(name string) (*corev1.ServiceAccount, error)
}

type serviceAccount struct {
	namespace string
}

func newServiceAccount(namespace string) *serviceAccount {
	return &serviceAccount{namespace: namespace}
}

func (s *serviceAccount) Get(name string) (*corev1.ServiceAccount, error) {
	rst, err := dao.QueryMeta("type", model.ResourceTypeSaAccess)
	if err != nil {
		return nil, err
	}
	for _, v := range *rst {
		var saAccess policyv1alpha1.ServiceAccountAccess
		err = json.Unmarshal([]byte(v), &saAccess)
		if err != nil {
			klog.Errorf("failed to unmarshal saAccess %v", err)
			return nil, err
		}
		if saAccess.Namespace == s.namespace && saAccess.Spec.ServiceAccount.Name == name {
			rst := saAccess.Spec.ServiceAccount
			rst.UID = saAccess.Spec.ServiceAccountUID
			return &rst, nil
		}
	}
	return nil, fmt.Errorf("serviceaccount %s/%s not found", s.namespace, name)
}

func CheckTokenExist(token string) bool {
	if token == "" {
		return false
	}
	metas, err := dao.QueryMeta("type", model.ResourceTypeServiceAccountToken)
	if err != nil {
		klog.Errorf("query meta %s failed: %v", model.ResourceTypeServiceAccountToken, err)
		return false
	}

	for _, v := range *metas {
		var tokenRequest authenticationv1.TokenRequest
		err = json.Unmarshal([]byte(v), &tokenRequest)
		if err != nil {
			klog.Errorf("unmarshal resource %s token request failed: %v", model.ResourceTypeServiceAccountToken, err)
			return false
		}
		if tokenRequest.Status.Token == token {
			return true
		}
	}
	return false
}
