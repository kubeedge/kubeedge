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

package kubeclientbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	fakekube "k8s.io/client-go/kubernetes/fake"

	kecoordinationv1 "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/typed/coordination/v1"
	kecorev1 "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/typed/core/v1"
	kestoragev1 "github.com/kubeedge/kubeedge/edge/pkg/edged/kubeclientbridge/typed/storage/v1"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

func TestNewSimpleClientset(t *testing.T) {
	assert := assert.New(t)

	metaClient := client.New()
	clientset := NewSimpleClientset(metaClient)

	assert.NotNil(clientset)
	assert.IsType(&Clientset{}, clientset)
	assert.Equal(metaClient, clientset.(*Clientset).MetaClient)
	assert.IsType(&fakekube.Clientset{}, &clientset.(*Clientset).Clientset)
}

func TestCoreV1(t *testing.T) {
	assert := assert.New(t)

	metaClient := client.New()
	clientset := NewSimpleClientset(metaClient).(*Clientset)

	coreV1 := clientset.CoreV1()

	assert.IsType(&kecorev1.CoreV1Bridge{}, coreV1)
	coreV1Bridge := coreV1.(*kecorev1.CoreV1Bridge)
	assert.Equal(metaClient, coreV1Bridge.MetaClient)
	assert.Equal(&clientset.Fake, coreV1Bridge.Fake)
}

func TestStorageV1(t *testing.T) {
	assert := assert.New(t)

	metaClient := client.New()
	clientset := NewSimpleClientset(metaClient).(*Clientset)

	storageV1 := clientset.StorageV1()

	assert.IsType(&kestoragev1.StorageV1Bridge{}, storageV1)
	storageV1Bridge := storageV1.(*kestoragev1.StorageV1Bridge)
	assert.Equal(metaClient, storageV1Bridge.MetaClient)
	assert.Equal(&clientset.Fake, storageV1Bridge.Fake)
}

func TestCoordinationV1(t *testing.T) {
	assert := assert.New(t)

	metaClient := client.New()
	clientset := NewSimpleClientset(metaClient).(*Clientset)

	coordinationV1 := clientset.CoordinationV1()

	assert.IsType(&kecoordinationv1.CoordinationV1Bridge{}, coordinationV1)
	coordinationV1Bridge := coordinationV1.(*kecoordinationv1.CoordinationV1Bridge)
	assert.Equal(metaClient, coordinationV1Bridge.MetaClient)
	assert.Equal(&clientset.Fake, coordinationV1Bridge.Fake)
}
