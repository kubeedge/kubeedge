/*
Copyright 2023 The Kubernetes Authors.

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

// Package clustertrustbundle abstracts access to ClusterTrustBundles so that
// projected volumes can use them.
package clustertrustbundle

import (
	"encoding/pem"
	"fmt"
	"math/rand"
	"time"

	certificatesv1alpha1 "k8s.io/api/certificates/v1alpha1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	lrucache "k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
)

const (
	maxLabelSelectorLength = 100 * 1024
)

// clusterTrustBundle is a type constraint for version-independent ClusterTrustBundle API
type clusterTrustBundle interface {
	certificatesv1alpha1.ClusterTrustBundle | certificatesv1beta1.ClusterTrustBundle
}

// clusterTrustBundlesLister is an API-verion independent ClusterTrustBundles lister
type clusterTrustBundlesLister[T clusterTrustBundle] interface {
	Get(string) (*T, error)
	List(labels.Selector) ([]*T, error)
}

type clusterTrustBundleHandlers[T clusterTrustBundle] interface {
	GetName(*T) string
	GetSignerName(*T) string
	GetTrustBundle(*T) string
}

type alphaClusterTrustBundleHandlers struct{}

type betaClusterTrustBundleHandlers struct{}

func (b *alphaClusterTrustBundleHandlers) GetName(ctb *certificatesv1alpha1.ClusterTrustBundle) string {
	return ctb.Name
}

func (b *alphaClusterTrustBundleHandlers) GetSignerName(ctb *certificatesv1alpha1.ClusterTrustBundle) string {
	return ctb.Spec.SignerName
}

func (b *alphaClusterTrustBundleHandlers) GetTrustBundle(ctb *certificatesv1alpha1.ClusterTrustBundle) string {
	return ctb.Spec.TrustBundle
}

func (b betaClusterTrustBundleHandlers) GetName(ctb *certificatesv1beta1.ClusterTrustBundle) string {
	return ctb.Name
}

func (b *betaClusterTrustBundleHandlers) GetSignerName(ctb *certificatesv1beta1.ClusterTrustBundle) string {
	return ctb.Spec.SignerName
}

func (b *betaClusterTrustBundleHandlers) GetTrustBundle(ctb *certificatesv1beta1.ClusterTrustBundle) string {
	return ctb.Spec.TrustBundle
}

// Manager abstracts over the ability to get trust anchors.
type Manager interface {
	GetTrustAnchorsByName(name string, allowMissing bool) ([]byte, error)
	GetTrustAnchorsBySigner(signerName string, labelSelector *metav1.LabelSelector, allowMissing bool) ([]byte, error)
}

// InformerManager is the "real" manager.  It uses informers to track
// ClusterTrustBundle objects.
type InformerManager[T clusterTrustBundle] struct {
	ctbInformer cache.SharedIndexInformer
	ctbLister   clusterTrustBundlesLister[T]

	ctbHandlers clusterTrustBundleHandlers[T]

	normalizationCache *lrucache.LRUExpireCache
	cacheTTL           time.Duration
}

var _ Manager = (*InformerManager[certificatesv1beta1.ClusterTrustBundle])(nil)

// GetTrustAnchorsByName returns normalized and deduplicated trust anchors from
// a single named ClusterTrustBundle.
func (m *InformerManager[T]) GetTrustAnchorsByName(name string, allowMissing bool) ([]byte, error) {
	if !m.ctbInformer.HasSynced() {
		return nil, fmt.Errorf("ClusterTrustBundle informer has not yet synced")
	}

	cacheKey := cacheKeyType{ctbName: name}

	if cachedAnchors, ok := m.normalizationCache.Get(cacheKey); ok {
		return cachedAnchors.([]byte), nil
	}

	ctb, err := m.ctbLister.Get(name)
	if k8serrors.IsNotFound(err) && allowMissing {
		return []byte{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("while getting ClusterTrustBundle: %w", err)
	}

	pemTrustAnchors, err := m.normalizeTrustAnchors([]*T{ctb})
	if err != nil {
		return nil, fmt.Errorf("while normalizing trust anchors: %w", err)
	}

	m.normalizationCache.Add(cacheKey, pemTrustAnchors, m.cacheTTL)

	return pemTrustAnchors, nil
}

// GetTrustAnchorsBySigner returns normalized and deduplicated trust anchors
// from a set of selected ClusterTrustBundles.
func (m *InformerManager[T]) GetTrustAnchorsBySigner(signerName string, labelSelector *metav1.LabelSelector, allowMissing bool) ([]byte, error) {
	if !m.ctbInformer.HasSynced() {
		return nil, fmt.Errorf("ClusterTrustBundle informer has not yet synced")
	}

	// Note that this function treats nil as "match nothing", and non-nil but
	// empty as "match everything".
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while parsing label selector: %w", err)
	}

	cacheKey := cacheKeyType{signerName: signerName, labelSelector: selector.String()}

	if lsLen := len(cacheKey.labelSelector); lsLen > maxLabelSelectorLength {
		return nil, fmt.Errorf("label selector length (%d) is larger than %d", lsLen, maxLabelSelectorLength)
	}

	if cachedAnchors, ok := m.normalizationCache.Get(cacheKey); ok {
		return cachedAnchors.([]byte), nil
	}

	rawCTBList, err := m.ctbLister.List(selector)
	if err != nil {
		return nil, fmt.Errorf("while listing ClusterTrustBundles matching label selector %v: %w", labelSelector, err)
	}

	ctbList := []*T{}
	for _, ctb := range rawCTBList {
		if m.ctbHandlers.GetSignerName(ctb) == signerName {
			ctbList = append(ctbList, ctb)
		}
	}

	if len(ctbList) == 0 {
		if allowMissing {
			return []byte{}, nil
		}
		return nil, fmt.Errorf("combination of signerName and labelSelector matched zero ClusterTrustBundles")
	}

	pemTrustAnchors, err := m.normalizeTrustAnchors(ctbList)
	if err != nil {
		return nil, fmt.Errorf("while normalizing trust anchors: %w", err)
	}

	m.normalizationCache.Add(cacheKey, pemTrustAnchors, m.cacheTTL)

	return pemTrustAnchors, nil
}

func (m *InformerManager[T]) normalizeTrustAnchors(ctbList []*T) ([]byte, error) {
	// Deduplicate trust anchors from all ClusterTrustBundles.
	trustAnchorSet := sets.Set[string]{}
	for _, ctb := range ctbList {
		rest := []byte(m.ctbHandlers.GetTrustBundle(ctb))
		var b *pem.Block
		for {
			b, rest = pem.Decode(rest)
			if b == nil {
				break
			}
			trustAnchorSet = trustAnchorSet.Insert(string(b.Bytes))
		}
	}

	// Give the list a stable ordering that changes each time Kubelet restarts.
	trustAnchorList := sets.List(trustAnchorSet)
	rand.Shuffle(len(trustAnchorList), func(i, j int) {
		trustAnchorList[i], trustAnchorList[j] = trustAnchorList[j], trustAnchorList[i]
	})

	pemTrustAnchors := []byte{}
	for _, ta := range trustAnchorList {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte(ta),
		}
		pemTrustAnchors = append(pemTrustAnchors, pem.EncodeToMemory(b)...)
	}

	return pemTrustAnchors, nil
}

type cacheKeyType struct {
	ctbName       string
	signerName    string
	labelSelector string
}

// NoopManager always returns an error, for use in static kubelet mode.
type NoopManager struct{}

var _ Manager = (*NoopManager)(nil)

// GetTrustAnchorsByName implements Manager.
func (m *NoopManager) GetTrustAnchorsByName(name string, allowMissing bool) ([]byte, error) {
	return nil, fmt.Errorf("ClusterTrustBundle projection is not supported in static kubelet mode")
}

// GetTrustAnchorsBySigner implements Manager.
func (m *NoopManager) GetTrustAnchorsBySigner(signerName string, labelSelector *metav1.LabelSelector, allowMissing bool) ([]byte, error) {
	return nil, fmt.Errorf("ClusterTrustBundle projection is not supported in static kubelet mode")
}
