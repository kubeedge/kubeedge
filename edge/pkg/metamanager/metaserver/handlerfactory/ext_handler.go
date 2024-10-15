package handlerfactory

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
)

func (f *Factory) Restart(namespace string) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		podNameBytes, err := limitedReadBody(req, int64(3*1024*1024))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var podNames []string
		err = json.Unmarshal(podNameBytes, &podNames)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		restartInfo := common.RestartInfo{
			PodNames:  podNames,
			Namespace: namespace,
		}
		restartResponse := f.storage.Restart(req.Context(), restartInfo)
		restartResBytes, err := json.Marshal(restartResponse)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(restartResBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	return h
}

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

		if logsInfo.Follow == "" || logsInfo.Follow == "false" {
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
			if res.StatusCode != http.StatusOK {
				http.Error(w, fmt.Sprintf("Unexpected status code from edged: %d", res.StatusCode), http.StatusInternalServerError)
				return
			}

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
			}
		}
	})
	return h
}

func (f *Factory) Exec(request *request.RequestInfo) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		queryparams := req.URL.Query()

		commands := queryparams["command"]
		container := queryparams.Get("container")
		stdinStr := queryparams.Get("stdin")
		stdoutStr := queryparams.Get("stdout")
		stderrStr := queryparams.Get("stderr")
		ttyStr := queryparams.Get("tty")

		stdin := (stdinStr == "true")
		stdout := (stdoutStr == "true")
		stderr := (stderrStr == "true")
		tty := (ttyStr == "true")

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

func (f *Factory) UpdateEdgeDevice(request *request.RequestInfo) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var device *v1beta1.Device
		if err := json.Unmarshal(body, &device); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		source := modules.MetaManagerModuleName
		target := modules.DeviceTwinModuleName
		resourece := device.Namespace + "/device/updated"

		operation := model.UpdateOperation

		device.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   v1beta1.GroupName,
			Version: v1beta1.Version,
			Kind:    constants.KindTypeDevice,
		})
		modelMsg := model.NewMessage("").
			SetResourceVersion(device.ResourceVersion).
			FillBody(device)
		modelMsg.BuildRouter(source, target, resourece, operation)
		resp, err := beehiveContext.SendSync(source, *modelMsg, 1*time.Minute)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respData, err := resp.GetContentData()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(respData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	return h
}
