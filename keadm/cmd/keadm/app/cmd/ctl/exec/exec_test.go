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

package exec

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func TestExecPodResponse(t *testing.T) {
	for _, test := range []struct {
		name     string
		command  string
		response string
		output   []string
		wantErr  string
	}{
		{
			name:     "successful command",
			command:  "true",
			response: `{"runOutMessages":["hello"]}`,
			output:   []string{"hello"},
		},
		{
			name:     "successful command with stderr",
			command:  "warn",
			response: `{"runErrMessages":["warning"]}`,
			output:   []string{"warning"},
		},
		{
			name:     "failed command",
			command:  "false",
			response: `{"runOutMessages":["partial output"],"runErrMessages":["remote stderr"],"errMessages":["command exited with status 1","execution failed"]}`,
			output:   []string{"partial output", "remote stderr"},
			wantErr:  "command exited with status 1\nexecution failed",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/namespaces/default/pods/pod/exec", r.URL.Path)
				assert.Equal(t, []string{test.command}, r.URL.Query()["command"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, test.response)
			}))
			t.Cleanup(server.Close)

			client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
			require.NoError(t, err)

			var output bytes.Buffer
			state := klog.CaptureState()
			t.Cleanup(state.Restore)
			klog.LogToStderr(false)
			klog.SetOutput(&output)

			opts := PodExecOptions{Namespace: "default"}
			err = opts.execPod(context.Background(), client, "pod", []string{test.command})
			if test.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.wantErr)
			}

			for _, message := range test.output {
				assert.Contains(t, output.String(), message)
			}
		})
	}
}
