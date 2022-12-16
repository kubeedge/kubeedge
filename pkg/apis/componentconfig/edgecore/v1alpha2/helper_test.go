/*
Copyright 2022 The KubeEdge Authors.

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

package v1alpha2

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEdgeCoreConfig_Parse(t *testing.T) {
	// unmarshal
	f, _ := os.CreateTemp(os.TempDir(), "config.yaml")
	_, _ = f.WriteString("aaa")
	// marshal
	conf := EdgeCoreConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: "test",
		},
	}
	out, _ := yaml.Marshal(conf)
	f2, _ := os.CreateTemp(os.TempDir(), "config2.yaml")
	_, _ = f2.WriteString(string(out))
	defer func() {
		f.Close()
		f2.Close()
		os.Remove(f.Name())
		os.Remove(f2.Name())
	}()

	tests := []struct {
		name         string
		TypeMeta     metav1.TypeMeta
		DataBase     *DataBase
		Modules      *Modules
		FeatureGates map[string]bool
		filename     string
		wantErr      bool
	}{
		{
			name:     "base",
			filename: "notexist",
			wantErr:  true,
		},
		{
			name:     "file exist but cannot unmarshal content",
			filename: f.Name(),
			wantErr:  true,
		},
		{
			name:     "file exist and can marshal properly",
			TypeMeta: metav1.TypeMeta{Kind: "test"},
			filename: f2.Name(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &EdgeCoreConfig{
				TypeMeta:     tt.TypeMeta,
				DataBase:     tt.DataBase,
				Modules:      tt.Modules,
				FeatureGates: tt.FeatureGates,
			}
			if err := c.Parse(tt.filename); (err != nil) != tt.wantErr {
				t.Errorf("EdgeCoreConfig.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
