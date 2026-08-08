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
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
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
	ms := dbclient.NewMetaService()
	svcAccounts, err := ms.QueryAllMeta("type", model.ResourceTypeServiceAccountToken)
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
			err := ms.DeleteMetaByKey(sa.Key)
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

// baseKey is the spec-derived key prefix that identifies the logical token
// (independent of which generation). All generations of the same logical token
// share this prefix. Used for prefix lookup in metaDB.
func baseKey(name, namespace string, tr *authenticationv1.TokenRequest) string {
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

// KeyFunc returns the per-generation storage key for a TokenRequest. The key
// is the base (spec-derived) key suffixed with the token's exp timestamp so
// that successive refreshed tokens for the same logical token are stored as
// separate rows. This is required to keep the previous (still un-expired)
// token's row in metaDB during the propagation window between a refresh and
// the moment the pod re-reads the new token from its projected volume; without
// it, MetaServer (which authenticates by checking metaDB membership) rejects
// requests bearing the previous token.
//
// Callers without a populated Status.ExpirationTimestamp (i.e., callers that
// only have a TokenRequest spec, like an upstream kubelet refresh call) should
// not use KeyFunc — they should use baseKey and a prefix query.
//
// Keys are nonconfidential and safe to log.
func KeyFunc(name, namespace string, tr *authenticationv1.TokenRequest) string {
	base := baseKey(name, namespace, tr)
	if tr.Status.ExpirationTimestamp.IsZero() {
		// Defensive: a write without exp is malformed. Return base so the
		// behavior is at worst the pre-refactor behavior (one row per
		// logical token, overwritten on refresh). The caller should ensure
		// Status.ExpirationTimestamp is populated.
		return base
	}
	return fmt.Sprintf("%s/%d", base, tr.Status.ExpirationTimestamp.UnixNano())
}

func getTokenLocally(name, namespace string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	prefix := baseKey(name, namespace, tr)
	ms := dbclient.NewMetaService()
	metas, err := ms.QueryMetaByKeyPrefix(prefix)
	if err != nil {
		klog.Errorf("query meta by prefix %s failed: %v", prefix, err)
		return nil, err
	}
	if metas == nil || len(*metas) == 0 {
		return nil, fmt.Errorf("no cached token for %s", prefix)
	}

	// Pick the newest un-expired generation. Older generations may still be
	// present (kept intentionally to bridge the refresh-propagation window;
	// the GC sweep removes them after their exp passes), but the freshest
	// one is what we want to return to the caller.
	now := time.Now()
	var newest *authenticationv1.TokenRequest
	for _, v := range *metas {
		var cur authenticationv1.TokenRequest
		if err := json.Unmarshal([]byte(v), &cur); err != nil {
			klog.Errorf("unmarshal cached token under prefix %s failed: %v", prefix, err)
			continue
		}
		if cur.Status.ExpirationTimestamp.IsZero() || !cur.Status.ExpirationTimestamp.Time.After(now) {
			continue
		}
		if newest == nil || cur.Status.ExpirationTimestamp.Time.After(newest.Status.ExpirationTimestamp.Time) {
			snapshot := cur
			newest = &snapshot
		}
	}
	if newest == nil {
		return nil, fmt.Errorf("no un-expired cached token for %s", prefix)
	}

	if requiresRefresh(newest) {
		// The freshest cached generation is past the 80%-of-TTL threshold;
		// trigger a remote refresh. Do NOT delete any cached rows here:
		// MetaServer authenticates pod requests by checking presence of the
		// token in metaDB (see CheckTokenExist). The previous patch removed
		// the eviction of this row; this patch additionally arranges for
		// the refreshed token to be stored under a *new* per-generation key
		// (see KeyFunc), so both rows coexist for a brief window. GC
		// (RunExpiredTokenGC) removes rows whose exp has passed.
		klog.V(4).Infof("resource %s token requires refresh", prefix)
		return nil, fmt.Errorf("resource %s token requires refresh", prefix)
	}
	return newest, nil
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
	rst, err := dbclient.NewMetaService().QueryMeta("type", model.ResourceTypeSaAccess)
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
	metas, err := dbclient.NewMetaService().QueryMeta("type", model.ResourceTypeServiceAccountToken)
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

// RunExpiredTokenGC periodically scans all service-account token rows in
// metaDB and deletes any whose Status.ExpirationTimestamp has passed.
//
// Because refreshed tokens are stored under per-generation keys (see KeyFunc)
// rather than overwriting the previous generation, expired generations
// accumulate until removed here. With a 10-minute token TTL and an 80%
// refresh threshold, steady-state row count per logical token is bounded
// by ~2; the GC interval below is generous.
//
// Stops when stopCh is closed.
func RunExpiredTokenGC(stopCh <-chan struct{}, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			gcExpiredTokensOnce()
		}
	}
}

func gcExpiredTokensOnce() {
	ms := dbclient.NewMetaService()
	rows, err := ms.QueryAllMeta("type", model.ResourceTypeServiceAccountToken)
	if err != nil {
		klog.Errorf("GC: query SA tokens failed: %v", err)
		return
	}
	if rows == nil {
		return
	}
	now := time.Now()
	for _, row := range *rows {
		var tr authenticationv1.TokenRequest
		if err := json.Unmarshal([]byte(row.Value), &tr); err != nil {
			// Don't delete: an unparseable row is a separate bug; leave it
			// for diagnosis rather than silently dropping it.
			klog.Errorf("GC: unmarshal SA token row %s failed: %v", row.Key, err)
			continue
		}
		if tr.Status.ExpirationTimestamp.IsZero() {
			continue
		}
		if tr.Status.ExpirationTimestamp.Time.After(now) {
			continue
		}
		if err := ms.DeleteMetaByKey(row.Key); err != nil {
			klog.Errorf("GC: delete expired SA token row %s failed: %v", row.Key, err)
		}
	}
}
