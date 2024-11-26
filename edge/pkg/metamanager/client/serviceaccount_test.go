package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewServiceAccountToken(t *testing.T) {
	assert := assert.New(t)

	sendInterface := newSend()

	sat := newServiceAccountToken(sendInterface)
	assert.NotNil(sat)
	assert.NotNil(sat.send)
	assert.Equal(sendInterface, sat.send)
}

func TestRequiresRefresh(t *testing.T) {
	assert := assert.New(t)

	now := time.Now()
	testcases := []struct {
		name     string
		tr       *authenticationv1.TokenRequest
		expected bool
	}{
		{
			name: "Not expired",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour)),
				},
			},
			expected: false,
		},
		{
			name: "Expired",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
				},
				Status: authenticationv1.TokenRequestStatus{
					ExpirationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
				},
			},
			expected: true,
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			result := requiresRefresh(test.tr)
			assert.Equal(test.expected, result)
		})
	}
}

func TestKeyFunc(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		name      string
		saName    string
		namespace string
		tr        *authenticationv1.TokenRequest
		expected  string
	}{
		{
			name:      "Basic TokenRequest",
			saName:    "test-sa",
			namespace: "default",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"audience1", "audience2"},
					ExpirationSeconds: func() *int64 { i := int64(3600); return &i }(),
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind: "Pod",
						Name: "test-pod",
						UID:  "12345",
					},
				},
			},
			expected: `"test-sa"/"default"/[]string{"audience1", "audience2"}/3600/v1.BoundObjectReference{Kind:"Pod", APIVersion:"", Name:"test-pod", UID:"12345"}`,
		},
		{
			name:      "TokenRequest with nil ExpirationSeconds",
			saName:    "test-sa",
			namespace: "kube-system",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences:         []string{"audience3"},
					ExpirationSeconds: nil,
					BoundObjectRef:    nil,
				},
			},
			expected: `"test-sa"/"kube-system"/[]string{"audience3"}/0/v1.BoundObjectReference{Kind:"", APIVersion:"", Name:"", UID:""}`,
		},
		{
			name:      "TokenRequest with empty fields",
			saName:    "test-sa",
			namespace: "default",
			tr: &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{},
			},
			expected: `"test-sa"/"default"/[]string(nil)/0/v1.BoundObjectReference{Kind:"", APIVersion:"", Name:"", UID:""}`,
		},
	}

	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			result := KeyFunc(test.saName, test.namespace, test.tr)
			assert.Equal(test.expected, result)
		})
	}
}

func TestHandleServiceAccountTokenFromMetaDB(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid TokenRequest JSON in array
	validTokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-token-request",
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{"audience1", "audience2"},
		},
		Status: authenticationv1.TokenRequestStatus{
			Token: "test-token",
		},
	}
	tokenRequestJSON, _ := json.Marshal(validTokenRequest)
	validContent, _ := json.Marshal([]string{string(tokenRequestJSON)})

	tokenRequest, err := handleServiceAccountTokenFromMetaDB(validContent)
	assert.NoError(err)
	assert.NotNil(tokenRequest)
	assert.Equal(validTokenRequest.Name, tokenRequest.Name)
	assert.Equal(validTokenRequest.Spec.Audiences, tokenRequest.Spec.Audiences)
	assert.Equal(validTokenRequest.Status.Token, tokenRequest.Status.Token)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(invalidContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "unmarshal message to serviceaccount list from db failed")

	// Test case 3: Empty array
	emptyContent, _ := json.Marshal([]string{})

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(emptyContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "serviceaccount length from meta db is 0")

	// Test case 4: Array with multiple elements
	multipleContent, _ := json.Marshal([]string{"{}", "{}"})

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(multipleContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "serviceaccount length from meta db is 2")

	// Test case 5: Invalid TokenRequest JSON in the array
	invalidTokenRequestContent, _ := json.Marshal([]string{"{invalid json}"})

	tokenRequest, err = handleServiceAccountTokenFromMetaDB(invalidTokenRequestContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "unmarshal message to serviceaccount token from db failed")
}

func TestHandleServiceAccountTokenFromMetaManager(t *testing.T) {
	assert := assert.New(t)

	// Test case 1: Valid TokenRequest JSON
	validTokenRequest := &authenticationv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-token-request",
		},
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{"audience1", "audience2"},
		},
		Status: authenticationv1.TokenRequestStatus{
			Token: "test-token",
		},
	}

	validContent, _ := json.Marshal(validTokenRequest)

	tokenRequest, err := handleServiceAccountTokenFromMetaManager(validContent)
	assert.NoError(err)
	assert.NotNil(tokenRequest)
	assert.Equal(validTokenRequest.Name, tokenRequest.Name)
	assert.Equal(validTokenRequest.Spec.Audiences, tokenRequest.Spec.Audiences)
	assert.Equal(validTokenRequest.Status.Token, tokenRequest.Status.Token)

	// Test case 2: Invalid JSON
	invalidContent := []byte("invalid json")

	tokenRequest, err = handleServiceAccountTokenFromMetaManager(invalidContent)
	assert.Error(err)
	assert.Nil(tokenRequest)
	assert.Contains(err.Error(), "unmarshal message to service account failed")

	// Test case 3: Empty JSON object
	emptyContent := []byte("{}")

	tokenRequest, err = handleServiceAccountTokenFromMetaManager(emptyContent)
	assert.NoError(err)
	assert.NotNil(tokenRequest)
	assert.Empty(tokenRequest.Name)
	assert.Empty(tokenRequest.Spec.Audiences)
	assert.Empty(tokenRequest.Status.Token)
}

func TestNewServiceAccount(t *testing.T) {
	namespace := "test-namespace"

	sa := newServiceAccount(namespace)

	assert.NotNil(t, sa)
	assert.IsType(t, &serviceAccount{}, sa)
	assert.Equal(t, namespace, sa.namespace)
}
