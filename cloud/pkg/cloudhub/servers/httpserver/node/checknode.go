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

package node

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
)

func CheckNode(request *restful.Request, response *restful.Response) {
	// get nodeName
	nodeName := request.PathParameter("nodename")
	if nodeName == "" {
		err := response.WriteErrorString(http.StatusBadRequest, "nodename parameter is required")
		if err != nil {
			klog.Errorf("failed to send the task resp to edge , err: %v", err)
		}
		return
	}

	_, err := client.GetKubeClient().CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// node not found or query error
			err = response.WriteErrorString(http.StatusNotFound, "Node not found")
			if err != nil {
				klog.Errorf("failed to send the task resp to edge , err: %v", err)
			}
			return
		}

		// Other errors
		klog.Errorf("failed to query the node, err: %v", err)
		err = response.WriteErrorString(http.StatusInternalServerError, "Failed to query node information")
		if err != nil {
			klog.Errorf("failed to send the response to edge, err: %v", err)
		}
		return
	}

	// node exists return success
	err = response.WriteErrorString(http.StatusOK, "Node founded")
	if err != nil {
		klog.Errorf("failed to send the task resp to edge , err: %v", err)
	}
}
