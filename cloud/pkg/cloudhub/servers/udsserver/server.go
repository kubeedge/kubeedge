package udsserver

import (
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	hubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/common/constants"
)

// StartServer serves
func StartServer(address string) {
	uds := NewUnixDomainSocket(address)
	uds.SetContextHandler(func(context string) string {
		// receive message from client
		klog.Infof("uds server receives context: %s", context)
		msg, err := ExtractMessage(context)
		if err != nil {
			klog.Errorf("Failed to extract message: %v", err)
			return feedbackError(err, msg)
		}

		// Send message to edge
		resp, err := beehiveContext.SendSync(hubmodel.SrcCloudHub, *msg, constants.CSISyncMsgRespTimeout)
		if err != nil {
			klog.Errorf("failed to send message to edge: %v", err)
			return feedbackError(err, msg)
		}
		// Marshal response message
		data, err := json.Marshal(resp)
		if err != nil {
			klog.Errorf("marshal response failed with error: %v", err)
			return feedbackError(err, msg)
		}
		klog.Infof("uds server send back data: %s resp: %v", string(data), resp)
		return string(data)
	})

	klog.Info("start unix domain socket server")
	if err := uds.StartServer(); err != nil {
		klog.Fatalf("failed to start uds server: %v", err)
		return
	}
}

// ExtractMessage extracts message from clients
func ExtractMessage(context string) (*model.Message, error) {
	var msg *model.Message
	if context != "" {
		err := json.Unmarshal([]byte(context), &msg)
		if err != nil {
			return nil, err
		}
	} else {
		err := errors.New("failed with error: context is empty")
		klog.Errorf("%v", err)
		return nil, err
	}
	return msg, nil
}

// feedbackError sends back error message
func feedbackError(err error, request *model.Message) string {
	// Build message
	errResponse := model.NewErrorMessage(request, err.Error()).SetRoute(hubmodel.SrcCloudHub, request.GetGroup())
	// Marshal message
	data, err := json.Marshal(errResponse)
	if err != nil {
		return fmt.Sprintf("feedbackError marshal failed with error: %v", err)
	}
	return string(data)
}
