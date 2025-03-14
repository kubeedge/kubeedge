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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
)

func (f *Factory) Logs(request *request.RequestInfo) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		queryParams := req.URL.Query()

		containerName := queryParams.Get("container")
		follow := queryParams.Get("follow")
		tailLines := queryParams.Get("tailLines")
		insecureSkipTLSVerifyBackend := queryParams.Get("insecureSkipTLSVerifyBackend")
		limitBytes := queryParams.Get("limitBytes")
		pretty := queryParams.Get("pretty")
		sinceSeconds := queryParams.Get("sinceSeconds")
		timestamps := queryParams.Get("timestamps")

		logsInfo := common.LogsInfo{
			PodName:                      request.Name,
			Namespace:                    request.Namespace,
			ContainerName:                containerName,
			Follow:                       follow,
			TailLines:                    tailLines,
			InsecureSkipTLSVerifyBackend: insecureSkipTLSVerifyBackend,
			LimitBytes:                   limitBytes,
			Pretty:                       pretty,
			SinceSeconds:                 sinceSeconds,
			Timestamps:                   timestamps,
		}

		logsResponse, res := f.storage.Logs(req.Context(), logsInfo)
		if res == nil {
			http.Error(w, "Failed to get logs from edged", http.StatusInternalServerError)
			return
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("Unexpected status code from edged: %d", res.StatusCode), http.StatusInternalServerError)
			return
		}

		if logsInfo.Follow != "true" {
			buf := &bytes.Buffer{}
			if _, err := io.Copy(buf, res.Body); err != nil {
				errMessage := fmt.Sprintf("failed to read logs with err:%v\n", err)
				klog.Warningf("[metaserver/logs] %v", errMessage)
				logsResponse.ErrMessages = append(logsResponse.ErrMessages, errMessage)
			}

			logsResponse.LogMessages = append(logsResponse.LogMessages, buf.String())

			logsResBytes, err := json.Marshal(logsResponse)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err = w.Write(logsResBytes)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// Logs are streamed
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Transfer-Encoding", "chunked")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming not supported", http.StatusInternalServerError)
				return
			}

			scanner := bufio.NewScanner(res.Body)
			for scanner.Scan() {
				fmt.Fprintln(w, scanner.Text())
				flusher.Flush()
			}

			if err := scanner.Err(); err != nil {
				http.Error(w, fmt.Sprintf("Error reading logs: %v", err), http.StatusInternalServerError)
				return
			}
		}
	})
	return h
}
