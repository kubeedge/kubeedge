package handlerfactory

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func updateEdgeDevice() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
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
