/*
Copyright 2021 The KubeEdge Authors.

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

package manager

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
)

func TestSecretManager_Events(t *testing.T) {
	type fields struct {
		events chan watch.Event
	}
	ch := make(chan watch.Event, 1)
	tests := []struct {
		name   string
		fields fields
		want   chan watch.Event
	}{
		{
			"TestSecretManager_Events(): Case 1",
			fields{
				events: ch,
			},
			ch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &SecretManager{
				events: tt.fields.events,
			}
			if got := sm.Events(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SecretManager.Events() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSecretManager(t *testing.T) {
	type args struct {
		informer cache.SharedIndexInformer
	}

	config := &v1alpha1.EdgeController{
		Buffer: &v1alpha1.EdgeControllerBuffer{
			ConfigMapEvent: 1024,
		},
	}

	tmpfile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tmpfile.Name())
	if err := os.WriteFile(tmpfile.Name(), []byte(mockKubeConfigContent), 0666); err != nil {
		t.Error(err)
	}
	client.InitKubeEdgeClient(&v1alpha1.KubeAPIConfig{
		KubeConfig:  tmpfile.Name(),
		QPS:         100,
		Burst:       200,
		ContentType: "application/vnd.kubernetes.protobuf",
	}, false)

	client.DefaultGetRestMapper = func() (mapper meta.RESTMapper, err error) { return nil, nil }

	tests := []struct {
		name string
		args args
	}{
		{
			"TestNewSecretManager(): Case 1",
			args{
				informers.GetInformersManager().GetKubeInformerFactory().Core().V1().Secrets().Informer(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err = NewSecretManager(config, tt.args.informer)
			assert.NoError(t, err)
		})
	}
}
