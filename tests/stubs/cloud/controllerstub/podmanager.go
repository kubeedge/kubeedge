/*
Copyright 2019 The KubeEdge Authors.

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

package controllerstub

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/tests/stubs/common/constants"
	"github.com/kubeedge/kubeedge/tests/stubs/common/types"
	"github.com/kubeedge/kubeedge/tests/stubs/common/utils"
)

// NewPodManager creates pod manger
func NewPodManager() (*PodManager, error) {
	event := make(chan *model.Message, 1024)
	pm := &PodManager{event: event}
	return pm, nil
}

// PodManager is a manager watch pod change event
type PodManager struct {
	// event
	event chan *model.Message
	// pods map
	pods sync.Map
}

// GetEvent return a channel which receives event
func (pm *PodManager) GetEvent() chan *model.Message {
	return pm.event
}

// AddPod adds pod in cache
func (pm *PodManager) AddPod(k string, v types.FakePod) {
	pm.pods.Store(k, v)
}

// DeletePod deletes pod in cache
func (pm *PodManager) DeletePod(k string) {
	pm.pods.Delete(k)
}

// UpdatePodStatus update pod status in cache
func (pm *PodManager) UpdatePodStatus(k string, s string) {
	v, ok := pm.pods.Load(k)
	if ok {
		pod := v.(types.FakePod)
		// Status becomes running in the first time
		if pod.Status != s && s == constants.PodRunning {
			pod.RunningTime = time.Now().UnixNano()
		}
		pod.Status = s
		pm.pods.Store(k, pod)
	}
}

// GetPod gets pod from cache
func (pm *PodManager) GetPod(key string) types.FakePod {
	v, ok := pm.pods.Load(key)
	if ok {
		return v.(types.FakePod)
	} else {
		return types.FakePod{}
	}
}

// ListPods lists all pods in cache
func (pm *PodManager) ListPods() []types.FakePod {
	pods := make([]types.FakePod, 0)
	pm.pods.Range(func(k, v interface{}) bool {
		pods = append(pods, v.(types.FakePod))
		return true
	})
	return pods
}

// PodHandlerFunc is used to receive and process message
func (pm *PodManager) PodHandlerFunc(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// List Pod
		log.LOGGER.Debugf("Receive list pod request")
		pods := pm.ListPods()
		log.LOGGER.Debugf("Current pods number: %v", len(pods))
		rspBodyBytes := new(bytes.Buffer)
		json.NewEncoder(rspBodyBytes).Encode(pods)
		w.Write(rspBodyBytes.Bytes())
		log.LOGGER.Debugf("Finish list pod request")
	case http.MethodPost:
		log.LOGGER.Debugf("Receive add pod request")
		var p types.FakePod
		// Get request body
		if req.Body != nil {
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				log.LOGGER.Errorf("Read body error %v", err)
				w.Write([]byte("Read request body error"))
				return
			}
			log.LOGGER.Debugf("Request body is %s", string(body))
			if err = json.Unmarshal(body, &p); err != nil {
				log.LOGGER.Errorf("Unmarshal request body error %v", err)
				w.Write([]byte("Unmarshal request body error"))
				return
			}
		}
		// Add Pod
		ns := constants.NamespaceDefault
		if p.Namespace != "" {
			ns = p.Namespace
		}

		// Build Add message
		msg := model.NewMessage("")
		resource, err := utils.BuildResource(p.NodeName, p.Namespace, model.ResourceTypePod, p.Name)
		if err != nil {
			log.LOGGER.Errorf("Build message resource failed with error: %s", err)
			w.Write([]byte("Build message resource failed with error"))
			return
		}
		msg.Content = p
		msg.BuildRouter(constants.ControllerStub, constants.GroupResource, resource, model.InsertOperation)

		// Add pod in cache
		p.CreateTime = time.Now().UnixNano()
		pm.AddPod(ns+"/"+p.Name, p)

		// Send msg
		select {
		case pm.event <- msg:
			log.LOGGER.Debugf("Finish add pod request")
		}
	case http.MethodDelete:
		// Delete Pod
		log.LOGGER.Debugf("Receive delete pod request")
		params := req.URL.Query()
		ns := params.Get("namespace")
		if ns == "" {
			ns = constants.NamespaceDefault
		}
		nodename := params.Get("nodename")
		name := params.Get("name")
		log.LOGGER.Debugf("Pod Namespace: %s NodeName: %s Name: %s", ns, nodename, name)

		// Build delete message
		msg := model.NewMessage("")
		resource, err := utils.BuildResource(nodename, ns, model.ResourceTypePod, name)
		if err != nil {
			log.LOGGER.Errorf("Build message resource failed with error: %s", err)
			w.Write([]byte("Build message resource failed with error"))
			return
		}
		msg.Content = pm.GetPod(ns + "/" + name)
		msg.BuildRouter(constants.ControllerStub, constants.GroupResource, resource, model.DeleteOperation)

		// Delete pod in cache
		pm.DeletePod(ns + "/" + name)

		// Send msg
		select {
		case pm.event <- msg:
			log.LOGGER.Debugf("Finish delete pod request")
		}

	default:
		log.LOGGER.Errorf("Http type: %s unsupported", req.Method)
	}
}
