package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/metamanager/dao"
	"k8s.io/api/core/v1"
)

const (
	name = "testManager"
)

func init() {
	core.Register(&testManager{})
}

type testManager struct {
	context    *context.Context
	moduleWait *sync.WaitGroup
}

type meta struct {
	UID string `json:"uid"`
}

func (tm *testManager) Name() string {
	return name
}

func (tm *testManager) Group() string {
	return core.MetaGroup
}

//Function to handle device addition and removal from the edgenode
func (tm *testManager) deviceHandler(w http.ResponseWriter, req *http.Request) {
	var operation string
	var Content interface{}

	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.LOGGER.Errorf("read body error %v", err)
			w.Write([]byte("read request body error"))
		}
		log.LOGGER.Infof("request body is %s\n", string(body))
		err = json.Unmarshal(body, &Content)
		if err != nil {
			log.LOGGER.Errorf("unmarshal request body error %v", err)
			w.Write([]byte("unmarshal request body error"))
		}
		switch req.Method {
		case "POST":
			operation = model.InsertOperation
		case "DELETE":
			operation = model.DeleteOperation
		case "PUT":
			operation = model.UpdateOperation
		}
		msgReq := message.BuildMsg("edgehub", "", "edgemgr", "membership", operation, Content)
		tm.context.Send("twin", *msgReq)
		log.LOGGER.Infof("send message to twingrp is %+v\n", msgReq)
	}
}

func (tm *testManager) secretHandler(w http.ResponseWriter, req *http.Request) {
	var operation string
	var p v1.Secret
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.LOGGER.Errorf("read body error %v", err)
			w.Write([]byte("read request body error"))
		}
		log.LOGGER.Infof("request body is %s\n", string(body))
		if err = json.Unmarshal(body, &p); err != nil {
			log.LOGGER.Errorf("unmarshal request body error %v", err)
			w.Write([]byte("unmarshal request body error"))
		}

		switch req.Method {
		case "POST":
			operation = model.InsertOperation
		case "DELETE":
			operation = model.DeleteOperation
		case "PUT":
			operation = model.UpdateOperation
		}

		msgReq := message.BuildMsg("edgehub", string(p.UID), "test", "fakeNamespace/secret/"+string(p.UID), operation, p)
		tm.context.Send("metaManager", *msgReq)
		log.LOGGER.Infof("send message to metaManager is %+v\n", msgReq)
	}
}

func (tm *testManager) configmapHandler(w http.ResponseWriter, req *http.Request) {
	var operation string
	var p v1.ConfigMap
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.LOGGER.Errorf("read body error %v", err)
			w.Write([]byte("read request body error"))
		}
		log.LOGGER.Infof("request body is %s\n", string(body))
		if err = json.Unmarshal(body, &p); err != nil {
			log.LOGGER.Errorf("unmarshal request body error %v", err)
			w.Write([]byte("unmarshal request body error"))
		}

		switch req.Method {
		case "POST":
			operation = model.InsertOperation
		case "DELETE":
			operation = model.DeleteOperation
		case "PUT":
			operation = model.UpdateOperation
		}

		msgReq := message.BuildMsg("edgehub", string(p.UID), "test", "fakeNamespace/configmap/"+string(p.UID), operation, p)
		tm.context.Send("metaManager", *msgReq)
		log.LOGGER.Infof("send message to metaManager is %+v\n", msgReq)
	}
}

func (tm *testManager) getPodsHandler(w http.ResponseWriter, r *http.Request) {
	var podList v1.PodList
	metas, err := dao.QueryMeta("type", "pod")
	if err != nil {
		log.LOGGER.Errorf("failed to query pods: %v", err)
	}
	for _, podContent := range *metas {
		var pod v1.Pod
		err := json.Unmarshal([]byte(podContent), &pod)
		if err != nil {
			log.LOGGER.Errorf("failed to unmarshal: %v", err)
		}
		podList.Items = append(podList.Items, pod)
	}
	respBody, err := json.Marshal(podList)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func (tm *testManager) podHandler(w http.ResponseWriter, req *http.Request) {
	var operation string
	var p v1.Pod
	if req.Method == http.MethodGet {
		tm.getPodsHandler(w, req)
	} else if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.LOGGER.Errorf("read body error %v", err)
			w.Write([]byte("read request body error"))
		}
		log.LOGGER.Infof("request body is %s\n", string(body))
		if err = json.Unmarshal(body, &p); err != nil {
			log.LOGGER.Errorf("unmarshal request body error %v", err)
			w.Write([]byte("unmarshal request body error"))
		}

		switch req.Method {
		case "POST":
			operation = model.InsertOperation
		case "DELETE":
			operation = model.DeleteOperation
		case "PUT":
			operation = model.UpdateOperation
		}

		ns := v1.NamespaceDefault
		if p.Namespace != "" {
			ns = p.Namespace
		}
		msgReq := message.BuildMsg("edgehub", string(p.UID), "test", ns+"/pod/"+string(p.UID), operation, p)
		tm.context.Send("metaManager", *msgReq)
		log.LOGGER.Infof("send message to metaManager is %+v\n", msgReq)
	}
}

func (tm *testManager) Start(c *context.Context) {
	tm.context = c
	defer tm.Cleanup()

	http.HandleFunc("/pod", tm.podHandler)
	http.HandleFunc("/configmap", tm.configmapHandler)
	http.HandleFunc("/secret", tm.secretHandler)
	http.HandleFunc("/devices", tm.deviceHandler)
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.LOGGER.Errorf("ListenAndServe: %v", err)
	}
}

func (tm *testManager) Cleanup() {
	tm.context.Cleanup(tm.Name())
}
