/*
Copyright 2025 The KubeEdge Authors.
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

package admissioncontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"

	v1 "github.com/kubeedge/api/apis/rules/v1"
	versioned_fake "github.com/kubeedge/api/client/clientset/versioned/fake"
	"github.com/kubeedge/kubeedge/cloud/cmd/admission/app/options"
)

const (
	defaultNamespace = "default"
)

func TestStrPtr(t *testing.T) {
	testStr := "test"
	ptr := strPtr(testStr)
	assert.NotNil(t, ptr, "String pointer should not be nil")
	assert.Equal(t, testStr, *ptr, "Pointer should point to the correct string value")
}

func TestConfigTLS(t *testing.T) {
	tempDir := t.TempDir()
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	certData := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)
	keyData := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)

	err := os.WriteFile(certFile, certData, 0600)
	assert.NoError(t, err, "Failed to write test cert file")

	err = os.WriteFile(keyFile, keyData, 0600)
	assert.NoError(t, err, "Failed to write test key file")

	opt := &options.AdmissionOptions{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	restConfig := &restclient.Config{}

	tlsConfig, err := configTLS(opt, restConfig)
	assert.NoError(t, err, "TLS config should be created successfully")
	assert.NotNil(t, tlsConfig, "TLS config should not be nil")
	assert.Len(t, tlsConfig.Certificates, 1, "TLS config should contain exactly one certificate")

	opt = &options.AdmissionOptions{}
	restConfig = &restclient.Config{}

	tlsConfig, err = configTLS(opt, restConfig)
	assert.Error(t, err, "Should return error when no TLS config data is provided")
	assert.Nil(t, tlsConfig, "TLS config should be nil when error occurs")
}

func TestSimpleToAdmissionResponse(t *testing.T) {
	err := errors.New("test error")
	resp := toAdmissionResponse(err)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Result)
	assert.Equal(t, "test error", resp.Result.Message)
}

func TestRunWithInvalidConfig(t *testing.T) {
	opt := &options.AdmissionOptions{
		Kubeconfig: "/non/existent/path/to/kubeconfig",
	}

	err := Run(opt)

	assert.Error(t, err)
}

func TestBasicServe(t *testing.T) {
	reviewer := func(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Message: "Allowed by test",
			},
		}
	}

	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serve(w, req, reviewer)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
	assert.True(t, resp.Response.Allowed)
	assert.Equal(t, "Allowed by test", resp.Response.Result.Message)
}

func TestServeWrongContentType(t *testing.T) {
	reviewer := func(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	req, err := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte("test")))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/xml")

	w := httptest.NewRecorder()

	serve(w, req, reviewer)

	assert.Equal(t, 0, w.Body.Len())
}

func TestServeInvalidJSON(t *testing.T) {
	reviewer := func(review admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	req, err := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte("invalid json")))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serve(w, req, reviewer)

	assert.NotEqual(t, 0, w.Body.Len())

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.NotNil(t, resp.Response.Result)
	assert.Contains(t, resp.Response.Result.Message, "json")
}

func TestServeRuleDelete(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Delete,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/rules", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveRule(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
	assert.True(t, resp.Response.Allowed)
}

func TestServeDeviceDelete(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Delete,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/devices", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveDevice(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
	assert.True(t, resp.Response.Allowed)
}

func TestServeDeviceModelDelete(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Delete,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/devicemodels", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveDeviceModel(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
	assert.True(t, resp.Response.Allowed)
}

func TestServeRuleEndpointDelete(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Delete,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/ruleendpoints", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveRuleEndpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
	assert.True(t, resp.Response.Allowed)
}

func TestServeNodeUpgradeJobDelete(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Delete,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/nodeupgradejobs", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveNodeUpgradeJob(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
	assert.True(t, resp.Response.Allowed)
}

func TestServeMutatingNodeUpgradeJob(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Create,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/mutating/nodeupgradejobs", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveMutatingNodeUpgradeJob(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
}

func TestServeOfflineMigration(t *testing.T) {
	ar := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: admissionv1.Create,
		},
	}

	body, err := json.Marshal(ar)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/offlinemigration", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	serveOfflineMigration(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp.Response)
	assert.Equal(t, ar.Request.UID, resp.Response.UID)
}

func TestRegisterValidateWebhookAdmission(t *testing.T) {
	client := fake.NewSimpleClientset()

	webhookConfig := admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "test.webhook.io",
			},
		},
	}

	err := registerValidateWebhook(client.AdmissionregistrationV1().ValidatingWebhookConfigurations(),
		[]admissionregistrationv1.ValidatingWebhookConfiguration{webhookConfig})
	assert.NoError(t, err)

	_, err = client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
		context.TODO(), "test-webhook", metav1.GetOptions{})
	assert.NoError(t, err)
}

func TestRegisterMutatingWebhookAdmission(t *testing.T) {
	client := fake.NewSimpleClientset()

	webhookConfig := admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "test.webhook.io",
			},
		},
	}

	err := registerMutatingWebhook(client.AdmissionregistrationV1().MutatingWebhookConfigurations(),
		[]admissionregistrationv1.MutatingWebhookConfiguration{webhookConfig})
	assert.NoError(t, err)

	_, err = client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
		context.TODO(), "test-webhook", metav1.GetOptions{})
	assert.NoError(t, err)
}

type TestAdmissionController struct {
	fakeClient    *fake.Clientset
	fakeCRDClient *versioned_fake.Clientset
}

func setupTestController() *TestAdmissionController {
	return &TestAdmissionController{
		fakeClient:    fake.NewSimpleClientset(),
		fakeCRDClient: versioned_fake.NewSimpleClientset(),
	}
}

func TestCRDClientMethods(t *testing.T) {
	tc := setupTestController()

	ruleEndpoint := &v1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint",
			Namespace: defaultNamespace,
		},
	}

	_, err := tc.fakeCRDClient.RulesV1().RuleEndpoints(defaultNamespace).Create(
		context.TODO(), ruleEndpoint, metav1.CreateOptions{})
	assert.NoError(t, err)

	rule := &v1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rule",
			Namespace: defaultNamespace,
		},
	}

	_, err = tc.fakeCRDClient.RulesV1().Rules(defaultNamespace).Create(
		context.TODO(), rule, metav1.CreateOptions{})
	assert.NoError(t, err)

	endpoint, err := tc.fakeCRDClient.RulesV1().RuleEndpoints(defaultNamespace).Get(
		context.TODO(), "test-endpoint", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "test-endpoint", endpoint.Name)

	endpoints, err := tc.fakeCRDClient.RulesV1().RuleEndpoints(defaultNamespace).List(
		context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(endpoints.Items))

	rules, err := tc.fakeCRDClient.RulesV1().Rules(defaultNamespace).List(
		context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rules.Items))
}

func TestRegisterWebhooks(t *testing.T) {
	ac := &AdmissionController{
		Client:    fake.NewSimpleClientset(),
		CrdClient: versioned_fake.NewSimpleClientset(),
	}

	opt := &options.AdmissionOptions{
		AdmissionServiceNamespace: defaultNamespace,
		AdmissionServiceName:      "admission-service",
		Port:                      443,
	}

	caBundle := []byte("test-ca-bundle")

	err := ac.registerWebhooks(opt, caBundle)
	assert.NoError(t, err, "registerWebhooks should not return an error")

	validateWebhook, err := ac.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
		context.TODO(), ValidateCRDWebhookConfigName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, validateWebhook)
	assert.Equal(t, ValidateCRDWebhookConfigName, validateWebhook.Name)

	mutateWebhook, err := ac.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
		context.TODO(), OfflineMigrationConfigName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, mutateWebhook)
	assert.Equal(t, OfflineMigrationConfigName, mutateWebhook.Name)

	mutateWebhook2, err := ac.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
		context.TODO(), MutatingAdmissionWebhookName, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, mutateWebhook2)
	assert.Equal(t, MutatingAdmissionWebhookName, mutateWebhook2.Name)

	assert.Equal(t, 5, len(validateWebhook.Webhooks))
	assert.Equal(t, ValidateDeviceWebhookName, validateWebhook.Webhooks[0].Name)
	assert.Equal(t, ValidateDeviceModelWebhookName, validateWebhook.Webhooks[1].Name)
	assert.Equal(t, ValidateRuleWebhookName, validateWebhook.Webhooks[2].Name)
	assert.Equal(t, ValidateRuleEndpointWebhookName, validateWebhook.Webhooks[3].Name)
	assert.Equal(t, ValidateNodeUpgradeWebhookName, validateWebhook.Webhooks[4].Name)

	assert.Equal(t, 1, len(mutateWebhook.Webhooks))
	assert.Equal(t, OfflineMigrationWebhookName, mutateWebhook.Webhooks[0].Name)

	assert.Equal(t, 1, len(mutateWebhook2.Webhooks))
	assert.Equal(t, MutatingNodeUpgradeWebhookName, mutateWebhook2.Webhooks[0].Name)
}

func TestGetRuleEndpoint(t *testing.T) {
	ac := &AdmissionController{
		CrdClient: versioned_fake.NewSimpleClientset(),
	}

	name := "test-endpoint"

	ruleEndpoint := &v1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: defaultNamespace,
		},
	}
	_, err := ac.CrdClient.RulesV1().RuleEndpoints(defaultNamespace).Create(context.TODO(), ruleEndpoint, metav1.CreateOptions{})
	assert.NoError(t, err)

	result, err := ac.getRuleEndpoint(defaultNamespace, name)
	assert.NoError(t, err)
	assert.Equal(t, name, result.Name)

	result, err = ac.getRuleEndpoint(defaultNamespace, "nonexistent-endpoint")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestListRule(t *testing.T) {
	ac := &AdmissionController{
		CrdClient: versioned_fake.NewSimpleClientset(),
	}

	rule1 := &v1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rule1",
			Namespace: defaultNamespace,
		},
	}
	rule2 := &v1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rule2",
			Namespace: defaultNamespace,
		},
	}

	_, err := ac.CrdClient.RulesV1().Rules(defaultNamespace).Create(
		context.TODO(), rule1, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = ac.CrdClient.RulesV1().Rules(defaultNamespace).Create(
		context.TODO(), rule2, metav1.CreateOptions{})
	assert.NoError(t, err)

	result, err := ac.listRule(defaultNamespace)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListRuleEndpoint(t *testing.T) {
	ac := &AdmissionController{
		CrdClient: versioned_fake.NewSimpleClientset(),
	}

	endpoint1 := &v1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "endpoint1",
			Namespace: defaultNamespace,
		},
	}
	endpoint2 := &v1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "endpoint2",
			Namespace: defaultNamespace,
		},
	}

	_, err := ac.CrdClient.RulesV1().RuleEndpoints(defaultNamespace).Create(
		context.TODO(), endpoint1, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = ac.CrdClient.RulesV1().RuleEndpoints(defaultNamespace).Create(
		context.TODO(), endpoint2, metav1.CreateOptions{})
	assert.NoError(t, err)

	result, err := ac.listRuleEndpoint(defaultNamespace)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestRunWithFakeClients(t *testing.T) {
	opt := &options.AdmissionOptions{
		Kubeconfig:                "/non/existent/path/to/kubeconfig",
		CaCertFile:                "/non/existent/path/to/cacert",
		CertFile:                  "/non/existent/path/to/cert",
		KeyFile:                   "/non/existent/path/to/key",
		AdmissionServiceName:      "test-service",
		AdmissionServiceNamespace: defaultNamespace,
	}

	err := Run(opt)
	assert.Error(t, err)

	opt.Kubeconfig, opt.CaCertFile, opt.CertFile, opt.KeyFile = "", "", "", ""

	err = Run(opt)
	assert.Error(t, err)
}
