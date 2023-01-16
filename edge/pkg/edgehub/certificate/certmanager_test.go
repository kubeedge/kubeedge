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

package certificate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
)

func init() {
	_, err := os.Stat("/tmp/edge.crt")
	if err != nil {
		err := util.GenerateTestCertificate("/tmp/", "edge", "edge")

		if err != nil {
			fmt.Printf("Failed to create certificate: %v\n", err)
		}
	}
}

func TestValidateCACerts(t *testing.T) {
	cacert, err := os.ReadFile("/tmp/edge.crt")
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}
	digest := sha256.Sum256(cacert)
	hash := hex.EncodeToString(digest[:])

	tests := []struct {
		cacert []byte
		hash   string
		want   bool
		ttName string
	}{
		{
			cacert: make([]byte, 0),
			hash:   "",
			want:   true,
			ttName: "empty cacert and empty hash",
		},
		{
			cacert: cacert,
			hash:   hash,
			want:   true,
			ttName: "valid cacert and hash",
		},
		{
			cacert: cacert,
			hash:   "invalid",
			want:   false,
			ttName: "invalid hash",
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, _, _ := ValidateCACerts(tt.cacert, tt.hash)
			if got != tt.want {
				t.Errorf("ValidateCACerts = %v, want %v", got, tt.want)
			}
		})
	}
}
