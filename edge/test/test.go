package test

import (
	"encoding/json"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"

	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"k8s.io/api/core/v1"
)

const (
	name            = "testManager"
	edgedEndPoint   = "http://127.0.0.1:10255"
	EdgedPodHandler = "/pods"
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
	//return core.MetaGroup
	return modules.MetaGroup
}

//Function to get the pods from Edged
func GetPodListFromEdged(w http.ResponseWriter) error {
	var pods v1.PodList
	var bytes io.Reader
	client := &http.Client{}
	t := time.Now()
	req, err := http.NewRequest(http.MethodGet, edgedEndPoint+EdgedPodHandler, bytes)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		log.LOGGER.Errorf("Frame HTTP request failed: %v", err)
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.LOGGER.Errorf("Sending HTTP request failed: %v", err)
		return err
	}
	log.LOGGER.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Now().Sub(t))
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.LOGGER.Errorf("HTTP Response reading has failed: %v", err)
		return err
	}
	err = json.Unmarshal(contents, &pods)
	if err != nil {
		log.LOGGER.Errorf("Json Unmarshal has failed: %v", err)
		return err
	}
	respBody, err := json.Marshal(pods)
	if err != nil {
		log.LOGGER.Errorf("Json Marshal failed: %v", err)
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)

	return nil
}

//Function to handle Get/Add/Delete deployment list.
func (tm *testManager) podHandler(w http.ResponseWriter, req *http.Request) {
	var operation string
	var p v1.Pod
	if req.Method == http.MethodGet {
		err := GetPodListFromEdged(w)
		if err != nil {
			log.LOGGER.Errorf("Get podlist from Edged has failed: %v", err)
		}
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
		msgReq := message.BuildMsg("resource", string(p.UID), "controller", ns+"/pod/"+string(p.Name), operation, p)
		tm.context.Send("metaManager", *msgReq)
		log.LOGGER.Infof("send message to metaManager is %+v\n", msgReq)
	}
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

func (tm *testManager) Start(c *context.Context) {
	tm.context = c
	defer tm.Cleanup()

	http.HandleFunc("/pods", tm.podHandler)
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
