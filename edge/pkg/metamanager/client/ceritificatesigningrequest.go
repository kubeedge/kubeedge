/*
Copyright 2024 The KubeEdge Authors.

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

package client

import (
	"encoding/json"
	"fmt"
	"reflect"

	v1 "k8s.io/api/certificates/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// CertificateSigningRequestsGetter to get CertificateSigningRequestInterface
type CertificateSigningRequestsGetter interface {
	CertificateSigningRequests() CertificateSigningRequestInterface
}

// CertificateSigningRequestInterface is interface for client CertificateSigningRequests
type CertificateSigningRequestInterface interface {
	Create(*v1.CertificateSigningRequest) (*v1.CertificateSigningRequest, error)
	Get(name string) (*v1.CertificateSigningRequest, error)
}

type certificateSigningRequests struct {
	send SendInterface
}

// CertificateSigningRequestResp represents CertificateSigningRequest response from API-Server
type CertificateSigningRequestResp struct {
	Object *v1.CertificateSigningRequest
	Err    apierrors.StatusError
}

func newCertificateSigningRequests(s SendInterface) *certificateSigningRequests {
	return &certificateSigningRequests{
		send: s,
	}
}

func (c *certificateSigningRequests) Create(csr *v1.CertificateSigningRequest) (*v1.CertificateSigningRequest, error) {
	resource := fmt.Sprintf("%s/%s/%s", "default", model.ResourceTypeCSR, csr.Name)
	csrMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, csr)
	resp, err := c.send.SendSync(csrMsg)
	if err != nil {
		return nil, fmt.Errorf("create csr failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to csr failed, err: %v", err)
	}

	return handleCertificatesSigningRequestResp(content)
}

func (c *certificateSigningRequests) Get(name string) (*v1.CertificateSigningRequest, error) {
	resource := fmt.Sprintf("%s/%s/%s", "default", model.ResourceTypeCSR, name)
	csrMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.QueryOperation, nil)
	resp, err := c.send.SendSync(csrMsg)
	if err != nil {
		return nil, fmt.Errorf("get csr failed, err: %v", err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to csr failed, err: %v", err)
	}

	if resp.GetOperation() == model.ResponseOperation && resp.GetSource() == modules.MetaManagerModuleName {
		return handleCertificateSigningRequestFromMetaDB(content)
	}
	return handleCertificateSigningRequestFromMetaManager(content)
}

func handleCertificateSigningRequestFromMetaDB(content []byte) (*v1.CertificateSigningRequest, error) {
	var list []string
	err := json.Unmarshal(content, &list)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to csr list from db failed, err: %v", err)
	}
	if len(list) != 1 {
		return nil, fmt.Errorf("csr length from meta db is %d", len(list))
	}

	var csr v1.CertificateSigningRequest
	err = json.Unmarshal([]byte(list[0]), &csr)
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

func handleCertificatesSigningRequestResp(content []byte) (*v1.CertificateSigningRequest, error) {
	var csrResp CertificateSigningRequestResp
	err := json.Unmarshal(content, &csrResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to CertificateSigningRequestResp failed, err: %v", err)
	}

	if reflect.DeepEqual(csrResp.Err, apierrors.StatusError{}) {
		return csrResp.Object, nil
	}
	return csrResp.Object, &csrResp.Err
}
