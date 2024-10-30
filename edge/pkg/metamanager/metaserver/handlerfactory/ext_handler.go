package handlerfactory

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/task/taskexecutor"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/upgradedb"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/pkg/version"
)

const (
	trueStr = "true"
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

func (f *Factory) ConfirmUpgrade() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		klog.Info("Begin to run upgrade command")
		opts := options.GetEdgeCoreOptions()
		var upgradeReq commontypes.NodeUpgradeJobRequest
		var nodeTaskReq types.NodeTaskRequest
		nodeTaskReq, _ = upgradedb.QueryNodeTaskRequestFromMetaV2()
		upgradeReq, _ = upgradedb.QueryNodeUpgradeJobRequestFromMetaV2()
		upgradeCmd := fmt.Sprintf("keadm upgrade edge --upgradeID %s --historyID %s --fromVersion %s --toVersion %s --config %s --image %s > /tmp/keadm.log 2>&1",
			upgradeReq.UpgradeID, upgradeReq.HistoryID, version.Get(), upgradeReq.Version, opts.ConfigFile, upgradeReq.Image)

		executor, _ := taskexecutor.GetExecutor(taskexecutor.TaskUpgrade)
		event, _ := executor.Do(nodeTaskReq)
		klog.Info("Confirm Upgrade:" + event.Type + "," + event.Msg)
		// run upgrade cmd to upgrade edge node
		// use nohup command to start a child progress
		command := fmt.Sprintf("nohup %s &", upgradeCmd)
		cmd := exec.Command("bash", "-c", command)
		s, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("run upgrade command %s failed: %v, res: %s", command, err, s),
				http.StatusInternalServerError)
			return
		}
		klog.Infof("Finish upgrade from Version %s to %s ...", version.Get(), upgradeReq.Version)
		err = upgradedb.DeleteNodeTaskRequestFromMetaV2()
		if err != nil {
			klog.Errorf("Failed to delete NodeTaskRequest%s", err.Error())
		}
		err = upgradedb.DeleteNodeUpgradeJobRequestFromMetaV2()
		if err != nil {
			klog.Errorf("Failed to delete NodeUpgradeJobRequest%s", err.Error())
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

func (f *Factory) Exec(request *request.RequestInfo) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		queryparams := req.URL.Query()

		commands := queryparams["command"]
		container := queryparams.Get("container")
		stdinStr := queryparams.Get("stdin")
		stdoutStr := queryparams.Get("stdout")
		stderrStr := queryparams.Get("stderr")
		ttyStr := queryparams.Get("tty")

		stdin := stdinStr == trueStr
		stdout := stdoutStr == trueStr
		stderr := stderrStr == trueStr
		tty := ttyStr == trueStr

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
