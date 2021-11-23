package cmd

import (
	"testing"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestQueryToken(t *testing.T) {
	tests := []struct {
		name        string
		clientSet   kubernetes.Interface
		expectError bool
	}{
		{
			name: "Existing test token",
			clientSet: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      common.TokenSecretName,
					Namespace: constants.SystemNamespace,
				},
				Data: map[string][]byte{
					common.TokenDataName: []byte("test"),
				},
			}),
			expectError: false,
		},
		{
			name:        "Missing namespace",
			clientSet:   fake.NewSimpleClientset(),
			expectError: true,
		},
		{
			name: "Missing token secret",
			clientSet: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "othersecret",
					Namespace: constants.SystemNamespace,
				},
			}),
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := queryToken(test.clientSet, constants.SystemNamespace, common.TokenSecretName)
			t.Logf("token: %s\nerror: %v", token, err)
			if test.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
