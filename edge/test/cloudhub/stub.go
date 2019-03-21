package test

import (
	"encoding/json"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	//	"github.com/kubeedge/kubeedge/edge/pkg/metamanager"
	"io"
	"io/ioutil"
	"net/http"

	"k8s.io/api/core/v1"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func init() {
	core.Register(&stubCloudHub{})
}

type Attributes struct {
	RoleName  string `json:"iam_role"`
	ProjectID string `json:"project_id"`
}

type record struct {
	Data          string `json:"data"`
	Partition_key string `json:"partition_key"`
}

type stubCloudHub struct {
	context *context.Context
	wsConn  *websocket.Conn
}

func (*stubCloudHub) Name() string {
	return "stubCloudHub"
}

func (*stubCloudHub) Group() string {
	//return core.MetaGroup
	return modules.MetaGroup
}

func (tm *stubCloudHub) eventReadLoop(conn *websocket.Conn, stop chan bool) {
	for {
		var event interface{}
		err := conn.ReadJSON(&event)
		if err != nil {
			log.LOGGER.Errorf("read error, connection will be closed: %v", err)
			stop <- true
			return
		}
		log.LOGGER.Infof("cloud hub receive message %+v", event)
	}
}

func (tm *stubCloudHub) serveEvent(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.LOGGER.Errorf("fail to build websocket connection: %v", err)
		http.Error(w, "fail to upgrade to websocket protocol", http.StatusInternalServerError)
		return
	}
	tm.wsConn = conn
	stop := make(chan bool, 1)
	log.LOGGER.Info("edge connected")
	go tm.eventReadLoop(conn, stop)
	<-stop
	tm.wsConn = nil
	log.LOGGER.Info("edge disconnected")
}

func (tm *stubCloudHub) podHandler(w http.ResponseWriter, req *http.Request) {
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.LOGGER.Errorf("read body error %v", err)
			w.Write([]byte("read request body error"))
			return
		}
		log.LOGGER.Infof("request body is %s\n", string(body))

		var pod v1.Pod
		if err = json.Unmarshal(body, &pod); err != nil {
			log.LOGGER.Errorf("unmarshal request body error %v", err)
			w.Write([]byte("unmarshal request body error"))
			return
		}
		var msgReq *model.Message
		switch req.Method {
		case "POST":
			msgReq = model.NewMessage("").BuildRouter("edgecontroller", "resource",
				"node/fake_node_id/pod/"+string(pod.UID), model.InsertOperation).FillBody(pod)
		case "DELETE":
			msgReq = model.NewMessage("").BuildRouter("edgecontroller", "resource",
				"node/fake_node_id/pod/"+string(pod.UID), model.DeleteOperation).FillBody(pod)
		}

		if tm.wsConn != nil {
			tm.wsConn.WriteJSON(*msgReq)
			log.LOGGER.Infof("send message to edgehub is %+v\n", *msgReq)
		}

		io.WriteString(w, "OK\n")
	}
}

func (tm *stubCloudHub) Start(c *context.Context) {
	tm.context = c
	defer tm.Cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/{group_id}/events", tm.serveEvent) // for edge-hub
	router.HandleFunc("/pod", tm.podHandler)               // for pod test

	s := http.Server{
		Addr:    "127.0.0.1:20000",
		Handler: router,
	}
	log.LOGGER.Info("Start cloud hub service")
	err := s.ListenAndServe()
	if err != nil {
		log.LOGGER.Errorf("ListenAndServe: %v", err)
	}

}

func (tm *stubCloudHub) Cleanup() {
	tm.context.Cleanup(tm.Name())
}
