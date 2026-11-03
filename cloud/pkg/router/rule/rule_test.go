package rule

import (
	"sync/atomic"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	routerv1 "github.com/kubeedge/api/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
)

const testEndpointType routerv1.RuleEndpointTypeDef = "faketest"

type fakeEndpoint struct {
	registered   int32
	unregistered int32
}

func (f *fakeEndpoint) Name() string { return string(testEndpointType) }

func (f *fakeEndpoint) RegisterListener(listener.Handle) error {
	atomic.AddInt32(&f.registered, 1)
	return nil
}

func (f *fakeEndpoint) UnregisterListener() {
	atomic.AddInt32(&f.unregistered, 1)
}

func (f *fakeEndpoint) Forward(provider.Target, interface{}) (interface{}, error) {
	return nil, nil
}

func (f *fakeEndpoint) GoToTarget(map[string]interface{}, chan struct{}) (interface{}, error) {
	return nil, nil
}

type fakeFactory struct {
	endpoint *fakeEndpoint
}

func (f *fakeFactory) Type() routerv1.RuleEndpointTypeDef { return testEndpointType }

func (f *fakeFactory) GetSource(*routerv1.RuleEndpoint, map[string]string) provider.Source {
	return f.endpoint
}

func (f *fakeFactory) GetTarget(*routerv1.RuleEndpoint, map[string]string) provider.Target {
	return f.endpoint
}

func setupRule(t *testing.T, name string) *fakeEndpoint {
	t.Helper()

	endpoint := &fakeEndpoint{}
	factory := &fakeFactory{endpoint: endpoint}
	provider.RegisterSource(factory)
	provider.RegisterTarget(factory)

	addRuleEndpoint(&routerv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "src"},
		Spec:       routerv1.RuleEndpointSpec{RuleEndpointType: testEndpointType},
	})
	addRuleEndpoint(&routerv1.RuleEndpoint{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "dst"},
		Spec:       routerv1.RuleEndpointSpec{RuleEndpointType: testEndpointType},
	})
	t.Cleanup(func() {
		deleteRuleEndpoint("default", "src")
		deleteRuleEndpoint("default", "dst")
		rules.Delete(getKey("default", name))
	})

	rule := &routerv1.Rule{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name},
		Spec: routerv1.RuleSpec{
			Source:         "src",
			SourceResource: map[string]string{"path": "/api"},
			Target:         "dst",
			TargetResource: map[string]string{"resource": "http://127.0.0.1:1/edge"},
		},
	}
	if err := addRule(rule); err != nil {
		t.Fatalf("addRule returned err: %v", err)
	}
	if got := atomic.LoadInt32(&endpoint.registered); got != 1 {
		t.Fatalf("RegisterListener called %d times, want 1", got)
	}
	return endpoint
}

func TestDelRuleUnregistersListenerWhenEndpointIsStillCached(t *testing.T) {
	endpoint := setupRule(t, "cached")

	delRule("default", "cached")

	if got := atomic.LoadInt32(&endpoint.unregistered); got != 1 {
		t.Fatalf("UnregisterListener called %d times, want 1", got)
	}
	if _, exist := rules.Load(getKey("default", "cached")); exist {
		t.Fatal("rule was left in the cache")
	}
}

func TestDelRuleUnregistersListenerWhenEndpointIsAlreadyDeleted(t *testing.T) {
	endpoint := setupRule(t, "endpointgone")

	deleteRuleEndpoint("default", "src")
	delRule("default", "endpointgone")

	if got := atomic.LoadInt32(&endpoint.unregistered); got != 1 {
		t.Fatalf("UnregisterListener called %d times, want 1", got)
	}
	if _, exist := rules.Load(getKey("default", "endpointgone")); exist {
		t.Fatal("rule was left in the cache")
	}
}

func TestDelRuleIgnoresUnknownRule(t *testing.T) {
	delRule("default", "missing")

	if _, exist := rules.Load(getKey("default", "missing")); exist {
		t.Fatal("unknown rule was stored")
	}
}
