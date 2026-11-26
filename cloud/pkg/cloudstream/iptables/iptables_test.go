//go:build linux
// +build linux

package iptables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestHandleCloudCorePodDelete(t *testing.T) {
	tests := []struct {
		name         string
		obj          interface{}
		wantEnqueued bool
		wantName     string
	}{
		{
			name: "Delete cloudcore pod",
			obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cloudcore-0",
					Labels: map[string]string{
						constants.SystemName: constants.CloudConfigMapName,
					},
				},
				Status: v1.PodStatus{
					PodIP: "10.0.0.1",
				},
			},
			wantEnqueued: true,
			wantName:     "cloudcore-0",
		},
		{
			name: "Delete tombstone cloudcore pod",
			obj: cache.DeletedFinalStateUnknown{
				Key: "kubeedge/cloudcore-1",
				Obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cloudcore-1",
						Labels: map[string]string{
							constants.SystemName: constants.CloudConfigMapName,
						},
					},
					Status: v1.PodStatus{
						PodIP: "10.0.0.2",
					},
				},
			},
			wantEnqueued: true,
			wantName:     "cloudcore-1",
		},
		{
			name: "Delete invalid tombstone",
			obj: cache.DeletedFinalStateUnknown{
				Key: "kubeedge/cloudcore-2",
				Obj: "not-a-pod",
			},
			wantEnqueued: false,
		},
		{
			name: "Delete pod without cloudcore label",
			obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "other-pod",
				},
				Status: v1.PodStatus{
					PodIP: "10.0.0.3",
				},
			},
			wantEnqueued: false,
		},
		{
			name: "Delete cloudcore pod without pod ip",
			obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cloudcore-2",
					Labels: map[string]string{
						constants.SystemName: constants.CloudConfigMapName,
					},
				},
			},
			wantEnqueued: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var enqueued *v1.Pod
			im := &Manager{
				enqueuePod: func(pod *v1.Pod) {
					enqueued = pod
				},
			}

			im.handleCloudCorePodDelete(tt.obj)

			if !tt.wantEnqueued {
				assert.Nil(t, enqueued)
				return
			}

			if assert.NotNil(t, enqueued) {
				assert.Equal(t, tt.wantName, enqueued.Name)
			}
		})
	}
}
