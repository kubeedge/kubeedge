/*
Copyright 2019 The KubeEdge Authors.

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

package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/util"
)

func TestNegotiateTunnelPort(t *testing.T) {
	type testCase struct {
		isConfigExits bool
		isPortExist   bool
		isPortUsed    bool
	}

	tests := []struct {
		name    string
		cases   testCase
		want    int
		wantErr bool
	}{
		{
			name:    "config not exits",
			want:    10351,
			wantErr: false,
		},
		{
			name:    "port record exits",
			cases:   testCase{isConfigExits: true, isPortExist: true},
			want:    10351,
			wantErr: false,
		},
		{
			name:    "port used",
			cases:   testCase{isConfigExits: true, isPortUsed: true},
			want:    10352,
			wantErr: false,
		},
	}

	hostnameOverride := util.GetHostname()
	localIP, _ := util.GetLocalIP(hostnameOverride)

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(client.CreateNamespaceIfNeeded, func(ctx context.Context, namespace string) error {
				return nil
			})

			patches.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
				if tt.cases.isConfigExits {
					record := "{}"
					if tt.cases.isPortExist {
						record = "{\"ipTunnelPort\":{\"" + localIP + "\":10351},\"port\":{\"10351\":true}}"
					} else if tt.cases.isPortUsed {
						record = "{\"ipTunnelPort\":{\"127.0.0.1\":10351},\"port\":{\"10351\":true}}"
					}
					cm := v1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      modules.TunnelPort,
							Namespace: constants.SystemNamespace,
							Annotations: map[string]string{
								modules.TunnelPortRecordAnnotationKey: record,
							},
						},
					}
					return fakekube.NewSimpleClientset(&cm)
				}
				return fakekube.NewSimpleClientset()
			})

			got, err := NegotiateTunnelPort()
			if (err != nil) != tt.wantErr {
				t.Errorf("NegotiateTunnelPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("NegotiateTunnelPort() got = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestNegotiateTunnelPort_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		mockNS       func() error
		setupClient  func() kubernetes.Interface
		expectErr    bool
		expectErrStr string
	}{
		{
			name: "CreateNamespaceIfNeeded fails",
			mockNS: func() error {
				return errors.New("simulated namespace error")
			},
			setupClient:  func() kubernetes.Interface { return fakekube.NewSimpleClientset() },
			expectErr:    true,
			expectErrStr: "failed to create system namespace: simulated namespace error",
		},
		{
			name:   "ConfigMap missing annotation",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				return fakekube.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      modules.TunnelPort,
						Namespace: constants.SystemNamespace,
					},
				})
			},
			expectErr:    true,
			expectErrStr: "failed to get tunnel port record",
		},
		{
			name:   "JSON unmarshal error",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				return fakekube.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      modules.TunnelPort,
						Namespace: constants.SystemNamespace,
						Annotations: map[string]string{
							modules.TunnelPortRecordAnnotationKey: "invalid-json-format",
						},
					},
				})
			},
			expectErr: true,
		},
		{
			name:   "JSON unmarshal empty record {}",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				return fakekube.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      modules.TunnelPort,
						Namespace: constants.SystemNamespace,
						Annotations: map[string]string{
							modules.TunnelPortRecordAnnotationKey: "{}",
						},
					},
				})
			},
			expectErr: false,
		},
		{
			name:   "JSON unmarshal missing ipTunnelPort map",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				return fakekube.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      modules.TunnelPort,
						Namespace: constants.SystemNamespace,
						Annotations: map[string]string{
							modules.TunnelPortRecordAnnotationKey: "{\"port\":{}}",
						},
					},
				})
			},
			expectErr: false,
		},
		{
			name:   "JSON unmarshal missing port map",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				return fakekube.NewSimpleClientset(&v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      modules.TunnelPort,
						Namespace: constants.SystemNamespace,
						Annotations: map[string]string{
							modules.TunnelPortRecordAnnotationKey: "{\"ipTunnelPort\":{}}",
						},
					},
				})
			},
			expectErr: false,
		},
		{
			name:   "ConfigMap Update fails",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				cm := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      modules.TunnelPort,
						Namespace: constants.SystemNamespace,
						Annotations: map[string]string{
							modules.TunnelPortRecordAnnotationKey: "{\"ipTunnelPort\":{},\"port\":{}}",
						},
					},
				}
				cs := fakekube.NewSimpleClientset(cm)
				cs.PrependReactor("update", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("simulated update error")
				})
				return cs
			},
			expectErr: true,
		},
		{
			name:   "ConfigMap Create fails (Not Found branch)",
			mockNS: func() error { return nil },
			setupClient: func() kubernetes.Interface {
				cs := fakekube.NewSimpleClientset() // Empty client causes IsNotFound
				cs.PrependReactor("create", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("simulated create error")
				})
				return cs
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(client.CreateNamespaceIfNeeded, func(ctx context.Context, namespace string) error {
				return tt.mockNS()
			})
			patches.ApplyFunc(client.GetKubeClient, tt.setupClient)

			port, err := NegotiateTunnelPort()
			if tt.expectErr {
				assert.Error(t, err)
				if tt.expectErrStr != "" {
					assert.EqualError(t, err, tt.expectErrStr)
				}
				assert.Nil(t, port)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, port)
			}
		})
	}
}

func TestNegotiatePort(t *testing.T) {
	record := map[int]bool{
		constants.ServerPort + 1: true,
		constants.ServerPort + 2: true,
	}
	port := negotiatePort(record)
	assert.Equal(t, constants.ServerPort+3, port)
}

func TestUpdateCloudCoreConfigMap(t *testing.T) {
	t.Run("Successfully updates config map", func(t *testing.T) {
		fakeClient := fakekube.NewSimpleClientset(&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.CloudConfigMapName,
				Namespace: constants.SystemNamespace,
			},
			Data: map[string]string{},
		})

		patches := gomonkey.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
			return fakeClient
		})
		defer patches.Reset()

		config := &v1alpha1.CloudCoreConfig{
			CommonConfig: &v1alpha1.CommonConfig{
				TunnelPort: 12345,
			},
		}
		updateCloudCoreConfigMap(config)

		// Assert exactly one get followed by one update action was executed
		actions := fakeClient.Actions()
		assert.Len(t, actions, 2)
		assert.Equal(t, "get", actions[0].GetVerb())
		assert.Equal(t, "update", actions[1].GetVerb())

		cm, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Get(context.TODO(), constants.CloudConfigMapName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Contains(t, cm.Data, "cloudcore.yaml")

		// Verify serialized YAML config value
		expectedBytes, err := yaml.Marshal(config)
		assert.NoError(t, err)
		assert.Equal(t, string(expectedBytes), cm.Data["cloudcore.yaml"])
	})

	t.Run("Fails to get ConfigMap", func(t *testing.T) {
		fakeClient := fakekube.NewSimpleClientset()
		patches := gomonkey.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
			return fakeClient
		})
		defer patches.Reset()

		config := &v1alpha1.CloudCoreConfig{}
		assert.NotPanics(t, func() {
			updateCloudCoreConfigMap(config)
		})

		// Assert that get was attempted but no update action was initiated
		actions := fakeClient.Actions()
		assert.Len(t, actions, 1)
		assert.Equal(t, "get", actions[0].GetVerb())
	})

	t.Run("Fails to update ConfigMap", func(t *testing.T) {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.CloudConfigMapName,
				Namespace: constants.SystemNamespace,
			},
			Data: map[string]string{},
		}
		fakeClient := fakekube.NewSimpleClientset(cm)
		fakeClient.PrependReactor("update", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.New("simulated update error")
		})

		patches := gomonkey.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
			return fakeClient
		})
		defer patches.Reset()

		config := &v1alpha1.CloudCoreConfig{}
		assert.NotPanics(t, func() {
			updateCloudCoreConfigMap(config)
		})

		// Assert both get and update actions were attempted
		actions := fakeClient.Actions()
		assert.Len(t, actions, 2)
		assert.Equal(t, "get", actions[0].GetVerb())
		assert.Equal(t, "update", actions[1].GetVerb())
	})

	t.Run("Successfully updates config map when Data is nil", func(t *testing.T) {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.CloudConfigMapName,
				Namespace: constants.SystemNamespace,
			},
			Data: nil,
		}
		fakeClient := fakekube.NewSimpleClientset(cm)

		patches := gomonkey.ApplyFunc(client.GetKubeClient, func() kubernetes.Interface {
			return fakeClient
		})
		defer patches.Reset()

		config := &v1alpha1.CloudCoreConfig{}
		assert.NotPanics(t, func() {
			updateCloudCoreConfigMap(config)
		})

		// Assert both get and update actions were executed
		actions := fakeClient.Actions()
		assert.Len(t, actions, 2)
		assert.Equal(t, "get", actions[0].GetVerb())
		assert.Equal(t, "update", actions[1].GetVerb())

		updatedCM, err := fakeClient.CoreV1().ConfigMaps(constants.SystemNamespace).Get(context.TODO(), constants.CloudConfigMapName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, updatedCM.Data)
		assert.Contains(t, updatedCM.Data, "cloudcore.yaml")
	})
}
