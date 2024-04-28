package handlerfactory

import (
	"encoding/json"
	"net/http"

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
