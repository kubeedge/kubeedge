package types

import (
	"crypto/x509"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

// PodStatusRequest is Message.Content which come from edge
type PodStatusRequest struct {
	UID    types.UID
	Name   string
	Status v1.PodStatus
}

//ExtendResource is extended resource details that come from edge
type ExtendResource struct {
	Name     string            `json:"name,omitempty"`
	Type     string            `json:"type,omitempty"`
	Capacity resource.Quantity `json:"capacity,omitempty"`
}

// NodeStatusRequest is Message.Content which come from edge
type NodeStatusRequest struct {
	UID             types.UID
	Status          v1.NodeStatus
	ExtendResources map[v1.ResourceName][]ExtendResource
}

// CertificateSigningRequest is message which use for certificate request from cloudcore
type CertificateSigningRequest struct {
	Request []byte             `json:"request"`
	Usages  []x509.ExtKeyUsage `json:"usages"`
}
