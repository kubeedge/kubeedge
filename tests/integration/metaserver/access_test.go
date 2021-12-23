package metaserver

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestGet(t *testing.T) {
	kubeclient, err := kubernetes.NewForConfig(&rest.Config{
		Host: constants.DefaultMetaServerAddr,
	})
	if err != nil {
		t.Fatalf("failed to new a kubeclient for testing")
	}
	ep, err := kubeclient.CoreV1().Endpoints("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get, %v", err)
	}
	klog.Infof("get ep:%v", ep)
}

func TestService(t *testing.T) {
	kubeclient, err := kubernetes.NewForConfig(&rest.Config{
		Host: constants.DefaultMetaServerAddr,
	})
	if err != nil {
		t.Fatalf("failed to get kubeclient, %v", err)
	}
	var (
		testPort int32 = 8888
		testNS         = "default"
		testName       = "metaserver-test-svc"
	)
	svc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: testName,
					Port: testPort,
				},
			},
		},
	}
	var cleanFn = func() {
		err = kubeclient.CoreV1().Services(testNS).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("failed to clean after test, %v", err)
		}
	}
	defer cleanFn()
	_, err = kubeclient.CoreV1().Services(testNS).Create(context.TODO(), &svc, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("failed to create, %v", err)
	}
	_, err = kubeclient.CoreV1().Services(testNS).Get(context.TODO(), svc.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get, %v", err)
	}
}
