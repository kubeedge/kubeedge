package clouddatastream

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/kubeedge/kubeedge/cloud/pkg/clouddatastream/config"
	"k8s.io/klog/v2"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type VideoSession struct {
	Session     *Session
	VideoConnID uint64
	sync.Mutex
}

type StreamServer struct {
	nextMessageID uint64
	container     *restful.Container
	tunnel        *TunnelServer
	sync.Mutex
	videoSessions map[string]*VideoSession
	upgrader      websocket.Upgrader
}

func newStreamServer(t *TunnelServer) *StreamServer {
	return &StreamServer{
		container:     restful.NewContainer(),
		tunnel:        t,
		videoSessions: make(map[string]*VideoSession),
		upgrader: websocket.Upgrader{
			HandshakeTimeout: time.Second * 2,
			ReadBufferSize:   1024,
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				w.WriteHeader(status)
				_, err := w.Write([]byte(reason.Error()))
				if err != nil {
					klog.Errorf("failed to write http response, err: %v", err)
				}
			},
		},
	}
}

func (s *StreamServer) installDebugHandler() {
	ws := new(restful.WebService)
	ws.Path("/video/{endpointName}")
	ws.Route(ws.GET("").
		To(s.getRTSP))
	s.container.Add(ws)

}

func (s *StreamServer) addVideoSession(id string, session *Session, videoConnID uint64) {
	s.Lock()
	s.videoSessions[id] = &VideoSession{
		Session:     session,
		VideoConnID: videoConnID,
	}
	s.Unlock()
}

func (s *StreamServer) getVideoSession(id string) (*VideoSession, bool) {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.videoSessions[id]
	return sess, ok
}

func (s *StreamServer) deleteVideoSession(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.videoSessions, id)
}

func (s *StreamServer) getRTSP(r *restful.Request, w *restful.Response) {
	ep := r.PathParameter("endpointName")

	var err error
	var session *Session
	var videoConnID uint64

	wsconn, err := upgrader.Upgrade(w.ResponseWriter, r.Request, nil)
	if err != nil {
		klog.Errorf("upgrade error: %v", err)
		return
	}
	defer wsconn.Close()

	if sess, ok := s.tunnel.getSession(ep); !ok {
		s.deleteVideoSession(ep)
		return
	} else if sess.IsTunnelClosed() {
		s.tunnel.delSession(ep)
		s.deleteVideoSession(ep)
		return
	}

	if sess, ok := s.getVideoSession(ep); ok {
		// rtsp流已经在传输
		session = sess.Session
		videoConnID = sess.VideoConnID
		conn, err := session.GetAPIServerConnection(videoConnID)
		if err != nil {
			klog.Errorf("Get apiserver connection %v from %s error %v", videoConnID, session.String(), err)
			return
		}

		// 使用类型断言
		if videoConn, ok := conn.(*ContainerRTSPConnection); ok {
			videoConn.AddWSConn(wsconn)
		} else {
			klog.Warningf("APIServerConnection is not a ContainerRTSPConnection, got %T", conn)
		}

		<-r.Request.Context().Done() // 客户端断开
		klog.Infof("client closed connection.")

	} else {
		videoSession, _ := s.tunnel.getSession(ep)

		videoConnection, err := videoSession.AddAPIServerConnection(s, &ContainerRTSPConnection{
			req:          r,
			wsConns:      []*websocket.Conn{wsconn},
			session:      videoSession,
			ctx:          r.Request.Context(),
			edgePeerStop: make(chan struct{}),
			closeChan:    make(chan bool),
			emptyChan:    make(chan struct{}, 1),
		})
		if err != nil {
			err = fmt.Errorf("add apiServer connection into %s error %v", videoSession.String(), err)
			return
		}

		s.addVideoSession(ep, videoSession, videoConnection.GetMessageID())

		defer func() {
			if err != nil {
				videoSession.DeleteAPIServerConnection(videoConnection)
				videoSession.Close()
				s.deleteVideoSession(ep)
			}
		}()
		go func() {
			if err := videoConnection.Serve(); err != nil {
				klog.Errorf("[rtspvideo] Serve error: %v", err)
				videoSession.DeleteAPIServerConnection(videoConnection)
				videoSession.Close()
				s.deleteVideoSession(ep)
				return
			}
		}()

		<-videoConnection.(*ContainerRTSPConnection).emptyChan
		// 没有人在听对应视频流，清理 session
		klog.Infof("[rtspvideo] no listeners left, deleting videoSession %s", ep)

		s.deleteVideoSession(ep)
		return
	}

}

func (s *StreamServer) Start() {
	s.installDebugHandler()

	pool := x509.NewCertPool()
	data, err := os.ReadFile(config.Config.TLSStreamCAFile)
	if err != nil {
		klog.Errorf("Read tls stream ca file error %v", err)
		return
	}
	pool.AppendCertsFromPEM(data)

	streamServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Config.StreamPort),
		Handler: s.container,
		TLSConfig: &tls.Config{
			ClientCAs:  pool,
			ClientAuth: tls.RequestClientCert,
			MinVersion: tls.VersionTLS12,
		},
	}
	klog.Infof("Prepare to start stream server ...")
	err = streamServer.ListenAndServeTLS(config.Config.TLSStreamCertFile, config.Config.TLSStreamPrivateKeyFile)
	if err != nil {
		klog.Errorf("Start stream server error %v", err)
		return
	}
}
