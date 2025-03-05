package status

// import (
// 	"context"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
// 	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
// 	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
// )

// func TestUpdateImagePrepullJobNodeTaskStatus(t *testing.T) {
// 	// TODO: fix to fake client
// 	ctx := context.TODO()
// 	client.InitKubeEdgeClient(&v1alpha1.KubeAPIConfig{
// 		KubeConfig: "/Users/willardhu/dev/gomods/github.com/kubeedge/kubeedge/_tmp/kant-test-1year-kubeconfig.yaml",
// 	}, false)

// 	var wg sync.WaitGroup
// 	wg.Add(1)

// 	Init(ctx)
// 	// Wait for the goroutine to start
// 	time.Sleep(200 * time.Millisecond)
// 	GetImagePrePullJobStatusUpdater().UpdateStatus(UpdateStatusOptions[operationsv1alpha2.ImagePrePullNodeTaskStatus]{
// 		JobName: "imagepull-01",
// 		NodeTaskStatus: operationsv1alpha2.ImagePrePullNodeTaskStatus{
// 			Action: operationsv1alpha2.ImagePrePullJobActionPull,
// 			BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
// 				NodeName: "ubuntu",
// 				Status:   operationsv1alpha2.NodeTaskStatusFailure,
// 			},
// 			ImageStatus: []operationsv1alpha2.ImageStatus{
// 				{
// 					Image:  "nginx:latest",
// 					State:  operationsv1alpha2.NodeTaskStatusFailure,
// 					Reason: "test error",
// 				},
// 			},
// 		},
// 		Callback: func(err error) {
// 			t.Log("running callback")
// 			if err != nil {
// 				t.Error(err)
// 			}
// 			wg.Done()
// 		},
// 	})

// 	wg.Wait()
// }
