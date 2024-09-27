package handlerfactory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"k8s.io/klog/v2"

	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/pkg/version"
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

func (f *Factory) ConfirmUpgrade(_ /*edgeappName*/ string) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		klog.Infof("Begin to run upgrade command")
		// TODO: How to get options...
		var upgradeReq commontypes.NodeUpgradeJobRequest
		var configFile string
		upgradeCmd := fmt.Sprintf("keadm upgrade edge --upgradeID %s --historyID %s --fromVersion %s --toVersion %s --config %s --image %s > /tmp/keadm.log 2>&1",
			upgradeReq.UpgradeID, upgradeReq.HistoryID, version.Get(), upgradeReq.Version, configFile, upgradeReq.Image)

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
		klog.Infof("!!! Finish upgrade from Version %s to %s ...", version.Get(), upgradeReq.Version)
		// TODO: How to proceed backup and rollback ...
	})
	return h
}
