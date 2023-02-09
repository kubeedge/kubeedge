package client

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/api/certificates/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// CertificateSigningRequestsGetter to get CertificateSigningRequests interface
type CertificateSigningRequestsGetter interface {
	CertificateSigningRequests(namespace string) CertificateSigningRequestInterface
}

// CertificateSigningRequestInterface is interface for client CertificateSigningRequests
type CertificateSigningRequestInterface interface {
	Create(*v1.CertificateSigningRequest) (*v1.CertificateSigningRequest, error)
	Get(name string) (*v1.CertificateSigningRequest, error)
}

type certificateSigningRequests struct {
	namespace string
	send      SendInterface
}

// CertificateSigningRequestResp represents CertificateSigningRequest response from the api-server
type CertificateSigningRequestResp struct {
	Object *v1.CertificateSigningRequest
	Err    apierrors.StatusError
}

func newCertificateSigningRequests(namespace string, s SendInterface) *certificateSigningRequests {
	return &certificateSigningRequests{
		send:      s,
		namespace: namespace,
	}
}

func (c *certificateSigningRequests) Create(csr *v1.CertificateSigningRequest) (*v1.CertificateSigningRequest, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeCSR, csr.Name)
	csrMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, csr)
	resp, err := c.send.SendSync(csrMsg)
	if err != nil {
		return nil, fmt.Errorf("create csr failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to csr failed, err: %v", err)
	}

	return handleCertificateSigningRequestResp(content)
}

func (c *certificateSigningRequests) Get(name string) (*v1.CertificateSigningRequest, error) {
	resource := fmt.Sprintf("%s/%s/%s", c.namespace, model.ResourceTypeCSR, name)
	csrMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	msg, err := c.send.SendSync(csrMsg)
	if err != nil {
		return nil, fmt.Errorf("get csr failed, err: %v", err)
	}

	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to csr failed, err: %v", err)
	}

	if msg.GetOperation() == model.ResponseOperation && msg.GetSource() == modules.MetaManagerModuleName {
		return handleCertificateSigningRequestFromMetaDB(content)
	}
	return handleCertificateSigningRequestFromMetaManager(content)
}

func handleCertificateSigningRequestFromMetaDB(content []byte) (*v1.CertificateSigningRequest, error) {
	var lists []string
	err := json.Unmarshal(content, &lists)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to node list from db failed, err: %v", err)
	}

	if len(lists) != 1 {
		return nil, fmt.Errorf("csr length from meta db is %d", len(lists))
	}

	var csr v1.CertificateSigningRequest
	err = json.Unmarshal([]byte(lists[0]), &csr)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csr from db failed, err: %v", err)
	}
	return &csr, nil
}

func handleCertificateSigningRequestFromMetaManager(content []byte) (*v1.CertificateSigningRequest, error) {
	var csr v1.CertificateSigningRequest
	err := json.Unmarshal(content, &csr)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csr failed, err: %v", err)
	}
	return &csr, nil
}

func handleCertificateSigningRequestResp(content []byte) (*v1.CertificateSigningRequest, error) {
	var csrResp CertificateSigningRequestResp
	err := json.Unmarshal(content, &csrResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to CertificateSigningRequest failed, err: %v", err)
	}

	if reflect.DeepEqual(csrResp.Err, apierrors.StatusError{}) {
		return csrResp.Object, nil
	}
	return csrResp.Object, &csrResp.Err
}
