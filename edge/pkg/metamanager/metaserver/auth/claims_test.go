package auth

import (
	"context"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2/jwt"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

func init() {
	now = func() time.Time {
		// epoch time: 1514764800
		return time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
}

type deletionTestCase struct {
	name      string
	time      *metav1.Time
	expectErr bool
}

type claimTestCase struct {
	name      string
	getter    serviceaccount.ServiceAccountTokenGetter
	private   *privateClaims
	expiry    jwt.NumericDate
	notBefore jwt.NumericDate
	expectErr string
}

func TestValidatePrivateClaims(t *testing.T) {
	var (
		nowUnix = int64(1514764800)

		serviceAccount = &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "saname", Namespace: "ns", UID: "sauid"}}
		secret         = &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secretname", Namespace: "ns", UID: "secretuid"}}
		pod            = &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "podname", Namespace: "ns", UID: "poduid"}}
	)

	deletionTestCases := []deletionTestCase{
		{
			name: "valid",
			time: nil,
		},
		{
			name: "deleted now",
			time: &metav1.Time{Time: time.Unix(nowUnix, 0)},
		},
		{
			name: "deleted near past",
			time: &metav1.Time{Time: time.Unix(nowUnix-1, 0)},
		},
		{
			name: "deleted near future",
			time: &metav1.Time{Time: time.Unix(nowUnix+1, 0)},
		},
		{
			name: "deleted now-leeway",
			time: &metav1.Time{Time: time.Unix(nowUnix-60, 0)},
		},
		{
			name:      "deleted now-leeway-1",
			time:      &metav1.Time{Time: time.Unix(nowUnix-61, 0)},
			expectErr: true,
		},
	}

	testcases := []claimTestCase{
		{
			name:      "good",
			getter:    fakeGetter{serviceAccount, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Namespace: "ns"}},
			expectErr: "",
		},
		{
			name:      "expired",
			getter:    fakeGetter{serviceAccount, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Namespace: "ns"}},
			expiry:    *jwt.NewNumericDate(now().Add(-1_000 * time.Hour)),
			expectErr: "service account token has expired",
		},
		{
			name:      "not yet valid",
			getter:    fakeGetter{serviceAccount, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Namespace: "ns"}},
			notBefore: *jwt.NewNumericDate(now().Add(1_000 * time.Hour)),
			expectErr: "service account token is not valid yet",
		},
		{
			name:      "missing serviceaccount",
			getter:    fakeGetter{nil, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Namespace: "ns"}},
			expectErr: `serviceaccounts "saname" not found`,
		},
		{
			name:      "missing secret",
			getter:    fakeGetter{serviceAccount, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Secret: &ref{Name: "secretname", UID: "secretuid"}, Namespace: "ns"}},
			expectErr: "service account token has been invalidated",
		},
		{
			name:      "missing pod",
			getter:    fakeGetter{serviceAccount, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Pod: &ref{Name: "podname", UID: "poduid"}, Namespace: "ns"}},
			expectErr: "service account token has been invalidated",
		},
		{
			name:      "different uid serviceaccount",
			getter:    fakeGetter{serviceAccount, nil, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauidold"}, Namespace: "ns"}},
			expectErr: "service account UID (sauid) does not match claim (sauidold)",
		},
		{
			name:      "different uid secret",
			getter:    fakeGetter{serviceAccount, secret, nil, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Secret: &ref{Name: "secretname", UID: "secretuidold"}, Namespace: "ns"}},
			expectErr: "secret UID (secretuid) does not match service account secret ref claim (secretuidold)",
		},
		{
			name:      "different uid pod",
			getter:    fakeGetter{serviceAccount, nil, pod, nil},
			private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Pod: &ref{Name: "podname", UID: "poduidold"}, Namespace: "ns"}},
			expectErr: "pod UID (poduid) does not match service account pod ref claim (poduidold)",
		},
	}

	for _, deletionTestCase := range deletionTestCases {
		var (
			deletedServiceAccount = serviceAccount.DeepCopy()
			deletedPod            = pod.DeepCopy()
			deletedSecret         = secret.DeepCopy()
		)
		deletedServiceAccount.DeletionTimestamp = deletionTestCase.time
		deletedPod.DeletionTimestamp = deletionTestCase.time
		deletedSecret.DeletionTimestamp = deletionTestCase.time

		var saDeletedErr, deletedErr string
		if deletionTestCase.expectErr {
			saDeletedErr = "service account ns/saname has been deleted"
			deletedErr = "service account token has been invalidated"
		}

		testcases = append(testcases,
			claimTestCase{
				name:      deletionTestCase.name + " serviceaccount",
				getter:    fakeGetter{deletedServiceAccount, nil, nil, nil},
				private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Namespace: "ns"}},
				expectErr: saDeletedErr,
			},
			claimTestCase{
				name:      deletionTestCase.name + " secret",
				getter:    fakeGetter{serviceAccount, deletedSecret, nil, nil},
				private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Secret: &ref{Name: "secretname", UID: "secretuid"}, Namespace: "ns"}},
				expectErr: deletedErr,
			},
			claimTestCase{
				name:      deletionTestCase.name + " pod",
				getter:    fakeGetter{serviceAccount, nil, deletedPod, nil},
				private:   &privateClaims{Kubernetes: kubernetes{Svcacct: ref{Name: "saname", UID: "sauid"}, Pod: &ref{Name: "podname", UID: "poduid"}, Namespace: "ns"}},
				expectErr: deletedErr,
			},
		)
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			v := &validator{tc.getter}
			expiry := jwt.NumericDate(nowUnix)
			tcExpiry := tc.expiry
			if tcExpiry != 0 {
				expiry = tcExpiry
			}
			_, err := v.Validate(context.Background(), "", &jwt.Claims{Expiry: &expiry, NotBefore: &tc.notBefore}, tc.private)
			if len(tc.expectErr) > 0 {
				if errStr := errString(err); tc.expectErr != errStr {
					t.Fatalf("expected error %q but got %q", tc.expectErr, errStr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

type fakeGetter struct {
	serviceAccount *v1.ServiceAccount
	secret         *v1.Secret
	pod            *v1.Pod
	node           *v1.Node
}

func (f fakeGetter) GetServiceAccount(_, name string) (*v1.ServiceAccount, error) {
	if f.serviceAccount == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "serviceaccounts"}, name)
	}
	return f.serviceAccount, nil
}
func (f fakeGetter) GetPod(_, name string) (*v1.Pod, error) {
	if f.pod == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "pods"}, name)
	}
	return f.pod, nil
}
func (f fakeGetter) GetSecret(_, name string) (*v1.Secret, error) {
	if f.secret == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "secrets"}, name)
	}
	return f.secret, nil
}
func (f fakeGetter) GetNode(name string) (*v1.Node, error) {
	if f.node == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "node"}, name)
	}
	return f.node, nil
}
