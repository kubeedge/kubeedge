package socket

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/common"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/beehive/pkg/core/socket/config"
	"github.com/kubeedge/beehive/pkg/core/socket/wrapper"
)

const (
	acceptPeriod = 5 * time.Second
)

func init() {
	configList, err := config.GetServerSocketConfig()
	if err != nil {
		klog.Errorf("failed to get server socket config, error: %+v", err)
		return
	}

	for _, sConfig := range configList {
		core.Register(&Server{
			enable:     true,
			name:       sConfig.ModuleName,
			address:    sConfig.Address,
			buffSize:   uint64(sConfig.BufferSize),
			socketType: sConfig.SocketType,
			connMax:    sConfig.ConnNumberMax,
			pipeKeeper: make(chan struct{}, sConfig.ConnNumberMax),
			stopChan:   make(chan struct{}),
		})
	}
}

func (m *Server) serveSocket() error {
	if strings.Contains(m.socketType, "unix") {
		err := os.Remove(m.address)
		if err != nil {
			klog.Errorf("failed to remove address, error: %+v", err)
		}
	}

	listener, err := net.Listen(m.socketType, m.address)
	//defer listener.Close()
	if err != nil {
		klog.Errorf("failed to listen to socket, error: %+v", err)
		return fmt.Errorf("failed to listen to socket, error: %+v", err)
	}
	m.listener = listener
	klog.Infof("Listening on addr: %s", listener.Addr().String())

	if _, err := os.Stat(m.address); err == nil {
		err = os.Chmod(m.address, 0600)
		if err != nil {
			klog.Errorf("Chmod failed: %s", m.address)
			return err
		}
	}

	for {
		conn, err := listener.Accept()
		klog.Infof("Connected from %s", conn.LocalAddr().String())
		if err != nil {
			klog.Errorf("failed to accept with error %+v", err)
			return fmt.Errorf("failed to accept, error: %+v", err)
		}

		select {
		case m.pipeKeeper <- struct{}{}:
			go m.handleServerConn(conn)
		default:
			klog.Warningf("reject remote, because of connection exceed the max value: %d", m.connMax)
			err := conn.Close()
			if err != nil {
				klog.Errorf("conn closed with error %+v", err)
			}
			time.Sleep(acceptPeriod)
		}
	}
}

// HandleServerConn handler sever
func (m *Server) handleServerConn(c net.Conn) {
	conn := wrapper.NewWrapper(m.socketType, c, int(m.buffSize))

	// close connectinon
	// release pipe keeper
	defer func() {
		err := conn.Close()
		<-m.pipeKeeper
		if err != nil {
			return
		}
	}()

	for {
		err := m.handleServerMessage(conn)
		if err != nil {
			klog.Errorf("failed to handle server message with error %+v", err)
			return
		}
	}
}

func (m *Server) processModuleMessage(conn wrapper.Conn, message *model.Message) error {
	switch message.GetOperation() {
	case common.OperationTypeModule:
		if !IsModuleEnabled(message.GetSource()) {
			return fmt.Errorf("this module is not enabled, message: %s", message.String())
		}
		moduleName := message.GetSource()
		moduleGroup := message.GetSource()

		add := &common.ModuleInfo{
			ModuleName: moduleName,
			ModuleType: m.socketType,
			ModuleSocket: common.ModuleSocket{
				IsRemote:   false,
				Connection: conn,
			},
		}

		beehiveContext.AddModule(add)
		beehiveContext.AddModuleGroup(moduleName, moduleGroup)
		resp := message.NewRespByMessage(message, core.GetModuleExchange())
		beehiveContext.SendResp(*resp)
	}
	return nil
}

// HandleServerContext handler ctx
func (m *Server) handleServerMessage(conn wrapper.Conn) error {
	var message model.Message
	err := conn.ReadJSON(&message)
	if err != nil {
		klog.Errorf("failed to read json with error %+v", err)
		return fmt.Errorf("failed to read json, error:%+v", err)
	}

	// log.LOGGER.Infof("receive  message: %+v", message)
	switch message.GetResource() {
	case common.ResourceTypeModule:
		// remote module message
		return m.processModuleMessage(conn, &message)
	}

	if !IsModuleEnabled(message.GetSource()) {
		klog.Warning("the module is not enabled, just discard")
		return fmt.Errorf("the module is not enabled, just discard")
	}

	// transmit the message
	// log.LOGGER.Infof("server: %+v", message)
	beehiveContext.Send(message.GetDestination(), message)
	return nil
}

// StartServer start server
func (m *Server) startServer() {
	err := m.serveSocket()
	if err != nil {
		klog.Errorf("failed to start server")
	}
}

// StopServer start server
func (m *Server) stopServer() {
	err := m.listener.Close()
	if err != nil {
		klog.Errorf("failed to close server with error %+v", err)
	}
}

func IsModuleEnabled(m string) bool {
	_, err := config.GetClientSocketConfig(m)
	return err == nil
}
