/*
Copyright 2025 The KubeEdge Authors.

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

package handlerfactory

import (
	"encoding/json"
	"net/http"

	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
)

func (f *Factory) Exec(request *request.RequestInfo) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		queryparams := req.URL.Query()

		commands := queryparams["command"]
		container := queryparams.Get("container")
		stdinStr := queryparams.Get("stdin")
		stdoutStr := queryparams.Get("stdout")
		stderrStr := queryparams.Get("stderr")
		ttyStr := queryparams.Get("tty")

		stdin := stdinStr == common.TrueStr
		stdout := stdoutStr == common.TrueStr
		stderr := stderrStr == common.TrueStr
		tty := ttyStr == common.TrueStr

		execInfo := common.ExecInfo{
			Namespace: request.Namespace,
			PodName:   request.Name,
			Container: container,
			Commands:  commands,
			Stdin:     stdin,
			Stdout:    stdout,
			Stderr:    stderr,
			TTY:       tty,
		}

		execResponse, handler := f.storage.Exec(req.Context(), execInfo)

		if handler != nil {
			handler.ServeHTTP(w, req)
		} else {
			execResBytes, err := json.Marshal(execResponse)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(execResBytes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	})
	return h
}
